package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
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
		Settings:  Settings{DefaultModel: "gpt-4o-mini"},
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
	if cfg.UsageLogs == nil {
		cfg.UsageLogs = []UsageLog{}
	}
	if cfg.Settings.DefaultModel == "" {
		cfg.Settings.DefaultModel = "gpt-4o-mini"
	}
	// 确保每个 provider 的 DisabledModels 已初始化
	for i := range cfg.Providers {
		if cfg.Providers[i].DisabledModels == nil {
			cfg.Providers[i].DisabledModels = []string{}
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

// --- Usage ---

func addUsageLog(logEntry UsageLog) {
	cfg := loadConfig()
	cfg.UsageLogs = append(cfg.UsageLogs, logEntry)
	// 更新 provider 统计
	for i := range cfg.Providers {
		if cfg.Providers[i].ID == logEntry.ProviderID {
			cfg.Providers[i].UsageCount++
			cfg.Providers[i].TotalTokens += int64(logEntry.TotalTokens)
			break
		}
	}
	// 保留最近 10000 条日志
	if len(cfg.UsageLogs) > 10000 {
		cfg.UsageLogs = cfg.UsageLogs[len(cfg.UsageLogs)-10000:]
	}
	saveConfig()
}

func getUsageStats() map[string]any {
	cfg := loadConfig()
	// 按站点统计
	byProvider := map[string]map[string]int64{}
	// 按模型统计
	byModel := map[string]map[string]int64{}
	// 按日期统计
	byDate := map[string]map[string]int64{}

	totalInput := 0
	totalOutput := 0
	totalAll := 0

	for _, log := range cfg.UsageLogs {
		totalInput += log.InputTokens
		totalOutput += log.OutputTokens
		totalAll += log.TotalTokens

		date := log.Timestamp.Format("2006-01-02")

		if byProvider[log.ProviderName] == nil {
			byProvider[log.ProviderName] = map[string]int64{"input": 0, "output": 0, "total": 0, "count": 0}
		}
		byProvider[log.ProviderName]["input"] += int64(log.InputTokens)
		byProvider[log.ProviderName]["output"] += int64(log.OutputTokens)
		byProvider[log.ProviderName]["total"] += int64(log.TotalTokens)
		byProvider[log.ProviderName]["count"]++

		if byModel[log.Model] == nil {
			byModel[log.Model] = map[string]int64{"input": 0, "output": 0, "total": 0, "count": 0}
		}
		byModel[log.Model]["input"] += int64(log.InputTokens)
		byModel[log.Model]["output"] += int64(log.OutputTokens)
		byModel[log.Model]["total"] += int64(log.TotalTokens)
		byModel[log.Model]["count"]++

		if byDate[date] == nil {
			byDate[date] = map[string]int64{"input": 0, "output": 0, "total": 0, "count": 0}
		}
		byDate[date]["input"] += int64(log.InputTokens)
		byDate[date]["output"] += int64(log.OutputTokens)
		byDate[date]["total"] += int64(log.TotalTokens)
		byDate[date]["count"]++
	}

	return map[string]any{
		"totalInput":  totalInput,
		"totalOutput": totalOutput,
		"totalTokens": totalAll,
		"totalReqs":   len(cfg.UsageLogs),
		"byProvider":  byProvider,
		"byModel":     byModel,
		"byDate":      byDate,
	}
}

func getRecentLogs(limit int) []UsageLog {
	cfg := loadConfig()
	if limit <= 0 || limit > len(cfg.UsageLogs) {
		limit = len(cfg.UsageLogs)
	}
	start := len(cfg.UsageLogs) - limit
	if start < 0 {
		start = 0
	}
	// 反序返回（最新在前）
	result := make([]UsageLog, 0, limit)
	for i := len(cfg.UsageLogs) - 1; i >= start; i-- {
		result = append(result, cfg.UsageLogs[i])
	}
	return result
}

// --- Settings ---

func getSettings() Settings {
	cfg := loadConfig()
	return cfg.Settings
}

func updateSettings(s Settings) {
	cfg := loadConfig()
	cfg.Settings = s
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
