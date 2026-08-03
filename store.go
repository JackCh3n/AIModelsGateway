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
	"time"
)

var (
	config   *Config
	configMu sync.Mutex
	cfgPath  string
)

func init() {
	exe, _ := os.Executable()
	dataDir := filepath.Join(filepath.Dir(exe), "data")
	os.MkdirAll(dataDir, 0755)
	cfgPath = filepath.Join(dataDir, "config.json")
}

func loadConfig() *Config {
	configMu.Lock()
	defer configMu.Unlock()

	if config != nil {
		return config
	}

	config = &Config{
		Providers: []Provider{},
		APIKeys:   []APIKey{},
		Settings:  Settings{DefaultModel: "all"},
		UsageLogs: []UsageLog{},
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return config
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("config parse error: %v", err)
		return config
	}

	if cfg.Providers == nil {
		cfg.Providers = []Provider{}
	}
	if cfg.APIKeys == nil {
		cfg.APIKeys = []APIKey{}
	}
	if cfg.Aliases == nil {
		cfg.Aliases = []ModelAlias{}
	}
	if cfg.UsageLogs == nil {
		cfg.UsageLogs = []UsageLog{}
	}
	if cfg.Settings.DefaultModel == "" {
		cfg.Settings.DefaultModel = "all"
	}
	if len(cfg.Settings.InputPresets) == 0 {
		cfg.Settings.InputPresets = []string{"32K", "64K", "128K", "256K", "384K", "512K", "1M"}
	}
	if len(cfg.Settings.OutputPresets) == 0 {
		cfg.Settings.OutputPresets = []string{"8K", "16K", "32K", "64K", "128K", "256K", "384K"}
	}
	// 确保每个 provider 的 DisabledModels 已初始化
	for i := range cfg.Providers {
		if cfg.Providers[i].DisabledModels == nil {
			cfg.Providers[i].DisabledModels = []string{}
		}
		if cfg.Providers[i].CustomHeaders == nil {
			cfg.Providers[i].CustomHeaders = map[string]string{}
		}
		if cfg.Providers[i].APIKeys == nil {
			cfg.Providers[i].APIKeys = []ProviderKey{}
		}
		if cfg.Providers[i].ModelConfigs == nil {
			cfg.Providers[i].ModelConfigs = []ModelConfig{}
		}
		// 向后兼容：如果 APIKey 不为空但 APIKeys 为空，迁移到 APIKeys
		if cfg.Providers[i].APIKey != "" && len(cfg.Providers[i].APIKeys) == 0 {
			cfg.Providers[i].APIKeys = []ProviderKey{{
				ID:     generateID("pk"),
				Key:    cfg.Providers[i].APIKey,
				Name:   "默认",
				Status: "active",
			}}
		}
	}
	// 迁移旧的 JSON UsageLogs 到 SQLite
	if len(cfg.UsageLogs) > 0 {
		initDB()
		if db != nil {
			log.Printf("[migrate] 迁移 %d 条日志到 SQLite", len(cfg.UsageLogs))
			for _, l := range cfg.UsageLogs {
				dbAddUsageLog(l)
			}
			cfg.UsageLogs = nil
			saveConfig()
		}
	}
	config = &cfg
	return config
}

func saveConfig() {
	configMu.Lock()
	defer configMu.Unlock()

	data, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(cfgPath, data, 0600); err != nil {
		log.Printf("save config failed: %v", err)
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

// --- Provider CRUD ---

func listProviders() []Provider {
	cfg := loadConfig()
	return cfg.Providers
}

func getProvider(id string) *Provider {
	cfg := loadConfig()
	for i := range cfg.Providers {
		if cfg.Providers[i].ID == id {
			return &cfg.Providers[i]
		}
	}
	return nil
}

func getActiveProvider() *Provider {
	cfg := loadConfig()
	if cfg.Settings.ActiveProviderID != "" {
		for i := range cfg.Providers {
			if cfg.Providers[i].ID == cfg.Settings.ActiveProviderID && cfg.Providers[i].Status == "active" {
				return &cfg.Providers[i]
			}
		}
	}
	// 回退：找第一个 active 的
	for i := range cfg.Providers {
		if cfg.Providers[i].Status == "active" {
			return &cfg.Providers[i]
		}
	}
	return nil
}

func addProvider(p Provider) {
	cfg := loadConfig()
	ensureDisabledModelsInit(&p)
	cfg.Providers = append(cfg.Providers, p)
	saveConfig()
}

func updateProvider(p Provider) bool {
	cfg := loadConfig()
	for i := range cfg.Providers {
		if cfg.Providers[i].ID == p.ID {
			ensureDisabledModelsInit(&p)
			cfg.Providers[i] = p
			saveConfig()
			return true
		}
	}
	return false
}

func deleteProvider(id string) bool {
	cfg := loadConfig()
	for i := range cfg.Providers {
		if cfg.Providers[i].ID == id {
			cfg.Providers = append(cfg.Providers[:i], cfg.Providers[i+1:]...)
			if cfg.Settings.ActiveProviderID == id {
				cfg.Settings.ActiveProviderID = ""
			}
			saveConfig()
			return true
		}
	}
	return false
}

func setProviderDefaultModel(providerID, model string) bool {
	cfg := loadConfig()
	for i := range cfg.Providers {
		if cfg.Providers[i].ID == providerID {
			cfg.Providers[i].DefaultModel = model
			saveConfig()
			return true
		}
	}
	return false
}

func setActiveProvider(id string) bool {
	cfg := loadConfig()
	for i := range cfg.Providers {
		if cfg.Providers[i].ID == id {
			cfg.Settings.ActiveProviderID = id
			saveConfig()
			return true
		}
	}
	return false
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
	cfg := loadConfig()
	for i := range cfg.Providers {
		if cfg.Providers[i].ID != providerID {
			continue
		}
		disabled := false
		for j, m := range cfg.Providers[i].DisabledModels {
			if m == model {
				cfg.Providers[i].DisabledModels = append(cfg.Providers[i].DisabledModels[:j], cfg.Providers[i].DisabledModels[j+1:]...)
				disabled = true
				break
			}
		}
		if !disabled {
			cfg.Providers[i].DisabledModels = append(cfg.Providers[i].DisabledModels, model)
		}
		if cfg.Providers[i].DisabledModels == nil {
			cfg.Providers[i].DisabledModels = []string{}
		}
		saveConfig()
		return true
	}
	return false
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

// keyRotation 简单的内存轮询计数器
var keyRotation = map[string]int{}
var keyRotMu sync.Mutex

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
	keyRotMu.Lock()
	idx := keyRotation[p.ID]
	keyRotation[p.ID] = (idx + 1) % len(activeKeys)
	keyRotMu.Unlock()
	return activeKeys[idx]
}

// getModelConfig 获取 provider 下指定模型的上下文配置，没有则返回 nil
func getModelConfig(p *Provider, model string) *ModelConfig {
	for i := range p.ModelConfigs {
		if p.ModelConfigs[i].Model == model {
			return &p.ModelConfigs[i]
		}
	}
	return nil
}

// setModelConfig 设置或更新 provider 下指定模型的上下文配置
// 如果 inputLimit 和 outputLimit 都为空，则删除该配置
func setModelConfig(providerID string, mc ModelConfig) bool {
	cfg := loadConfig()
	for i := range cfg.Providers {
		if cfg.Providers[i].ID == providerID {
			for j := range cfg.Providers[i].ModelConfigs {
				if cfg.Providers[i].ModelConfigs[j].Model == mc.Model {
					if mc.InputLimit == "" && mc.OutputLimit == "" {
						// 删除
						cfg.Providers[i].ModelConfigs = append(cfg.Providers[i].ModelConfigs[:j], cfg.Providers[i].ModelConfigs[j+1:]...)
					} else {
						cfg.Providers[i].ModelConfigs[j] = mc
					}
					saveConfig()
					return true
				}
			}
			// 不存在且非空则新增
			if mc.InputLimit != "" || mc.OutputLimit != "" {
				cfg.Providers[i].ModelConfigs = append(cfg.Providers[i].ModelConfigs, mc)
			}
			saveConfig()
			return true
		}
	}
	return false
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
	cfg := loadConfig()
	return cfg.APIKeys
}

func addAPIKey(k APIKey) {
	cfg := loadConfig()
	cfg.APIKeys = append(cfg.APIKeys, k)
	saveConfig()
}

func deleteAPIKey(id string) bool {
	cfg := loadConfig()
	for i := range cfg.APIKeys {
		if cfg.APIKeys[i].ID == id {
			cfg.APIKeys = append(cfg.APIKeys[:i], cfg.APIKeys[i+1:]...)
			saveConfig()
			return true
		}
	}
	return false
}

func validateAPIKey(key string) bool {
	cfg := loadConfig()
	if len(cfg.APIKeys) == 0 {
		return true // 未配置 key 则允许无鉴权
	}
	for _, k := range cfg.APIKeys {
		if k.Key == key && k.Status == "active" {
			return true
		}
	}
	return false
}

// --- ModelAlias CRUD ---

func listAliases() []ModelAlias {
	cfg := loadConfig()
	// 填充 providerName
	for i := range cfg.Aliases {
		if p := getProvider(cfg.Aliases[i].ProviderID); p != nil {
			cfg.Aliases[i].ProviderName = p.Name
		}
	}
	return cfg.Aliases
}

func addAlias(a ModelAlias) {
	cfg := loadConfig()
	a.ID = generateID("alias")
	cfg.Aliases = append(cfg.Aliases, a)
	saveConfig()
}

func updateAlias(a ModelAlias) bool {
	cfg := loadConfig()
	for i := range cfg.Aliases {
		if cfg.Aliases[i].ID == a.ID {
			cfg.Aliases[i] = a
			saveConfig()
			return true
		}
	}
	return false
}

func deleteAlias(id string) bool {
	cfg := loadConfig()
	for i := range cfg.Aliases {
		if cfg.Aliases[i].ID == id {
			cfg.Aliases = append(cfg.Aliases[:i], cfg.Aliases[i+1:]...)
			saveConfig()
			return true
		}
	}
	return false
}

// getAliasByModel 根据模型名查找别名
func getAliasByModel(model string) *ModelAlias {
	cfg := loadConfig()
	for i := range cfg.Aliases {
		if cfg.Aliases[i].Name == model {
			return &cfg.Aliases[i]
		}
	}
	return nil
}

// --- Usage ---

func addUsageLog(logEntry UsageLog) {
	// 写入 SQLite
	dbAddUsageLog(logEntry)
	// 更新 provider 统计（仍用 config）
	cfg := loadConfig()
	for i := range cfg.Providers {
		if cfg.Providers[i].ID == logEntry.ProviderID {
			cfg.Providers[i].UsageCount++
			cfg.Providers[i].TotalTokens += int64(logEntry.TotalTokens)
			break
		}
	}
	saveConfig()
}

func getUsageStats() map[string]any {
	return dbGetUsageStats()
}

func getRecentLogs(page, pageSize int) ([]UsageLog, int) {
	return dbGetRecentLogs(page, pageSize)
}

func clearLogs() {
	dbClearLogs()
}

// --- Settings ---

func getSettings() Settings {
	cfg := loadConfig()
	return cfg.Settings
}

func updateSettings(s Settings) {
	cfg := loadConfig()
	cfg.Settings.ActiveProviderID = s.ActiveProviderID
	cfg.Settings.DefaultModel = s.DefaultModel
	if len(s.InputPresets) > 0 {
		cfg.Settings.InputPresets = s.InputPresets
	}
	if len(s.OutputPresets) > 0 {
		cfg.Settings.OutputPresets = s.OutputPresets
	}
	saveConfig()
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
