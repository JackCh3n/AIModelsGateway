package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// registerAdminRoutes 注册管理后台路由
func registerAdminRoutes(mux *http.ServeMux) {
	// 前端页面
	mux.HandleFunc("/admin/", adminPageHandler)
	mux.HandleFunc("/admin", adminPageHandler)

	// Provider CRUD
	mux.HandleFunc("/admin/api/providers", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			writeJSON(w, http.StatusOK, listProviders())
		case "POST":
			var p Provider
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			p.ID = generateID("prov")
			if p.Status == "" {
				p.Status = "active"
			}
			if p.Format == "" {
				p.Format = "openai"
			}
			if p.Models == nil {
				p.Models = []string{}
			}
			if p.DisabledModels == nil {
				p.DisabledModels = []string{}
			}
			if p.CustomHeaders == nil {
				p.CustomHeaders = map[string]string{}
			}
			if p.ModelConfigs == nil {
				p.ModelConfigs = []ModelConfig{}
			}
			addProvider(p)
			writeJSON(w, http.StatusOK, p)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}))

	mux.HandleFunc("/admin/api/providers/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/admin/api/providers/")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
			return
		}

		switch r.Method {
		case "PUT":
			var p Provider
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			p.ID = id
			if !updateProvider(p) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
				return
			}
			writeJSON(w, http.StatusOK, p)
		case "DELETE":
			if !deleteProvider(id) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}))

	// 设置活跃站点
	mux.HandleFunc("/admin/api/providers/active/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/admin/api/providers/active/")
		if r.Method != "PUT" || id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "PUT with id required"})
			return
		}
		if !setActiveProvider(id) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))

	// 切换模型启用/禁用: PUT /admin/api/providers/{id}/models/toggle?model=xxx
	mux.HandleFunc("/admin/api/providers/models/toggle/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "PUT required"})
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/admin/api/providers/models/toggle/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) < 1 || parts[0] == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider id required"})
			return
		}
		providerID := parts[0]
		model := r.URL.Query().Get("model")
		if model == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "model query param required"})
			return
		}
		if !toggleModelEnabled(providerID, model) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
			return
		}
		p := getProvider(providerID)
		writeJSON(w, http.StatusOK, map[string]any{
			"status":         "ok",
			"providerId":     providerID,
			"model":          model,
			"enabled":        isModelEnabled(p, model),
			"disabledModels": p.DisabledModels,
		})
	}))

	// 设置站点默认模型: PUT /admin/api/providers/{id}/models/default?model=xxx
	mux.HandleFunc("/admin/api/providers/models/default/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "PUT required"})
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/admin/api/providers/models/default/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) < 1 || parts[0] == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider id required"})
			return
		}
		providerID := parts[0]
		model := r.URL.Query().Get("model")
		if model == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "model query param required"})
			return
		}
		if !setProviderDefaultModel(providerID, model) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "ok",
			"providerId":   providerID,
			"defaultModel": model,
		})
	}))

	// 打卡签到: POST /admin/api/providers/{id}/checkin
	mux.HandleFunc("/admin/api/providers/checkin/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/admin/api/providers/checkin/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) < 1 || parts[0] == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider id required"})
			return
		}
		ok, msg := checkinProvider(parts[0])
		writeJSON(w, http.StatusOK, map[string]any{
			"success": ok,
			"message": truncate(msg, 500),
		})
	}))

	// 模型上下文配置: PUT /admin/api/providers/{id}/models/config?model=xxx
	mux.HandleFunc("/admin/api/providers/models/config/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "PUT required"})
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/admin/api/providers/models/config/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) < 1 || parts[0] == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider id required"})
			return
		}
		providerID := parts[0]
		model := r.URL.Query().Get("model")
		if model == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "model query param required"})
			return
		}
		var mc ModelConfig
		if err := json.NewDecoder(r.Body).Decode(&mc); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		mc.Model = model
		if !setModelConfig(providerID, mc) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"providerId": providerID,
			"config":     mc,
		})
	}))

	// API Key CRUD
	mux.HandleFunc("/admin/api/keys", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			writeJSON(w, http.StatusOK, listAPIKeys())
		case "POST":
			var req struct {
				Name string `json:"name"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			k := APIKey{
				ID:        generateID("key"),
				Key:       generateAPIKey(),
				Name:      req.Name,
				Status:    "active",
				CreatedAt: time.Now(),
			}
			addAPIKey(k)
			writeJSON(w, http.StatusOK, k)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}))

	mux.HandleFunc("/admin/api/keys/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/admin/api/keys/")
		if r.Method != "DELETE" || id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "DELETE with id required"})
			return
		}
		if !deleteAPIKey(id) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}))

	// 统计
	mux.HandleFunc("/admin/api/stats", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, getUsageStats())
	}))

	mux.HandleFunc("/admin/api/logs", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if pageSize <= 0 {
			pageSize = 50
		}
		logs, total := getRecentLogs(page, pageSize)
		writeJSON(w, http.StatusOK, map[string]any{
			"logs":     logs,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
			"pages":    (total + pageSize - 1) / pageSize,
		})
	}))

	// 清空日志
	mux.HandleFunc("/admin/api/logs/clear", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		clearLogs()
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))

	// 设置
	mux.HandleFunc("/admin/api/settings", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			writeJSON(w, http.StatusOK, getSettings())
		case "PUT":
			var s Settings
			if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			updateSettings(s)
			writeJSON(w, http.StatusOK, s)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}))

	// 测试站点 key 和模型
	mux.HandleFunc("/admin/api/test", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		var req struct {
			BaseURL       string            `json:"baseUrl"`
			APIKey        string            `json:"apiKey"`
			Format        string            `json:"format"`
			Model         string            `json:"model"`
			CustomHeaders map[string]string `json:"customHeaders"`
			ProxyEnabled  bool              `json:"proxyEnabled"`
			ProxyType     string            `json:"proxyType"`
			ProxyAddr     string            `json:"proxyAddr"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		result := testProvider(req.BaseURL, req.APIKey, req.Format, req.Model, req.CustomHeaders, req.ProxyEnabled, req.ProxyType, req.ProxyAddr)
		writeJSON(w, http.StatusOK, result)
	}))

	// 按 provider+model 测试（从已保存的站点配置读取）
	mux.HandleFunc("/admin/api/providers/test/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/admin/api/providers/test/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) < 1 || parts[0] == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider id required"})
			return
		}
		p := getProvider(parts[0])
		if p == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
			return
		}
		model := r.URL.Query().Get("model")
		if model == "" {
			model = getSettings().DefaultModel
		}
		result := testProvider(p.BaseURL, pickAPIKey(p), p.Format, model, p.CustomHeaders, p.ProxyEnabled, p.ProxyType, p.ProxyAddr)
		// 记录测试用量
		if success, ok := result["success"].(bool); ok && success {
			input, _ := result["inputTokens"].(int)
			output, _ := result["outputTokens"].(int)
			logEntry := newUsageLog(p.ID, p.Name, model, input, output, p.Format)
			addUsageLog(logEntry)
		}
		writeJSON(w, http.StatusOK, result)
	}))

	// --- 模型别名 CRUD ---
	mux.HandleFunc("/admin/api/aliases", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			writeJSON(w, http.StatusOK, listAliases())
		case "POST":
			var a ModelAlias
			if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			addAlias(a)
			writeJSON(w, http.StatusOK, a)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}))

	mux.HandleFunc("/admin/api/aliases/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/admin/api/aliases/")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
			return
		}
		switch r.Method {
		case "PUT":
			var a ModelAlias
			if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			a.ID = id
			if !updateAlias(a) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "alias not found"})
				return
			}
			writeJSON(w, http.StatusOK, a)
		case "DELETE":
			if !deleteAlias(id) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "alias not found"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}))

	// --- 中转站导出/导入 ---
	// 导出单个: GET /admin/api/providers/export/{id}
	mux.HandleFunc("/admin/api/providers/export/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/admin/api/providers/export/")
		p := getProvider(id)
		if p == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "provider not found"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=provider_"+id+".json")
		json.NewEncoder(w).Encode(p)
	}))

	// 导入: POST /admin/api/providers/import
	mux.HandleFunc("/admin/api/providers/import", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		var p Provider
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		// 重新生成 ID，避免冲突
		p.ID = generateID("prov")
		if p.Status == "" {
			p.Status = "active"
		}
		if p.Models == nil {
			p.Models = []string{}
		}
		if p.DisabledModels == nil {
			p.DisabledModels = []string{}
		}
		addProvider(p)
		writeJSON(w, http.StatusOK, p)
	}))

	// 导入 OpenCode 配置: POST /admin/api/providers/import/opencode
	mux.HandleFunc("/admin/api/providers/import/opencode", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		p, err := parseOpenCodeConfig(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		addProvider(*p)
		writeJSON(w, http.StatusOK, p)
	}))

	// 导入 newapi 连接配置: POST /admin/api/providers/import/conn
	mux.HandleFunc("/admin/api/providers/import/conn", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		var raw struct {
			Type string `json:"_type"`
			Key  string `json:"key"`
			URL  string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if raw.Type != "newapi_channel_conn" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "不支持的格式: " + raw.Type})
			return
		}
		baseURL := strings.Trim(raw.URL, "`")
		if baseURL == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "缺少 url"})
			return
		}
		// OpenAI 格式：若 url 未包含 /v1 则追加
		if !strings.Contains(baseURL, "/v1") {
			baseURL = strings.TrimSuffix(baseURL, "/") + "/v1"
		}
		p := Provider{
			ID:             generateID("prov"),
			Name:           hostnameOf(baseURL),
			BaseURL:        baseURL,
			APIKey:         raw.Key,
			APIKeys:        []ProviderKey{{ID: generateID("pk"), Key: raw.Key, Name: "导入", Status: "active"}},
			Format:         "openai",
			Models:         []string{},
			DisabledModels: []string{},
			Status:         "active",
		}
		addProvider(p)
		writeJSON(w, http.StatusOK, p)
	}))

	// 一键获取模型: POST /admin/api/providers/fetch-models
	mux.HandleFunc("/admin/api/providers/fetch-models", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
			return
		}
		var req struct {
			BaseURL       string            `json:"baseUrl"`
			APIKey        string            `json:"apiKey"`
			Format        string            `json:"format"`
			CustomHeaders map[string]string `json:"customHeaders"`
			ProxyEnabled  bool              `json:"proxyEnabled"`
			ProxyType     string            `json:"proxyType"`
			ProxyAddr     string            `json:"proxyAddr"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if req.BaseURL == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "baseUrl required"})
			return
		}
		result := fetchProviderModels(req.BaseURL, req.APIKey, req.Format, req.CustomHeaders, req.ProxyEnabled, req.ProxyType, req.ProxyAddr)
		writeJSON(w, http.StatusOK, result)
	}))
}

// fetchProviderModels 通过 /v1/models 接口获取模型列表
func fetchProviderModels(baseURL, apiKey, format string, customHeaders map[string]string, proxyEnabled bool, proxyType, proxyAddr string) map[string]any {
	upstreamURL := strings.TrimSuffix(baseURL, "/") + "/models"

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if format == "anthropic" {
		if apiKey != "" {
			headers["x-api-key"] = apiKey
			headers["anthropic-version"] = "2023-06-01"
		}
	} else {
		if apiKey != "" {
			headers["Authorization"] = "Bearer " + apiKey
		}
	}
	for k, v := range customHeaders {
		headers[k] = v
	}

	req, err := newHTTPRequest("GET", upstreamURL, nil, headers)
	if err != nil {
		return map[string]any{"success": false, "error": "create request: " + err.Error()}
	}

	proxyProvider := &Provider{ProxyEnabled: proxyEnabled, ProxyType: proxyType, ProxyAddr: proxyAddr}
	resp, err := getHTTPClient(proxyProvider).Do(req)
	if err != nil {
		return map[string]any{"success": false, "error": "request failed: " + err.Error()}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	respStr := string(respBody)

	if resp.StatusCode != 200 {
		return map[string]any{
			"success": false,
			"status":  resp.StatusCode,
			"error":   truncate(respStr, 500),
		}
	}

	var respObj map[string]any
	if err := json.Unmarshal(respBody, &respObj); err != nil {
		// 可能是裸数组 [{"id":"model"}, ...] 或 ["model1", ...]
		var respArr []any
		if err2 := json.Unmarshal(respBody, &respArr); err2 != nil {
			return map[string]any{"success": false, "error": "parse response: " + err.Error()}
		}
		respObj = map[string]any{"data": respArr}
	}

	// 标准 OpenAI 格式: {"data": [{"id": "model-name", ...}, ...]}
	var models []string
	seen := map[string]bool{}
	extractModel := func(item any) {
		switch v := item.(type) {
		case string:
			if v != "" && !seen[v] {
				seen[v] = true
				models = append(models, v)
			}
		case map[string]any:
			if id, ok := v["id"].(string); ok && id != "" && !seen[id] {
				seen[id] = true
				models = append(models, id)
			}
		}
	}

	if data, ok := respObj["data"].([]any); ok {
		for _, item := range data {
			extractModel(item)
		}
	}

	// 如果没有 data 数组，尝试兼容 {"models": [...]}
	if len(models) == 0 {
		if raw, ok := respObj["models"].([]any); ok {
			for _, item := range raw {
				extractModel(item)
			}
		}
	}

	log.Printf("  fetch models: %s -> %d, got %d models", upstreamURL, resp.StatusCode, len(models))

	return map[string]any{
		"success": true,
		"status":  resp.StatusCode,
		"models":  models,
		"count":   len(models),
		"raw":     truncate(respStr, 1000),
	}
}

// parseOpenCodeConfig 解析 OpenCode 格式配置并生成 Provider
func parseOpenCodeConfig(rd io.Reader) (*Provider, error) {
	var raw map[string]any
	if err := json.NewDecoder(rd).Decode(&raw); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %v", err)
	}
	prov, ok := raw["provider"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("缺少 provider 字段")
	}
	// 支持 openai / anthropic
	var format, baseURL, apiKey string
	var modelsMap map[string]any
	if openai, ok := prov["openai"].(map[string]any); ok {
		format = "openai"
		if opts, ok := openai["options"].(map[string]any); ok {
			baseURL, _ = opts["baseURL"].(string)
			apiKey, _ = opts["apiKey"].(string)
		}
		modelsMap, _ = openai["models"].(map[string]any)
	} else if anthropic, ok := prov["anthropic"].(map[string]any); ok {
		format = "anthropic"
		if opts, ok := anthropic["options"].(map[string]any); ok {
			baseURL, _ = opts["baseURL"].(string)
			apiKey, _ = opts["apiKey"].(string)
		}
		modelsMap, _ = anthropic["models"].(map[string]any)
	} else {
		return nil, fmt.Errorf("provider 下未找到 openai 或 anthropic 配置")
	}
	// 清理 baseURL 反引号
	baseURL = strings.Trim(baseURL, "`")
	// 提取模型 + 上下文配置
	models := []string{}
	modelConfigs := []ModelConfig{}
	for name, mraw := range modelsMap {
		models = append(models, name)
		m, _ := mraw.(map[string]any)
		limit, _ := m["limit"].(map[string]any)
		inputLimit := tokenToPreset(int64(toInt(limit["context"])))
		outputLimit := tokenToPreset(int64(toInt(limit["output"])))
		mc := ModelConfig{Model: name}
		if inputLimit != "" {
			mc.InputLimit = inputLimit
		}
		if outputLimit != "" {
			mc.OutputLimit = outputLimit
		}
		if mc.InputLimit != "" || mc.OutputLimit != "" {
			modelConfigs = append(modelConfigs, mc)
		}
	}
	if baseURL == "" {
		return nil, fmt.Errorf("缺少 baseURL")
	}
	name := hostnameOf(baseURL)
	p := Provider{
		ID:             generateID("prov"),
		Name:           name,
		BaseURL:        baseURL,
		APIKey:         apiKey,
		APIKeys:        []ProviderKey{{ID: generateID("pk"), Key: apiKey, Name: "导入", Status: "active"}},
		Format:         format,
		Models:         models,
		DisabledModels: []string{},
		ModelConfigs:   modelConfigs,
		Status:         "active",
	}
	return &p, nil
}

// toInt 安全转 int
func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

// tokenToPreset token 数转预算预设标签
func tokenToPreset(n int64) string {
	if n <= 0 {
		return ""
	}
	k := n / 1000
	if k >= 1000 {
		return fmt.Sprintf("%dM", k/1000)
	}
	return fmt.Sprintf("%dK", k)
}

// hostnameOf 从 URL 提取站点名
func hostnameOf(url string) string {
	u := strings.TrimPrefix(url, "https://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimSuffix(u, "/v1")
	u = strings.TrimSuffix(u, "/")
	if idx := strings.Index(u, "/"); idx > 0 {
		u = u[:idx]
	}
	return u
}

// testProvider 测试中转站 key 和模型可用性（真实对话测试）
func testProvider(baseURL, apiKey, format, model string, customHeaders map[string]string, proxyEnabled bool, proxyType, proxyAddr string) map[string]any {
	if model == "" {
		model = "gpt-4o-mini"
	}

	testMessage := "Hi, reply with just 'ok'"

	var reqBody map[string]any
	var upstreamURL string
	var headers map[string]string

	if format == "anthropic" {
		upstreamURL = strings.TrimSuffix(baseURL, "/") + "/messages"
		reqBody = map[string]any{
			"model":      model,
			"max_tokens": 100,
			"messages":   []any{map[string]any{"role": "user", "content": testMessage}},
		}
		headers = map[string]string{
			"Content-Type":      "application/json",
			"x-api-key":         apiKey,
			"anthropic-version": "2023-06-01",
		}
	} else {
		upstreamURL = strings.TrimSuffix(baseURL, "/") + "/chat/completions"
		reqBody = map[string]any{
			"model":       model,
			"max_tokens":  100,
			"messages":    []any{map[string]any{"role": "user", "content": testMessage}},
			"temperature": 0,
		}
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + apiKey,
		}
	}

	// 合并自定义请求头（覆盖同名默认头）
	for k, v := range customHeaders {
		headers[k] = v
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req, err := newHTTPRequest("POST", upstreamURL, bodyJSON, headers)
	if err != nil {
		return map[string]any{"success": false, "error": "create request: " + err.Error()}
	}

	// 创建带代理的 client
	proxyProvider := &Provider{ProxyEnabled: proxyEnabled, ProxyType: proxyType, ProxyAddr: proxyAddr}
	resp, err := getHTTPClient(proxyProvider).Do(req)
	if err != nil {
		return map[string]any{"success": false, "error": "request failed: " + err.Error()}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	respStr := string(respBody)

	if resp.StatusCode != 200 {
		return map[string]any{
			"success":     false,
			"status":      resp.StatusCode,
			"error":       truncate(respStr, 500),
			"testUrl":     upstreamURL,
			"testMessage": testMessage,
			"reqHeaders":  safeHeaders(headers), // 隐藏 Authorization/x-api-key，避免泄露密钥
		}
	}

	// 尝试解析响应内容
	var respObj map[string]any
	json.Unmarshal(respBody, &respObj)

	content := ""
	if format == "anthropic" {
		if contentArr, ok := respObj["content"].([]any); ok && len(contentArr) > 0 {
			if blk, ok := contentArr[0].(map[string]any); ok {
				if t, ok := blk["text"].(string); ok {
					content = t
				}
			}
		}
	} else {
		content = fmt.Sprintf("%v", getNested(respObj, "choices", 0, "message", "content"))
	}

	// 提取用量
	input, output := extractUsage(respObj, format)

	log.Printf("  test: %s %s -> %d content=%s tokens=%d/%d", baseURL, model, resp.StatusCode, truncate(content, 50), input, output)

	return map[string]any{
		"success":      true,
		"status":       resp.StatusCode,
		"content":      truncate(content, 200),
		"raw":          truncate(respStr, 500),
		"testUrl":      upstreamURL,
		"testMessage":  testMessage,
		"reqHeaders":   safeHeaders(headers), // 隐藏 Authorization/x-api-key，避免泄露密钥
		"inputTokens":  input,
		"outputTokens": output,
	}
}

// safeHeaders 隐藏请求头中的敏感字段（Authorization / x-api-key），避免在接口响应中泄露密钥
func safeHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		lower := strings.ToLower(k)
		switch lower {
		case "authorization", "x-api-key", "api-key":
			if v != "" {
				out[k] = v[:min(len(v), 8)] + "..."
			} else {
				out[k] = ""
			}
		default:
			out[k] = v
		}
	}
	return out
}

func newHTTPRequest(method, url string, body []byte, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req, nil
}
