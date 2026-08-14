package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestMetricsStatsEndToEnd 端到端验证：请求经网关转发（mock 上游返回含缓存命中的 usage），
// 强制刷盘后统计接口应输出 平均首Token延迟/平均输出速度/缓存命中率，日志接口应带 ttftMs/durationMs。
func TestMetricsStatsEndToEnd(t *testing.T) {
	loadConfig() // 先触发 configOnce，避免覆盖测试配置

	// mock 上游：非流式响应，usage 含 DeepSeek 风格缓存命中字段。
	// 加 30ms 延迟模拟真实网络/生成耗时，避免本地毫秒取整为 0 无法验证计时。
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		time.Sleep(30 * time.Millisecond)
		resp := map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "m1",
			"choices": []any{map[string]any{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": "你好"},
				"finish_reason": "stop",
			}},
			"usage": map[string]any{
				"prompt_tokens":           float64(100),
				"completion_tokens":       float64(20),
				"total_tokens":            float64(120),
				"prompt_cache_hit_tokens": float64(70),
				"prompt_cache_miss_tokens": float64(30),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer up.Close()

	cfg := &Config{
		Providers: []Provider{{
			ID: "p1", Name: "mock", BaseURL: up.URL, Format: "openai",
			Status: "active", Models: []string{"m1"},
			APIKeys: []ProviderKey{{ID: "k1", Key: "sk-test", Status: "active"}},
		}},
		APIKeys:  []APIKey{},
		Settings: Settings{DefaultModel: "all", FailoverTimeout: 60},
	}
	cfg.idx = buildIndex(cfg)
	configPtr.Store(cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(w, r, "openai", "")
	}))
	registerAdminRoutes(mux)
	gw := httptest.NewServer(adminAuth(mux))
	defer gw.Close()

	// 发 3 个请求
	for i := 0; i < 3; i++ {
		body, _ := json.Marshal(map[string]any{
			"model":    "m1",
			"messages": []any{map[string]any{"role": "user", "content": "hi"}},
		})
		resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("req %d: %v", i, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("req %d: HTTP %d", i, resp.StatusCode)
		}
	}
	flushUsageLogs() // 强制刷盘（正常由后台 worker 每 1s 刷）

	// 统计接口
	sRes, err := http.Get(gw.URL + "/admin/api/stats")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	var stats map[string]any
	json.NewDecoder(sRes.Body).Decode(&stats)
	sRes.Body.Close()

	if stats["totalReqs"] != float64(3) {
		t.Errorf("totalReqs = %v, want 3", stats["totalReqs"])
	}
	avgTTFT, _ := stats["avgTTFTMs"].(float64)
	if avgTTFT <= 0 {
		t.Errorf("avgTTFTMs = %v, want > 0", avgTTFT)
	}
	avgSpeed, _ := stats["avgOutputSpeed"].(float64)
	if avgSpeed <= 0 {
		t.Errorf("avgOutputSpeed = %v, want > 0", avgSpeed)
	}
	cacheRate, _ := stats["cacheHitRate"].(float64)
	if cacheRate < 0.69 || cacheRate > 0.71 {
		t.Errorf("cacheHitRate = %v, want ~0.70 (70/100)", cacheRate)
	}
	t.Logf("stats: avgTTFTMs=%.0f avgOutputSpeed=%.0f cacheHitRate=%.2f", avgTTFT, avgSpeed, cacheRate)

	// 日志接口
	lRes, err := http.Get(gw.URL + "/admin/api/logs?page=1&pageSize=5")
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	var ldata map[string]any
	json.NewDecoder(lRes.Body).Decode(&ldata)
	lRes.Body.Close()
	logs, _ := ldata["logs"].([]any)
	if len(logs) == 0 {
		t.Fatal("logs 为空")
	}
	first := logs[0].(map[string]any)
	if first["ttftMs"] == nil || first["ttftMs"] == float64(0) {
		t.Errorf("log ttftMs 缺失: %v", first["ttftMs"])
	}
	if first["cacheHit"] != float64(70) {
		t.Errorf("log cacheHit = %v, want 70", first["cacheHit"])
	}
	if first["cacheMiss"] != float64(30) {
		t.Errorf("log cacheMiss = %v, want 30", first["cacheMiss"])
	}
	t.Logf("log sample: ttft=%v duration=%v cacheHit=%v cacheMiss=%v", first["ttftMs"], first["durationMs"], first["cacheHit"], first["cacheMiss"])
}
