package main

import (
	"context"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	rdb          atomic.Pointer[redis.Client] // 无锁读：请求路径 Load() 直接取 client
	redisEnabled atomic.Bool                  // 无锁读：请求路径 Load() 判断是否启用
	redisMu      sync.Mutex
	// redisLogCh 异步写入 channel：请求路径不再同步等 Pipeline 返回，支持万级并发
	redisLogCh   chan UsageLog
	redisLogOnce sync.Once
)

// redisReady 判断 Redis 统计是否可用（全部原子读，无锁竞争）
func redisReady() bool {
	return redisEnabled.Load() && rdb.Load() != nil && redisLogCh != nil
}

// dayAgg 按天聚合的统计单元（用于近 N 天窗口聚合）
type dayAgg struct {
	count, input, output, total                int64
	ttft, duration, cacheHit, cacheMiss        int64
}

// initRedis 初始化 Redis 连接（狂暴模式启用时调用）
func initRedis(addr, password string, db int) {
	redisMu.Lock()
	defer redisMu.Unlock()

	if addr == "" {
		log.Printf("[redis] 地址为空，狂暴模式未启用")
		return
	}

	if c := rdb.Load(); c != nil {
		c.Close()
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     50,
		MinIdleConns: 10,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	rdb.Store(client)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[redis] 连接失败: %v", err)
		rdb.Store(nil)
		redisEnabled.Store(false)
		return
	}

	redisEnabled.Store(true)
	// 启动异步 worker（仅一次）
	redisLogOnce.Do(func() {
		redisLogCh = make(chan UsageLog, 20000)
		go redisWorker()
	})
	applyRageModePool()
	log.Printf("[redis] 已连接（狂暴模式启用）: %s db=%d", addr, db)
}

// testRedisConnection 仅测试 Redis 连接是否可用，不改变当前狂暴模式状态
func testRedisConnection(addr, password string, db int) error {
	c := redis.NewClient(&redis.Options{
		Addr:        addr,
		Password:    password,
		DB:          db,
		PoolSize:    2,
		DialTimeout: 3 * time.Second,
		ReadTimeout: 2 * time.Second,
	})
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return c.Ping(ctx).Err()
}

// closeRedis 关闭 Redis 连接（狂暴模式关闭时调用）
// 顺序：先标记禁用（阻止新数据入 channel）-> 刷盘待写入统计 -> 关闭连接 -> 恢复普通连接池
func closeRedis() {
	redisMu.Lock()
	redisEnabled.Store(false) // 先标记禁用，redisIncrUsage 不再投递
	redisMu.Unlock()

	// 刷盘 channel 中待写入的统计（避免丢数据）
	flushRedisLogs()

	if c := rdb.Load(); c != nil {
		c.Close()
	}
	rdb.Store(nil)

	restoreNormalPool()
	log.Printf("[redis] 已断开（恢复普通模式）")
}

// redisIncrUsage 异步递增统计：仅向 channel 投递，不阻塞请求路径
// 由后台 redisWorker 批量执行 Pipeline，一次网络往返完成 N*16 条 HINCRBY
func redisIncrUsage(entry UsageLog) {
	if !redisReady() {
		return
	}
	select {
	case redisLogCh <- entry:
	default:
		// channel 满时降级同步处理（极端情况，20000 缓冲已很难触发）
		processRedisBatch([]UsageLog{entry})
	}
}

// redisWorker 后台批量执行 Redis Pipeline，每 100ms 或满 500 条刷一次
// 500 条 * 16 命令 = 8000 HINCRBY 一次网络往返，吞吐极高
func redisWorker() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	batch := make([]UsageLog, 0, 500)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		processRedisBatch(batch)
		batch = batch[:0]
	}
	for {
		select {
		case entry := <-redisLogCh:
			batch = append(batch, entry)
			if len(batch) >= 500 {
				flush()
			}
		case <-ticker.C:
			flush()
			cleanupRedisOldKeys() // 顺带清理 7 天前的按天分片 key
		}
	}
}

// processRedisBatch 批量执行 Pipeline：聚合同 key+field 的增量，减少命令数
// 注意：不检查 redisEnabled，仅检查 rdb，以便 closeRedis 时刷盘待写入统计
func processRedisBatch(batch []UsageLog) {
	client := rdb.Load()
	if client == nil || len(batch) == 0 {
		return
	}
	ctx := context.Background()
	pipe := client.Pipeline()
	// 聚合：同一 (key, field) 累加，减少 HIncrBy 命令数
	type agg struct {
		count                int64
		input, output, total int64
		ttft, duration       int64 // 性能指标：首 token 延迟与总耗时（毫秒）
		cacheHit, cacheMiss  int64 // 缓存命中/未命中输入 token
	}
	total := agg{}
	byProvider := map[string]*agg{}
	byModel := map[string]*agg{}
	byDate := map[string]*agg{}
	today := time.Now().Format("2006-01-02")
	byModelToday := map[string]int64{}
	for _, e := range batch {
		total.count++
		total.input += int64(e.InputTokens)
		total.output += int64(e.OutputTokens)
		total.total += int64(e.TotalTokens)
		total.ttft += int64(e.TTFTMs)
		total.duration += int64(e.DurationMs)
		total.cacheHit += int64(e.CacheHit)
		total.cacheMiss += int64(e.CacheMiss)

		p := byProvider[e.ProviderName]
		if p == nil {
			p = &agg{}
			byProvider[e.ProviderName] = p
		}
		p.count++
		p.input += int64(e.InputTokens)
		p.output += int64(e.OutputTokens)
		p.total += int64(e.TotalTokens)

		m := byModel[e.Model]
		if m == nil {
			m = &agg{}
			byModel[e.Model] = m
		}
		m.count++
		m.input += int64(e.InputTokens)
		m.output += int64(e.OutputTokens)
		m.total += int64(e.TotalTokens)

		date := e.Timestamp.Format("2006-01-02")
		d := byDate[date]
		if d == nil {
			d = &agg{}
			byDate[date] = d
		}
		d.count++
		d.input += int64(e.InputTokens)
		d.output += int64(e.OutputTokens)
		d.total += int64(e.TotalTokens)

		// 当日模型请求次数
		if date == today {
			byModelToday[e.Model]++
		}
	}
	// 总计（按天分片存储，读取时聚合近 N 天窗口）
	pipe.HIncrBy(ctx, "usage:total:"+today, "totalReqs", total.count)
	pipe.HIncrBy(ctx, "usage:total:"+today, "totalInput", total.input)
	pipe.HIncrBy(ctx, "usage:total:"+today, "totalOutput", total.output)
	pipe.HIncrBy(ctx, "usage:total:"+today, "totalTokens", total.total)
	pipe.HIncrBy(ctx, "usage:total:"+today, "totalTTFT", total.ttft)
	pipe.HIncrBy(ctx, "usage:total:"+today, "totalDuration", total.duration)
	pipe.HIncrBy(ctx, "usage:total:"+today, "cacheHit", total.cacheHit)
	pipe.HIncrBy(ctx, "usage:total:"+today, "cacheMiss", total.cacheMiss)
	// 按站点
	for name, a := range byProvider {
		pipe.HIncrBy(ctx, "usage:byProvider:"+today, name+"|count", a.count)
		pipe.HIncrBy(ctx, "usage:byProvider:"+today, name+"|input", a.input)
		pipe.HIncrBy(ctx, "usage:byProvider:"+today, name+"|output", a.output)
		pipe.HIncrBy(ctx, "usage:byProvider:"+today, name+"|total", a.total)
	}
	// 按模型
	for model, a := range byModel {
		pipe.HIncrBy(ctx, "usage:byModel:"+today, model+"|count", a.count)
		pipe.HIncrBy(ctx, "usage:byModel:"+today, model+"|input", a.input)
		pipe.HIncrBy(ctx, "usage:byModel:"+today, model+"|output", a.output)
		pipe.HIncrBy(ctx, "usage:byModel:"+today, model+"|total", a.total)
	}
	// 按日期（每天一个 hash，field 为 count/input/output/total）
	for date, a := range byDate {
		pipe.HIncrBy(ctx, "usage:byDate:"+date, "count", a.count)
		pipe.HIncrBy(ctx, "usage:byDate:"+date, "input", a.input)
		pipe.HIncrBy(ctx, "usage:byDate:"+date, "output", a.output)
		pipe.HIncrBy(ctx, "usage:byDate:"+date, "total", a.total)
	}
	// 当日模型请求次数（按日期为键，避免跨天残留）
	for model, count := range byModelToday {
		pipe.HIncrBy(ctx, "usage:byModelToday:"+today, model, count)
	}
	pipe.Do(ctx)
}

// lastNDays 返回近 n 天的日期列表（含今天，从新到旧）
func lastNDays(n int) []string {
	days := make([]string, 0, n)
	now := time.Now()
	for i := 0; i < n; i++ {
		days = append(days, now.AddDate(0, 0, -i).Format("2006-01-02"))
	}
	return days
}

// cleanupRedisOldKeys 清理 7 天前的 usage:* 按天分片 key，避免无限增长
func cleanupRedisOldKeys() {
	client := rdb.Load()
	if client == nil {
		return
	}
	ctx := context.Background()
	cutoff := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	keys, err := client.Keys(ctx, "usage:*:*").Result()
	if err != nil {
		return
	}
	for _, k := range keys {
		parts := strings.Split(k, ":")
		// 格式均为 usage:维度:YYYY-MM-DD，日期在第 3 段
		if len(parts) == 3 && parts[2] < cutoff {
			client.Del(ctx, k)
		}
	}
}

// flushRedisLogs 刷盘待写入的 Redis 日志（用于优雅关闭时调用）
func flushRedisLogs() {
	if redisLogCh == nil {
		return
	}
	batch := make([]UsageLog, 0, 500)
	for {
		select {
		case entry := <-redisLogCh:
			batch = append(batch, entry)
			if len(batch) >= 500 {
				processRedisBatch(batch)
				batch = batch[:0]
			}
		default:
			if len(batch) > 0 {
				processRedisBatch(batch)
			}
			return
		}
	}
}

// redisGetUsageStats 从 Redis 读取统计数据（聚合近 7 天的按天分片）
func redisGetUsageStats() map[string]any {
	if !redisReady() {
		return nil
	}
	client := rdb.Load()
	ctx := context.Background()
	result := map[string]any{}

	total := &dayAgg{}
	byProvider := map[string]*dayAgg{}
	byModel := map[string]*dayAgg{}
	byDate := map[string]*dayAgg{}

	// 聚合近 7 天（含今日）；byDate 提前初始化，保证图表有完整 7 天轴
	days := lastNDays(7)
	for _, day := range days {
		byDate[day] = &dayAgg{}
		// 总计
		if f, err := client.HGetAll(ctx, "usage:total:"+day).Result(); err == nil {
			total.count += parseInt64(f["totalReqs"])
			total.input += parseInt64(f["totalInput"])
			total.output += parseInt64(f["totalOutput"])
			total.total += parseInt64(f["totalTokens"])
			total.ttft += parseInt64(f["totalTTFT"])
			total.duration += parseInt64(f["totalDuration"])
			total.cacheHit += parseInt64(f["cacheHit"])
			total.cacheMiss += parseInt64(f["cacheMiss"])
		}
		// 按站点
		if pf, err := client.HGetAll(ctx, "usage:byProvider:"+day).Result(); err == nil {
			for k, v := range pf {
				parts := strings.SplitN(k, "|", 2)
				if len(parts) != 2 {
					continue
				}
				a := byProvider[parts[0]]
				if a == nil {
					a = &dayAgg{}
					byProvider[parts[0]] = a
				}
				addAggField(a, parts[1], parseInt64(v))
			}
		}
		// 按模型
		if mf, err := client.HGetAll(ctx, "usage:byModel:"+day).Result(); err == nil {
			for k, v := range mf {
				parts := strings.SplitN(k, "|", 2)
				if len(parts) != 2 {
					continue
				}
				a := byModel[parts[0]]
				if a == nil {
					a = &dayAgg{}
					byModel[parts[0]] = a
				}
				addAggField(a, parts[1], parseInt64(v))
			}
		}
		// 按日期（每天一个 hash）
		if df, err := client.HGetAll(ctx, "usage:byDate:"+day).Result(); err == nil {
			d := byDate[day]
			d.count = parseInt64(df["count"])
			d.input = parseInt64(df["input"])
			d.output = parseInt64(df["output"])
			d.total = parseInt64(df["total"])
		}
	}

	result["totalInput"] = total.input
	result["totalOutput"] = total.output
	result["totalTokens"] = total.total
	result["totalReqs"] = total.count
	result["avgTTFTMs"] = float64(0)
	result["avgOutputSpeed"] = float64(0)
	result["cacheHitRate"] = float64(-1)
	if total.count > 0 {
		result["avgTTFTMs"] = float64(total.ttft) / float64(total.count)
		if total.duration > 0 {
			result["avgOutputSpeed"] = float64(total.output) / (float64(total.duration) / 1000.0)
		}
	}
	if total.cacheHit+total.cacheMiss > 0 {
		result["cacheHitRate"] = float64(total.cacheHit) / float64(total.cacheHit+total.cacheMiss)
	}

	provOut := map[string]map[string]int64{}
	for name, a := range byProvider {
		provOut[name] = map[string]int64{"count": a.count, "input": a.input, "output": a.output, "total": a.total}
	}
	modelOut := map[string]map[string]int64{}
	for name, a := range byModel {
		modelOut[name] = map[string]int64{"count": a.count, "input": a.input, "output": a.output, "total": a.total}
	}
	dateOut := map[string]map[string]int64{}
	for day, a := range byDate {
		dateOut[day] = map[string]int64{"count": a.count, "input": a.input, "output": a.output, "total": a.total}
	}
	result["byProvider"] = provOut
	result["byModel"] = modelOut
	result["byDate"] = dateOut

	// 当日各模型请求次数
	today := time.Now().Format("2006-01-02")
	todayModelFields, err := client.HGetAll(ctx, "usage:byModelToday:"+today).Result()
	byModelToday := map[string]int64{}
	if err == nil {
		for model, v := range todayModelFields {
			byModelToday[model] = parseInt64(v)
		}
	}
	result["byModelToday"] = byModelToday

	return result
}

// addAggField 将 "name|field" 聚合值累加到按天聚合结构
func addAggField(a *dayAgg, field string, v int64) {
	switch field {
	case "count":
		a.count += v
	case "input":
		a.input += v
	case "output":
		a.output += v
	case "total":
		a.total += v
	}
}

func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// redisClearStats 清空 Redis 统计（全部 usage:* key，含按天分片）
func redisClearStats() {
	if !redisReady() {
		return
	}
	client := rdb.Load()
	ctx := context.Background()
	keys, err := client.Keys(ctx, "usage:*").Result()
	if err == nil && len(keys) > 0 {
		client.Del(ctx, keys...)
	}
}
