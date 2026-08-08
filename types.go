package main

import "time"

// Provider 中转站/官方站配置
type Provider struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	BaseURL        string            `json:"baseUrl"`        // e.g. https://api.openai.com/v1
	APIKey         string            `json:"apiKey"`         // 主 API Key（向后兼容）
	APIKeys        []ProviderKey     `json:"apiKeys"`        // 多 Key 轮询
	Format         string            `json:"format"`         // "openai" 或 "anthropic"
	Models         []string          `json:"models"`         // 支持的模型列表
	DisabledModels []string          `json:"disabledModels"` // 被禁用的模型
	DefaultModel   string            `json:"defaultModel"`   // 站点默认模型（model=all时使用）
	ModelConfigs   []ModelConfig     `json:"modelConfigs"`   // 每个模型的上下文配置
	CustomHeaders  map[string]string `json:"customHeaders"`  // 自定义请求头
	CheckinURL     string            `json:"checkinUrl"`     // 打卡签到地址
	LastCheckin    time.Time         `json:"lastCheckin"`    // 上次打卡时间
	Status         string            `json:"status"`         // active, disabled
	UsageCount     int64             `json:"usageCount"`
	TotalTokens    int64             `json:"totalTokens"`
	CreatedAt      time.Time         `json:"createdAt"`
	ProxyEnabled   bool              `json:"proxyEnabled"` // 是否启用代理
	ProxyType      string            `json:"proxyType"`    // http, https, socks5
	ProxyAddr      string            `json:"proxyAddr"`    // 代理地址，如 127.0.0.1:7890
}

// ModelConfig 模型上下文配置
type ModelConfig struct {
	Model       string `json:"model"`       // 模型名
	InputLimit  string `json:"inputLimit"`  // 输入上下文预算: 32K/64K/128K/256K/1M
	OutputLimit string `json:"outputLimit"` // 输出预算: 8K/16K/32K/64K/128K/256K
}

// ProviderKey 站点的多 Key
type ProviderKey struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Name   string `json:"name"`   // 备注
	Status string `json:"status"` // active, disabled
}

// APIKey 网关访问密钥
type APIKey struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Status    string    `json:"status"` // active, disabled
}

// UsageLog Token 用量记录
type UsageLog struct {
	ID           string    `json:"id"`
	ProviderID   string    `json:"providerId"`
	ProviderName string    `json:"providerName"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"inputTokens"`
	OutputTokens int       `json:"outputTokens"`
	TotalTokens  int       `json:"totalTokens"`
	Timestamp    time.Time `json:"timestamp"`
	ClientFormat string    `json:"clientFormat"` // openai 或 anthropic
}

// Settings 全局设置
type Settings struct {
	ActiveProviderID string   `json:"activeProviderId"`
	DefaultModel     string   `json:"defaultModel"`
	InputPresets     []string `json:"inputPresets"`  // 输入上下文预算预设
	OutputPresets    []string `json:"outputPresets"` // 输出预算预设
	// 全局自定义 User-Agent：转发到上游时使用的请求头，覆盖 Go 默认的 go-http-client/1.1
	UserAgent string `json:"userAgent"`
	// 狂暴模式：启用 Redis 统计 + 加大连接池，支持更高并发
	RageMode      bool   `json:"rageMode"`
	RedisAddr     string `json:"redisAddr"`     // Redis 地址，如 127.0.0.1:6379
	RedisPassword string `json:"redisPassword"` // Redis 密码
	RedisDB       int    `json:"redisDb"`       // Redis DB 编号
}

// ModelAlias 模型路由别名
// 客户端用固定模型名调用，网关自动路由到指定站点的指定模型
type ModelAlias struct {
	ID           string `json:"id"`
	Name         string `json:"name"`         // 别名，客户端请求时用的模型名
	ProviderID   string `json:"providerId"`   // 实际使用的站点 ID
	ProviderName string `json:"providerName"` // 显示用
	Model        string `json:"model"`        // 实际使用的模型
}

// FailoverEntry 主备路由中的一个站点（按优先级 1/2/3 排列）
type FailoverEntry struct {
	Order      int    `json:"order"`      // 优先级：1 主站，2 备站，3 备站
	ProviderID string `json:"providerId"` // 站点 ID
	Model      string `json:"model"`      // 该站点使用的模型
}

// FailoverRoute 主备路由
// 客户端用固定模型名调用，主站连续失败（3 次重试后仍失败）时顺延下一个站点，
// 最多支持 3 个站点（1 主 2 备）。仅当请求模型命中该路由时才启用此效果。
type FailoverRoute struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`    // 客户端请求时用的模型名
	Entries []FailoverEntry `json:"entries"` // 按优先级排序，最多 3 个
}

// Config 持久化配置
type Config struct {
	Providers []Provider     `json:"providers"`
	APIKeys   []APIKey       `json:"apiKeys"`
	Aliases   []ModelAlias   `json:"aliases"`
	Failovers []FailoverRoute `json:"failovers"` // 主备路由
	Settings  Settings       `json:"settings"`
	UsageLogs []UsageLog     `json:"usageLogs,omitempty"` // 已迁移到SQLite，保留字段向后兼容
	idx       configIndex    `json:"-"`                   // 预计算索引，不序列化
}

// configIndex 预计算索引，将热路径 O(n) 查找优化为 O(1)
type configIndex struct {
	apiKeySet         map[string]bool        // API Key 验证：key -> active
	aliasMap          map[string]ModelAlias  // 别名查找：aliasName -> alias
	failoverMap       map[string]FailoverRoute // 主备路由查找：name -> route
	providerMap       map[string]int         // 站点查找：providerID -> slice index
	activeProviderIdx int                     // 活跃站点在 Providers 中的下标，-1 表示无
}
