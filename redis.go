package main

import (
	"context"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	rdb          *redis.Client
	redisEnabled bool
	redisMu      sync.Mutex
	// redisLogCh 异步写入 channel：请求路径不再同步等 Pipeline 返回，支持万级并发
	redisLogCh   chan UsageLog
	redisLogOnce sync.Once
)

// initRedis 初始化 Redis 连接（狂暴模式启用时调用）
func initRedis(addr, password string, db int) {
	redisMu.Lock()
	defer redisMu.Unlock()

	if addr == "" {
		log.Printf("[redis] 地址为空，狂暴模式未启用")
		return
	}

	if rdb != nil {
		rdb.Close()
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     50,
		MinIdleConns: 10,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[redis] 连接失败: %v", err)
		rdb = nil
		redisEnabled = false
		return
	}

	redisEnabled = true
	// 启动异步 worker（仅一次）
	redisLogOnce.Do(func() {
		redisLogCh = make(chan UsageLog, 20000)
		go redisWorker()
	})
	applyRageModePool()
	log.Printf("[redis] 已连接（狂暴模式启用）: %s db=%d", addr, db)
}

// closeRedis 关闭 Redis 连接（狂暴模式关闭时调用）
func closeRedis() {
	redisMu.Lock()
	defer redisMu.Unlock()

	if rdb != nil {
		rdb.Close()
		rdb = nil
	}
	redisEnabled = false
	restoreNormalPool()
	log.Printf("[redis] 已断开（恢复普通模式）")
}

// redisIncrUsage 异步递增统计：仅向 channel 投递，不阻塞请求路径
// 由后台 redisWorker 批量执行 Pipeline，一次网络往返完成 N*16 条 HINCRBY
func redisIncrUsage(entry UsageLog) {
	if !redisEnabled || rdb == nil || redisLogCh == nil {
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
		}
	}
}

// processRedisBatch 批量执行 Pipeline：聚合同 key+field 的增量，减少命令数
func processRedisBatch(batch []UsageLog) {
	if !redisEnabled || rdb == nil || len(batch) == 0 {
		return
	}
	ctx := context.Background()
	pipe := rdb.Pipeline()
	// 聚合：同一 (key, field) 累加，减少 HIncrBy 命令数
	type agg struct {
		count               int64
		input, output, total int64
	}
	total := agg{}
	byProvider := map[string]*agg{}
	byModel := map[string]*agg{}
	byDate := map[string]*agg{}
	for _, e := range batch {
		total.count++
		total.input += int64(e.InputTokens)
		total.output += int64(e.OutputTokens)
		total.total += int64(e.TotalTokens)

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
	}
	// 总计
	pipe.HIncrBy(ctx, "usage:total", "totalReqs", total.count)
	pipe.HIncrBy(ctx, "usage:total", "totalInput", total.input)
	pipe.HIncrBy(ctx, "usage:total", "totalOutput", total.output)
	pipe.HIncrBy(ctx, "usage:total", "totalTokens", total.total)
	// 按站点
	for name, a := range byProvider {
		pipe.HIncrBy(ctx, "usage:byProvider", name+"|count", a.count)
		pipe.HIncrBy(ctx, "usage:byProvider", name+"|input", a.input)
		pipe.HIncrBy(ctx, "usage:byProvider", name+"|output", a.output)
		pipe.HIncrBy(ctx, "usage:byProvider", name+"|total", a.total)
	}
	// 按模型
	for model, a := range byModel {
		pipe.HIncrBy(ctx, "usage:byModel", model+"|count", a.count)
		pipe.HIncrBy(ctx, "usage:byModel", model+"|input", a.input)
		pipe.HIncrBy(ctx, "usage:byModel", model+"|output", a.output)
		pipe.HIncrBy(ctx, "usage:byModel", model+"|total", a.total)
	}
	// 按日期
	for date, a := range byDate {
		pipe.HIncrBy(ctx, "usage:byDate", date+"|count", a.count)
		pipe.HIncrBy(ctx, "usage:byDate", date+"|input", a.input)
		pipe.HIncrBy(ctx, "usage:byDate", date+"|output", a.output)
		pipe.HIncrBy(ctx, "usage:byDate", date+"|total", a.total)
	}
	pipe.Do(ctx)
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

// redisGetUsageStats 从 Redis 读取统计数据
func redisGetUsageStats() map[string]any {
	if !redisEnabled || rdb == nil {
		return nil
	}
	ctx := context.Background()
	result := map[string]any{}

	// 总计
	totalFields, err := rdb.HGetAll(ctx, "usage:total").Result()
	if err == nil {
		result["totalInput"] = parseInt64(totalFields["totalInput"])
		result["totalOutput"] = parseInt64(totalFields["totalOutput"])
		result["totalTokens"] = parseInt64(totalFields["totalTokens"])
		result["totalReqs"] = parseInt64(totalFields["totalReqs"])
	} else {
		result["totalInput"] = int64(0)
		result["totalOutput"] = int64(0)
		result["totalTokens"] = int64(0)
		result["totalReqs"] = int64(0)
	}

	// 按维度解析 Hash
	result["byProvider"] = parseUsageHash(rdb.HGetAll(ctx, "usage:byProvider").Result())
	result["byModel"] = parseUsageHash(rdb.HGetAll(ctx, "usage:byModel").Result())
	result["byDate"] = parseUsageHash(rdb.HGetAll(ctx, "usage:byDate").Result())

	return result
}

// parseUsageHash 解析 "name|field" -> value 的 Hash 为嵌套 map
func parseUsageHash(fields map[string]string, err error) map[string]map[string]int64 {
	m := map[string]map[string]int64{}
	if err != nil {
		return m
	}
	for k, v := range fields {
		parts := strings.SplitN(k, "|", 2)
		if len(parts) != 2 {
			continue
		}
		name, field := parts[0], parts[1]
		if m[name] == nil {
			m[name] = map[string]int64{}
		}
		m[name][field] = parseInt64(v)
	}
	return m
}

func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// redisClearStats 清空 Redis 统计
func redisClearStats() {
	if !redisEnabled || rdb == nil {
		return
	}
	ctx := context.Background()
	rdb.Del(ctx, "usage:total", "usage:byProvider", "usage:byModel", "usage:byDate")
}
