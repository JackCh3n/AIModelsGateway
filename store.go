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

	normalizeConfig(&c)

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

// normalizeConfig 补齐配置的默认值与字段初始化（加载与还原共用）
func normalizeConfig(c *Config) {
	if c.Providers == nil {
		c.Providers = []Provider{}
	}
	if c.APIKeys == nil {
		c.APIKeys = []APIKey{}
	}
	if c.Aliases == nil {
		c.Aliases = []ModelAlias{}
	}
	if c.Failovers == nil {
		c.Failovers = []FailoverRoute{}
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
}

// restoreConfig 用备份 JSON 整体替换当前配置并持久化（一键还原）
func restoreConfig(data []byte) error {
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("配置 JSON 解析失败: %v", err)
	}
	if c.Providers == nil && c.APIKeys == nil && c.Aliases == nil && c.Failovers == nil {
		return fmt.Errorf("无效的配置备份（缺少 providers/apiKeys/aliases/failovers 字段）")
	}
	normalizeConfig(&c)
	c.idx = buildIndex(&c)
	configPtr.Store(&c)
	persistConfig(&c)
	return nil
}

// buildIndex 构建预计算索引，将热路径 O(n) 查找优化为 O(1)
func buildIndex(cfg *Config) configIndex {
	idx := configIndex{
		apiKeySet:         make(map[string]bool),
		aliasMap:          make(map[string]ModelAlias),
		failoverMap:       make(map[string]FailoverRoute),
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
	for _, f := range cfg.Failovers {
		idx.failoverMap[f.Name] = f
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
	// 同步持久化：管理操作低频，且避免多个异步 goroutine 乱序写盘导致配置丢失
	persistConfig(&newCfg)
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
	if old.Failovers != nil {
		newCfg.Failovers = make([]FailoverRoute, len(old.Failovers))
		copy(newCfg.Failovers, old.Failovers)
		for i := range newCfg.Failovers {
			f := &newCfg.Failovers[i]
			if f.Entries != nil {
				f.Entries = append([]FailoverEntry{}, f.Entries...)
			}
		}
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
			// 清理引用该站点的别名与主备路由条目，避免产生悬空引用
			aliases := cfg.Aliases[:0]
			for _, a := range cfg.Aliases {
				if a.ProviderID != id {
					aliases = append(aliases, a)
				}
			}
			cfg.Aliases = aliases
			failovers := cfg.Failovers[:0]
			for _, f := range cfg.Failovers {
				entries := f.Entries[:0]
				for _, e := range f.Entries {
					if e.ProviderID != id {
						entries = append(entries, e)
					}
				}
				f.Entries = entries
				if len(f.Entries) > 0 { // 条目清空的路由一并删除
					failovers = append(failovers, f)
				}
			}
			cfg.Failovers = failovers
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

func addAlias(a ModelAlias) (ModelAlias, error) {
	if a.Name == "" {
		return a, fmt.Errorf("路由名称不能为空")
	}
	if isAliasNameConflict(a.Name, "") {
		return a, fmt.Errorf("路由名称「%s」与已有别名或主备路由重复，请换一个", a.Name)
	}
	a.ID = generateID("alias")
	mutateConfig(func(cfg *Config) {
		cfg.Aliases = append(cfg.Aliases, a)
	})
	return a, nil
}

func updateAlias(a ModelAlias) (bool, error) {
	if a.Name == "" {
		return false, fmt.Errorf("路由名称不能为空")
	}
	if isAliasNameConflict(a.Name, a.ID) {
		return false, fmt.Errorf("路由名称「%s」与已有别名或主备路由重复，请换一个", a.Name)
	}
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
	return updated, nil
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

// --- Failover 主备路由 ---

func listFailovers() []FailoverRoute {
	cfg := loadConfig()
	result := make([]FailoverRoute, len(cfg.Failovers))
	copy(result, cfg.Failovers)
	return result
}

// isModelNameConflict 检查名称是否与已有别名或主备路由冲突（排除自身）
func isModelNameConflict(name, excludeFailoverID string) bool {
	cfg := loadConfig()
	if _, ok := cfg.idx.aliasMap[name]; ok {
		return true
	}
	for _, f := range cfg.Failovers {
		if f.ID != excludeFailoverID && f.Name == name {
			return true
		}
	}
	return false
}

// isAliasNameConflict 检查别名名称是否与已有别名（排除自身）或主备路由冲突
func isAliasNameConflict(name, excludeAliasID string) bool {
	if name == "" {
		return true
	}
	cfg := loadConfig()
	for _, a := range cfg.Aliases {
		if a.ID != excludeAliasID && a.Name == name {
			return true
		}
	}
	for _, f := range cfg.Failovers {
		if f.Name == name {
			return true
		}
	}
	return false
}

func addFailover(f FailoverRoute) (FailoverRoute, error) {
	if f.Name == "" {
		return f, fmt.Errorf("路由名称不能为空")
	}
	if isModelNameConflict(f.Name, "") {
		return f, fmt.Errorf("路由名称「%s」与已有别名或主备路由重复，请换一个", f.Name)
	}
	f.ID = generateID("fo")
	sortFailoverEntries(&f)
	mutateConfig(func(cfg *Config) {
		cfg.Failovers = append(cfg.Failovers, f)
	})
	return f, nil
}

func updateFailover(f FailoverRoute) (bool, error) {
	if f.Name == "" {
		return false, fmt.Errorf("路由名称不能为空")
	}
	if isModelNameConflict(f.Name, f.ID) {
		return false, fmt.Errorf("路由名称「%s」与已有别名或主备路由重复，请换一个", f.Name)
	}
	updated := false
	sortFailoverEntries(&f)
	mutateConfig(func(cfg *Config) {
		for i := range cfg.Failovers {
			if cfg.Failovers[i].ID == f.ID {
				cfg.Failovers[i] = f
				updated = true
				return
			}
		}
	})
	return updated, nil
}

func deleteFailover(id string) bool {
	deleted := false
	mutateConfig(func(cfg *Config) {
		for i := range cfg.Failovers {
			if cfg.Failovers[i].ID == id {
				cfg.Failovers = append(cfg.Failovers[:i], cfg.Failovers[i+1:]...)
				deleted = true
				return
			}
		}
	})
	return deleted
}

// sortFailoverEntries 按优先级排序（1 主站优先），并截断最多 6 个
func sortFailoverEntries(f *FailoverRoute) {
	entries := f.Entries
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].Order < entries[j-1].Order; j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
	if len(entries) > 6 {
		entries = entries[:6]
	}
	f.Entries = entries
}

// getFailoverByName 根据模型名查找主备路由（O(1)）
func getFailoverByName(model string) *FailoverRoute {
	cfg := loadConfig()
	if f, ok := cfg.idx.failoverMap[model]; ok {
		return &f
	}
	return nil
}

// --- Usage ---

// usageLogCh 异步写入 channel：请求路径不再同步写文件，大幅提升并发能力
// 狂暴模式下 buffer 加大到 20000，避免高并发满溢降级为同步写拖慢请求
var usageLogCh = make(chan UsageLog, 20000)

// errorLogCh 错误日志异步写入 channel，避免请求路径同步写库拖慢
var errorLogCh = make(chan ErrorLog, 10000)

// worker 停止信号：优雅关闭时先让 worker 刷掉自己手里的 batch，再 drain 剩余，
// 避免 worker 与 flush 并发抢 channel 导致已取走但未落库的数据丢失
var (
	usageWorkerStop    = make(chan struct{})
	usageWorkerStopOnce sync.Once
	errorWorkerStop    = make(chan struct{})
	errorWorkerStopOnce sync.Once
	// worker 退出等待：flush 时 close 停止信号后，等待 worker 刷完自己的 batch 再返回
	usageWorkerWG sync.WaitGroup
	errorWorkerWG sync.WaitGroup
)

func init() {
	usageWorkerWG.Add(1)
	errorWorkerWG.Add(1)
	go usageLogWorker()
	go errorLogWorker()
}

// errorLogWorker 后台批量写入错误日志，每 1 秒或满 500 条刷一次
func errorLogWorker() {
	defer errorWorkerWG.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	batch := make([]ErrorLog, 0, 500)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		dbBatchInsertErrorLogs(batch)
		batch = batch[:0]
	}
	for {
		select {
		case entry := <-errorLogCh:
			batch = append(batch, entry)
			if len(batch) >= 500 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-errorWorkerStop:
			flush() // 先刷掉已取走的 batch 再退出
			return
		}
	}
}

// usageLogWorker 后台批量写入 SQLite，每 1 秒或满 500 条刷一次
// 500 条一个事务，减少 SQLite 写锁竞争，提升吞吐
func usageLogWorker() {
	defer usageWorkerWG.Done()
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
		case <-usageWorkerStop:
			flush() // 先刷掉已取走的 batch 再退出
			return
		}
	}
}

// addUsageLog 异步记录用量（非阻塞），不拖慢请求路径
func addUsageLog(logEntry UsageLog) {
	// 成本估算：输入/输出 token × 每百万价格（Settings 配置，默认 0 不计费）
	s := getSettings()
	logEntry.Cost = float64(logEntry.InputTokens)/1e6*s.InputPricePerMTok +
		float64(logEntry.OutputTokens)/1e6*s.OutputPricePerMTok
	// 可观测性：累计 token / 缓存 / TTFT 指标（原子计数，无锁）
	metricInputTokens.Add(uint64(logEntry.InputTokens))
	metricOutputTokens.Add(uint64(logEntry.OutputTokens))
	metricCacheHit.Add(uint64(logEntry.CacheHit))
	metricCacheMiss.Add(uint64(logEntry.CacheMiss))
	if logEntry.TTFTMs > 0 {
		metricTTFTSumMs.Add(uint64(logEntry.TTFTMs))
		metricTTFTCount.Add(1)
	}
	// 日志明细异步写 SQLite（用于日志查看页面）
	select {
	case usageLogCh <- logEntry:
	default:
		dbAddUsageLog(logEntry)
	}
	// 狂暴模式：Redis 实时统计
	if redisReady() {
		redisIncrUsage(logEntry)
	}
}

// flushUsageLogs 刷盘待写入的用量日志（用于优雅关闭时调用）
func flushUsageLogs() {
	// 先通知 worker 退出并刷掉它已取走的 batch，等待其落库完成，再 drain 剩余，避免丢数据
	usageWorkerStopOnce.Do(func() { close(usageWorkerStop) })
	usageWorkerWG.Wait()
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

// flushErrorLogs 刷盘待写入的错误日志（用于优雅关闭时调用）
func flushErrorLogs() {
	// 先通知 worker 退出并刷掉它已取走的 batch，等待其落库完成，再 drain 剩余，避免丢数据
	errorWorkerStopOnce.Do(func() { close(errorWorkerStop) })
	errorWorkerWG.Wait()
	batch := make([]ErrorLog, 0, 500)
	for {
		select {
		case entry := <-errorLogCh:
			batch = append(batch, entry)
			if len(batch) >= 500 {
				dbBatchInsertErrorLogs(batch)
				batch = batch[:0]
			}
		default:
			if len(batch) > 0 {
				dbBatchInsertErrorLogs(batch)
			}
			return
		}
	}
}

func getUsageStats() map[string]any {
	// 狂暴模式：优先从 Redis 读取（内存级延迟）
	if redisReady() {
		if stats := redisGetUsageStats(); stats != nil {
			return stats
		}
	}
	return dbGetUsageStats()
}

func getRecentLogs(page, pageSize int) ([]UsageLog, int) {
	return dbGetRecentLogs(page, pageSize)
}

// addErrorLog 记录一条上游错误日志（校验后写入）
func addErrorLog(entry ErrorLog) {
	if entry.StatusCode == 0 {
		return
	}
	// 可观测性：上游错误计数
	metricUpstreamErrors.Add(1)
	if len(entry.Message) > 500 {
		entry.Message = entry.Message[:500]
	}
	// 异步写入，不阻塞请求路径；channel 满时降级为同步写
	select {
	case errorLogCh <- entry:
	default:
		dbAddErrorLog(entry)
	}
}

func getErrorLogs(page, pageSize int) ([]ErrorLog, int) {
	return dbGetErrorLogs(page, pageSize)
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
	oldRedisAddr := ""
	oldRedisPwd := ""
	oldRedisDB := 0
	mutateConfig(func(cfg *Config) {
		oldRage = cfg.Settings.RageMode
		oldRedisAddr = cfg.Settings.RedisAddr
		oldRedisPwd = cfg.Settings.RedisPassword
		oldRedisDB = cfg.Settings.RedisDB
		cfg.Settings.ActiveProviderID = s.ActiveProviderID
		cfg.Settings.DefaultModel = s.DefaultModel
		cfg.Settings.ListenAddr = s.ListenAddr
		cfg.Settings.InputPricePerMTok = s.InputPricePerMTok
		cfg.Settings.OutputPricePerMTok = s.OutputPricePerMTok
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
		// UserAgent 允许清空（前端每次保存都会携带该字段）
		cfg.Settings.UserAgent = s.UserAgent
		// 管理员登录密码摘要（前端不回传，仅在 PUT 提供时更新）
		cfg.Settings.AdminPasswordHash = s.AdminPasswordHash
		// 主备路由单节点超时（秒），0 视为默认 60
		if s.FailoverTimeout > 0 {
			cfg.Settings.FailoverTimeout = s.FailoverTimeout
		}
	})
	// 狂暴模式切换：开 -> 连 Redis；关 -> 断 Redis；地址变更 -> 重连
	redisChanged := s.RedisAddr != oldRedisAddr || s.RedisPassword != oldRedisPwd || s.RedisDB != oldRedisDB
	if s.RageMode && !oldRage {
		// 开启：初始化 Redis + 加大连接池
		go initRedis(s.RedisAddr, s.RedisPassword, s.RedisDB)
	} else if !s.RageMode && oldRage {
		// 关闭：刷盘 + 断开 Redis + 恢复普通连接池
		go closeRedis()
	} else if s.RageMode && oldRage && redisChanged {
		// 已启用但 Redis 配置变更：先关再开（复用刷盘逻辑避免丢数据）
		go func() {
			closeRedis()
			initRedis(s.RedisAddr, s.RedisPassword, s.RedisDB)
		}()
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
