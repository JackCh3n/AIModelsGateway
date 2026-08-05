package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	configPtr  atomic.Pointer[Config] // 无锁读：请求路径 Load() 无任何锁竞争
	configOnce sync.Once              // 首次加载保护
	cfgPath    string
	persistMu  sync.Mutex // 持久化文件写互斥，防并发写冲突
)

func init() {
	exe, _ := os.Executable()
	dataDir := filepath.Join(filepath.Dir(exe), "data")
	os.MkdirAll(dataDir, 0755)
	cfgPath = filepath.Join(dataDir, "config.json")
}

// loadConfig 无锁读：atomic.Pointer.Load，O(1)，数千并发零竞争
func loadConfig() *Config {
	configOnce.Do(loadConfigFromFile)
	return configPtr.Load()
}

// loadConfigFromFile 从文件加载配置（仅首次调用）
func loadConfigFromFile() {
	cfg := &Config{
		Providers: []Provider{},
		APIKeys:   []APIKey{},
		Settings:  Settings{DefaultModel: "all"},
		UsageLogs: []UsageLog{},
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		cfg.idx = buildIndex(cfg)
		configPtr.Store(cfg)
		return
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		log.Printf("config parse error: %v", err)
		cfg.idx = buildIndex(cfg)
		configPtr.Store(cfg)
		return
	}

	if c.Providers == nil {
		c.Providers = []Provider{}
	}
	if c.APIKeys == nil {
		c.APIKeys = []APIKey{}
	}
	if c.Aliases == nil {
		c.Aliases = []ModelAlias{}
	}
	if c.UsageLogs == nil {
		c.UsageLogs = []UsageLog{}
	}
	if c.Settings.DefaultModel == "" {
		c.Settings.DefaultModel = "all"
	}
	if len(c.Settings.InputPresets) == 0 {
		c.Settings.InputPresets = []string{"32K", "64K", "128K", "256K", "384K", "512K", "1M"}
	}
	if len(c.Settings.OutputPresets) == 0 {
		c.Settings.OutputPresets = []string{"8K", "16K", "32K", "64K", "128K", "256K", "384K"}
	}
	for i := range c.Providers {
		ensureDisabledModelsInit(&c.Providers[i])
		if c.Providers[i].APIKey != "" && len(c.Providers[i].APIKeys) == 0 {
			c.Providers[i].APIKeys = []ProviderKey{{
				ID:     generateID("pk"),
				Key:    c.Providers[i].APIKey,
				Name:   "默认",
				Status: "active",
			}}
		}
	}

	// 迁移旧 JSON UsageLogs 到 SQLite
	migrated := false
	if len(c.UsageLogs) > 0 {
		initDB()
		if db != nil {
			log.Printf("[migrate] 迁移 %d 条日志到 SQLite", len(c.UsageLogs))
			for _, l := range c.UsageLogs {
				dbAddUsageLog(l)
			}
			c.UsageLogs = nil
			migrated = true
		}
	}

	c.idx = buildIndex(&c)
	configPtr.Store(&c)

	if migrated {
		go persistConfig(&c)
	}
}

// buildIndex 构建预计算索引，将热路径 O(n) 查找优化为 O(1)
func buildIndex(cfg *Config) configIndex {
	idx := configIndex{
		apiKeySet:         make(map[string]bool),
		aliasMap:          make(map[string]ModelAlias),
		providerMap:       make(map[string]int),
		activeProviderIdx: -1,
	}
	for i := range cfg.Providers {
		idx.providerMap[cfg.Providers[i].ID] = i
	}
	for _, k := range cfg.APIKeys {
		if k.Status == "active" {
			idx.apiKeySet[k.Key] = true
		}
	}
	for _, a := range cfg.Aliases {
		idx.aliasMap[a.Name] = a
	}
	if cfg.Settings.ActiveProviderID != "" {
		if i, ok := idx.providerMap[cfg.Settings.ActiveProviderID]; ok && cfg.Providers[i].Status == "active" {
			idx.activeProviderIdx = i
		}
	}
	if idx.activeProviderIdx < 0 {
		for i := range cfg.Providers {
			if cfg.Providers[i].Status == "active" {
				idx.activeProviderIdx = i
				break
			}
		}
	}
	return idx
}

// mutateConfig copy-on-write：深拷贝当前配置，修改副本，原子替换指针
// 请求路径读到的旧指针永远不会被修改，彻底消除数据竞争
func mutateConfig(fn func(cfg *Config)) {
	old := configPtr.Load()
	newCfg := shallowCopyConfig(old)
	newCfg.idx = old.idx // 复用旧索引（slice 下标不变），供 fn 内部查找
	fn(&newCfg)
	newCfg.idx = buildIndex(&newCfg) // fn 修改后重建索引
	configPtr.Store(&newCfg)
	go persistConfig(&newCfg)
}

// shallowCopyConfig 深拷贝配置（管理操作低频，拷贝开销可忽略）
func shallowCopyConfig(old *Config) Config {
	newCfg := Config{
		Settings:  old.Settings,
		UsageLogs: old.UsageLogs,
	}
	if old.Providers != nil {
		newCfg.Providers = make([]Provider, len(old.Providers))
		copy(newCfg.Providers, old.Providers)
		for i := range newCfg.Providers {
			p := &newCfg.Providers[i]
			if p.Models != nil {
				p.Models = append([]string{}, p.Models...)
			}
			if p.DisabledModels != nil {
				p.DisabledModels = append([]string{}, p.DisabledModels...)
			}
			if p.CustomHeaders != nil {
				m := make(map[string]string, len(p.CustomHeaders))
				for k, v := range p.CustomHeaders {
					m[k] = v
				}
				p.CustomHeaders = m
			}
			if p.APIKeys != nil {
				p.APIKeys = append([]ProviderKey{}, p.APIKeys...)
			}
			if p.ModelConfigs != nil {
				p.ModelConfigs = append([]ModelConfig{}, p.ModelConfigs...)
			}
		}
	}
	if old.APIKeys != nil {
		newCfg.APIKeys = append([]APIKey{}, old.APIKeys...)
	}
	if old.Aliases != nil {
		newCfg.Aliases = append([]ModelAlias{}, old.Aliases...)
	}
	return newCfg
}

// persistConfig 持久化到文件（原子写 + mutex 防并发）
func persistConfig(cfg *Config) {
	persistMu.Lock()
	defer persistMu.Unlock()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Printf("persist config marshal failed: %v", err)
		return
	}
	tmp := cfgPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		log.Printf("persist config failed: %v", err)
		return
	}
	if err := os.Rename(tmp, cfgPath); err != nil {
		log.Printf("persist config rename failed: %v", err)
	}
}

// saveConfig 同步持久化（用于需要立即写盘的场景）
func saveConfig() {
	cfg := configPtr.Load()
	if cfg != nil {
		persistConfig(cfg)
	}
}

// generateID 生成随机 ID
func generateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

// generateAPIKey 生成 sk- 前缀的 API Key
func generateAPIKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "sk-aim-" + hex.EncodeToString(b)
}

// --- Provider 读（无锁 + O(1) 索引）---

func listProviders() []Provider {
	return loadConfig().Providers
}

func getProvider(id string) *Provider {
	cfg := loadConfig()
	if i, ok := cfg.idx.providerMap[id]; ok {
		return &cfg.Providers[i]
	}
	return nil
}

func getActiveProvider() *Provider {
	cfg := loadConfig()
	if cfg.idx.activeProviderIdx >= 0 {
		return &cfg.Providers[cfg.idx.activeProviderIdx]
	}
	return nil
}

// --- Provider 写（COW）---

func addProvider(p Provider) {
	ensureDisabledModelsInit(&p)
	mutateConfig(func(cfg *Config) {
		cfg.Providers = append(cfg.Providers, p)
	})
}

func updateProvider(p Provider) bool {
	updated := false
	ensureDisabledModelsInit(&p)
	mutateConfig(func(cfg *Config) {
		if i, ok := cfg.idx.providerMap[p.ID]; ok {
			cfg.Providers[i] = p
			updated = true
		}
	})
	return updated
}

func deleteProvider(id string) bool {
	deleted := false
	mutateConfig(func(cfg *Config) {
		if i, ok := cfg.idx.providerMap[id]; ok {
			cfg.Providers = append(cfg.Providers[:i], cfg.Providers[i+1:]...)
			if cfg.Settings.ActiveProviderID == id {
				cfg.Settings.ActiveProviderID = ""
			}
			deleted = true
		}
	})
	return deleted
}

func setProviderDefaultModel(providerID, model string) bool {
	updated := false
	mutateConfig(func(cfg *Config) {
		if i, ok := cfg.idx.providerMap[providerID]; ok {
			cfg.Providers[i].DefaultModel = model
			updated = true
		}
	})
	return updated
}

func checkinProvider(providerID string) (bool, string) {
	cfg := loadConfig()
	if i, ok := cfg.idx.providerMap[providerID]; ok {
		if cfg.Providers[i].CheckinURL == "" {
			return false, "该站点未配置打卡地址"
		}
		now := time.Now()
		mutateConfig(func(c *Config) {
			if j, ok := c.idx.providerMap[providerID]; ok {
				c.Providers[j].LastCheckin = now
			}
		})
		return true, "已记录打卡时间"
	}
	return false, "站点不存在"
}

func setActiveProvider(id string) bool {
	updated := false
	mutateConfig(func(cfg *Config) {
		if _, ok := cfg.idx.providerMap[id]; ok {
			cfg.Settings.ActiveProviderID = id
			updated = true
		}
	})
	return updated
}

// --- 模型启用/禁用 ---

// isModelEnabled 判断 provider 下的某模型是否启用
func isModelEnabled(p *Provider, model string) bool {
	for _, m := range p.DisabledModels {
		if m == model {
			return false
		}
	}
	return true
}

// toggleModelEnabled 切换 provider 下某模型的启用状态
func toggleModelEnabled(providerID, model string) bool {
	updated := false
	mutateConfig(func(cfg *Config) {
		if i, ok := cfg.idx.providerMap[providerID]; ok {
			found := false
			for j, m := range cfg.Providers[i].DisabledModels {
				if m == model {
					cfg.Providers[i].DisabledModels = append(cfg.Providers[i].DisabledModels[:j], cfg.Providers[i].DisabledModels[j+1:]...)
					found = true
					break
				}
			}
			if !found {
				cfg.Providers[i].DisabledModels = append(cfg.Providers[i].DisabledModels, model)
			}
			if cfg.Providers[i].DisabledModels == nil {
				cfg.Providers[i].DisabledModels = []string{}
			}
			updated = true
		}
	})
	return updated
}

// ensureDisabledModelsInit 确保字段已初始化
func ensureDisabledModelsInit(p *Provider) {
	if p.DisabledModels == nil {
		p.DisabledModels = []string{}
	}
	if p.CustomHeaders == nil {
		p.CustomHeaders = map[string]string{}
	}
	if p.APIKeys == nil {
		p.APIKeys = []ProviderKey{}
	}
	if p.ModelConfigs == nil {
		p.ModelConfigs = []ModelConfig{}
	}
}

// --- Key 轮询（atomic 无锁）---

// keyRotationMap per-provider atomic 计数器，无锁轮询
var keyRotationMap sync.Map // providerID -> *atomic.Uint64

// pickAPIKey 从 provider 的多 Key 中轮询选取一个 active 的 key
func pickAPIKey(p *Provider) string {
	activeKeys := []string{}
	for _, k := range p.APIKeys {
		if k.Status == "active" && k.Key != "" {
			activeKeys = append(activeKeys, k.Key)
		}
	}
	if len(activeKeys) == 0 {
		return p.APIKey // 回退到旧字段
	}
	if len(activeKeys) == 1 {
		return activeKeys[0]
	}
	// atomic 无锁轮询：每个 provider 独立计数器，无全局锁竞争
	val, _ := keyRotationMap.LoadOrStore(p.ID, &atomic.Uint64{})
	idx := val.(*atomic.Uint64).Add(1) - 1
	return activeKeys[idx%uint64(len(activeKeys))]
}

// getModelConfig 获取 provider 下指定模型的上下文配置
func getModelConfig(p *Provider, model string) *ModelConfig {
	for i := range p.ModelConfigs {
		if p.ModelConfigs[i].Model == model {
			return &p.ModelConfigs[i]
		}
	}
	return nil
}

// setModelConfig 设置或更新 provider 下指定模型的上下文配置
func setModelConfig(providerID string, mc ModelConfig) bool {
	updated := false
	mutateConfig(func(cfg *Config) {
		if i, ok := cfg.idx.providerMap[providerID]; ok {
			for j := range cfg.Providers[i].ModelConfigs {
				if cfg.Providers[i].ModelConfigs[j].Model == mc.Model {
					if mc.InputLimit == "" && mc.OutputLimit == "" {
						cfg.Providers[i].ModelConfigs = append(cfg.Providers[i].ModelConfigs[:j], cfg.Providers[i].ModelConfigs[j+1:]...)
					} else {
						cfg.Providers[i].ModelConfigs[j] = mc
					}
					updated = true
					return
				}
			}
			if mc.InputLimit != "" || mc.OutputLimit != "" {
				cfg.Providers[i].ModelConfigs = append(cfg.Providers[i].ModelConfigs, mc)
			}
			updated = true
		}
	})
	return updated
}

// limitToTokens 将 "32K"/"1M" 等转换为 token 数
func limitToTokens(limit string) int {
	s := strings.ToUpper(strings.TrimSpace(limit))
	var n int
	switch {
	case strings.HasSuffix(s, "M"):
		fmt.Sscanf(s, "%dM", &n)
		return n * 1000000
	case strings.HasSuffix(s, "K"):
		fmt.Sscanf(s, "%dK", &n)
		return n * 1000
	default:
		fmt.Sscanf(s, "%d", &n)
		return n
	}
}

// --- APIKey CRUD ---

func listAPIKeys() []APIKey {
	return loadConfig().APIKeys
}

func addAPIKey(k APIKey) {
	mutateConfig(func(cfg *Config) {
		cfg.APIKeys = append(cfg.APIKeys, k)
	})
}

func deleteAPIKey(id string) bool {
	deleted := false
	mutateConfig(func(cfg *Config) {
		for i := range cfg.APIKeys {
			if cfg.APIKeys[i].ID == id {
				cfg.APIKeys = append(cfg.APIKeys[:i], cfg.APIKeys[i+1:]...)
				deleted = true
				return
			}
		}
	})
	return deleted
}

func validateAPIKey(key string) bool {
	cfg := loadConfig()
	if len(cfg.APIKeys) == 0 {
		return true // 未配置 key 则允许无鉴权
	}
	return cfg.idx.apiKeySet[key] // O(1) 查找
}

// --- ModelAlias CRUD ---

func listAliases() []ModelAlias {
	cfg := loadConfig()
	result := make([]ModelAlias, len(cfg.Aliases))
	copy(result, cfg.Aliases)
	for i := range result {
		if p := getProvider(result[i].ProviderID); p != nil {
			result[i].ProviderName = p.Name
		}
	}
	return result
}

func addAlias(a ModelAlias) {
	a.ID = generateID("alias")
	mutateConfig(func(cfg *Config) {
		cfg.Aliases = append(cfg.Aliases, a)
	})
}

func updateAlias(a ModelAlias) bool {
	updated := false
	mutateConfig(func(cfg *Config) {
		for i := range cfg.Aliases {
			if cfg.Aliases[i].ID == a.ID {
				cfg.Aliases[i] = a
				updated = true
				return
			}
		}
	})
	return updated
}

func deleteAlias(id string) bool {
	deleted := false
	mutateConfig(func(cfg *Config) {
		for i := range cfg.Aliases {
			if cfg.Aliases[i].ID == id {
				cfg.Aliases = append(cfg.Aliases[:i], cfg.Aliases[i+1:]...)
				deleted = true
				return
			}
		}
	})
	return deleted
}

// getAliasByModel 根据模型名查找别名（O(1)）
func getAliasByModel(model string) *ModelAlias {
	cfg := loadConfig()
	if a, ok := cfg.idx.aliasMap[model]; ok {
		return &a
	}
	return nil
}

// --- Usage ---

// usageLogCh 异步写入 channel：请求路径不再同步写文件，大幅提升并发能力
// 狂暴模式下 buffer 加大到 20000，避免高并发满溢降级为同步写拖慢请求
var usageLogCh = make(chan UsageLog, 20000)

func init() {
	go usageLogWorker()
}

// usageLogWorker 后台批量写入 SQLite，每 1 秒或满 500 条刷一次
// 500 条一个事务，减少 SQLite 写锁竞争，提升吞吐
func usageLogWorker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	batch := make([]UsageLog, 0, 500)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		dbBatchInsertUsageLogs(batch)
		batch = batch[:0]
	}
	for {
		select {
		case entry := <-usageLogCh:
			batch = append(batch, entry)
			if len(batch) >= 500 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// addUsageLog 异步记录用量（非阻塞），不拖慢请求路径
func addUsageLog(logEntry UsageLog) {
	// 日志明细异步写 SQLite（用于日志查看页面）
	select {
	case usageLogCh <- logEntry:
	default:
		dbAddUsageLog(logEntry)
	}
	// 狂暴模式：Redis 实时统计
	if redisEnabled {
		redisIncrUsage(logEntry)
	}
}

// flushUsageLogs 刷盘待写入的用量日志（用于优雅关闭时调用）
func flushUsageLogs() {
	batch := make([]UsageLog, 0, 500)
	for {
		select {
		case entry := <-usageLogCh:
			batch = append(batch, entry)
			if len(batch) >= 500 {
				dbBatchInsertUsageLogs(batch)
				batch = batch[:0]
			}
		default:
			if len(batch) > 0 {
				dbBatchInsertUsageLogs(batch)
			}
			// 同步刷盘 Redis 待写入统计
			flushRedisLogs()
			return
		}
	}
}

func getUsageStats() map[string]any {
	// 狂暴模式：优先从 Redis 读取（内存级延迟）
	if redisEnabled {
		if stats := redisGetUsageStats(); stats != nil {
			return stats
		}
	}
	return dbGetUsageStats()
}

func getRecentLogs(page, pageSize int) ([]UsageLog, int) {
	return dbGetRecentLogs(page, pageSize)
}

func clearLogs() {
	dbClearLogs()
	redisClearStats()
}

// --- Settings ---

func getSettings() Settings {
	return loadConfig().Settings
}

func updateSettings(s Settings) {
	oldRage := false
	mutateConfig(func(cfg *Config) {
		oldRage = cfg.Settings.RageMode
		cfg.Settings.ActiveProviderID = s.ActiveProviderID
		cfg.Settings.DefaultModel = s.DefaultModel
		if len(s.InputPresets) > 0 {
			cfg.Settings.InputPresets = s.InputPresets
		}
		if len(s.OutputPresets) > 0 {
			cfg.Settings.OutputPresets = s.OutputPresets
		}
		cfg.Settings.RageMode = s.RageMode
		cfg.Settings.RedisAddr = s.RedisAddr
		cfg.Settings.RedisPassword = s.RedisPassword
		cfg.Settings.RedisDB = s.RedisDB
	})
	// 狂暴模式切换：开 -> 连 Redis；关 -> 断 Redis
	if s.RageMode && !oldRage {
		go initRedis(s.RedisAddr, s.RedisPassword, s.RedisDB)
	} else if !s.RageMode && oldRage {
		go closeRedis()
	}
}

// newUsageLog 创建用量记录
func newUsageLog(providerID, providerName, model string, input, output int, clientFormat string) UsageLog {
	return UsageLog{
		ID:           generateID("log"),
		ProviderID:   providerID,
		ProviderName: providerName,
		Model:        model,
		InputTokens:  input,
		OutputTokens: output,
		TotalTokens:  input + output,
		Timestamp:    time.Now(),
		ClientFormat: clientFormat,
	}
}
