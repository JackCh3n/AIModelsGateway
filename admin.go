package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
		writeJSON(w, http.StatusOK, getRecentLogs(100))
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
			BaseURL string `json:"baseUrl"`
			APIKey  string `json:"apiKey"`
			Format  string `json:"format"`
			Model   string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		result := testProvider(req.BaseURL, req.APIKey, req.Format, req.Model)
		writeJSON(w, http.StatusOK, result)
	}))
}

// testProvider 测试中转站 key 和模型可用性
func testProvider(baseURL, apiKey, format, model string) map[string]any {
	if model == "" {
		model = "gpt-4o-mini"
	}

	var reqBody map[string]any
	var upstreamURL string
	var headers map[string]string

	if format == "anthropic" {
		upstreamURL = strings.TrimSuffix(baseURL, "/") + "/messages"
		reqBody = map[string]any{
			"model":      model,
			"max_tokens": 100,
			"messages":   []any{map[string]any{"role": "user", "content": "Hi, reply with just 'ok'"}},
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
			"messages":    []any{map[string]any{"role": "user", "content": "Hi, reply with just 'ok'"}},
			"temperature": 0,
		}
		headers = map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + apiKey,
		}
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req, err := newHTTPRequest("POST", upstreamURL, bodyJSON, headers)
	if err != nil {
		return map[string]any{"success": false, "error": "create request: " + err.Error()}
	}

	resp, err := httpClient.Do(req)
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

	log.Printf("  test: %s %s -> %d content=%s", baseURL, model, resp.StatusCode, truncate(content, 50))

	return map[string]any{
		"success": true,
		"status":  resp.StatusCode,
		"content": truncate(content, 200),
		"raw":     truncate(respStr, 500),
	}
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
