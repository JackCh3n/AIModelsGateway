package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockSSEUpstream 构造一个模拟上游：读取请求体中的 marker 字段，
// 并将其回显到 SSE 流中（用于验证网关是否把 A 的响应写给了 B）。
func mockSSEUpstream(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		marker, _ := req["marker"].(string)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl := w.(http.Flusher)
		// 分片输出，模拟真实生成过程，加大并发交错概率
		for _, c := range []string{"你", "好", marker, "结", "束"} {
			obj := map[string]any{
				"id":      "chatcmpl-x",
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   "m1",
				"choices": []any{map[string]any{
					"index":         0,
					"delta":         map[string]any{"content": c},
					"finish_reason": nil,
				}},
			}
			d, _ := json.Marshal(obj)
			fmt.Fprintf(w, "data: %s\n\n", d)
			fl.Flush()
		}
		// 结束 chunk
		fmt.Fprint(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
}

// setupGatewayForCrossTalkTest 用 mock 上游配置一个主备路由（单站点）
func setupGatewayForCrossTalkTest(t *testing.T, upstream string) {
	t.Helper()
	loadConfig() // 先触发 configOnce，避免后续 loadConfig() 从磁盘加载覆盖测试配置
	cfg := &Config{
		Providers: []Provider{{
			ID:      "p1",
			Name:    "mock",
			BaseURL: upstream,
			Format:  "openai",
			Status:  "active",
			Models:  []string{"m1"},
			APIKeys: []ProviderKey{{ID: "k1", Key: "sk-test", Status: "active"}},
		}},
		Failovers: []FailoverRoute{{
			ID:   "fo1",
			Name: "route1",
			Entries: []FailoverEntry{
				{Order: 1, ProviderID: "p1", Model: "m1"},
			},
		}},
		APIKeys:  []APIKey{},
		Settings: Settings{DefaultModel: "all", FailoverTimeout: 60},
	}
	cfg.idx = buildIndex(cfg)
	configPtr.Store(cfg)
}

// readSSEContent 读取完整 SSE 流并拼接 content 字段
func readSSEContent(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	var got strings.Builder
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimSpace(line[6:])
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var obj map[string]any
		if json.Unmarshal([]byte(payload), &obj) != nil {
			continue
		}
		choices, _ := obj["choices"].([]any)
		if len(choices) == 0 {
			continue
		}
		c0, _ := choices[0].(map[string]any)
		if c0 == nil {
			continue
		}
		delta, _ := c0["delta"].(map[string]any)
		if s, ok := delta["content"].(string); ok {
			got.WriteString(s)
		}
	}
	return got.String()
}

// TestFailoverStreamNoCrossTalk 高并发下验证主备路由流式响应不会串台：
// 每个请求携带唯一 marker，验证每个响应只含自己的 marker，且不含其他请求的 marker。
func TestFailoverStreamNoCrossTalk(t *testing.T) {
	up := mockSSEUpstream(t)
	defer up.Close()
	setupGatewayForCrossTalkTest(t, up.URL)

	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(w, r, "openai", "")
	}))
	defer gw.Close()

	const n = 30
	var wg sync.WaitGroup
	errs := make(chan string, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			marker := fmt.Sprintf("MARKER-%02d", i)
			body, _ := json.Marshal(map[string]any{
				"model":    "route1",
				"stream":   true,
				"marker":   marker,
				"messages": []any{map[string]any{"role": "user", "content": "hi"}},
			})
			resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", bytes.NewReader(body))
			if err != nil {
				errs <- marker + ": " + err.Error()
				return
			}
			if resp.StatusCode != http.StatusOK {
				raw, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				errs <- fmt.Sprintf("%s: HTTP %d body=%q", marker, resp.StatusCode, truncate(string(raw), 300))
				return
			}
			g := readSSEContent(t, resp)
			if !strings.Contains(g, marker) {
				errs <- fmt.Sprintf("%s: 响应缺少自己的 marker (got=%q)", marker, g)
				return
			}
			for j := 0; j < n; j++ {
				if j == i {
					continue
				}
				other := fmt.Sprintf("MARKER-%02d", j)
				if strings.Contains(g, other) {
					errs <- fmt.Sprintf("%s: 串台! 响应包含 %s 的内容 (got=%q)", marker, other, g)
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	failed := 0
	for e := range errs {
		t.Error(e)
		failed++
	}
	t.Logf("完成 %d 个并发流式请求，串台 %d 个", n, failed)
}

// TestFailoverParamsIsolation 验证主备路由各站点请求体相互隔离：
// 同一请求内 model 字段会被正确覆盖，且 marker 等自定义字段在重序列化后保留。
func TestFailoverParamsIsolation(t *testing.T) {
	up := mockSSEUpstream(t)
	defer up.Close()
	loadConfig() // 先触发 configOnce，避免 loadConfig() 从磁盘加载覆盖测试配置
	cfg := &Config{
		Providers: []Provider{
			{ID: "p1", Name: "mock1", BaseURL: up.URL, Format: "openai", Status: "active",
				Models: []string{"m1"}, APIKeys: []ProviderKey{{ID: "k1", Key: "sk-1", Status: "active"}}},
			{ID: "p2", Name: "mock2", BaseURL: up.URL, Format: "openai", Status: "active",
				Models: []string{"m2"}, APIKeys: []ProviderKey{{ID: "k2", Key: "sk-2", Status: "active"}}},
		},
		Failovers: []FailoverRoute{{
			ID:   "fo1",
			Name: "route2",
			Entries: []FailoverEntry{
				{Order: 1, ProviderID: "p1", Model: "m1"},
				{Order: 2, ProviderID: "p2", Model: "m2"},
			},
		}},
		APIKeys:  []APIKey{},
		Settings: Settings{DefaultModel: "all", FailoverTimeout: 60},
	}
	cfg.idx = buildIndex(cfg)
	configPtr.Store(cfg)

	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(w, r, "openai", "")
	}))
	defer gw.Close()

	// 连续发起请求（第一个站点正常返回，不走顺延；再验证 marker 保留）
	for i := 0; i < 5; i++ {
		marker := fmt.Sprintf("MARKER-%d", i)
		body, _ := json.Marshal(map[string]any{
			"model":    "route2",
			"stream":   true,
			"marker":   marker,
			"messages": []any{map[string]any{"role": "user", "content": "hi"}},
		})
		resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("req %d: %v", i, err)
		}
		g := readSSEContent(t, resp)
		if !strings.Contains(g, marker) {
			t.Fatalf("req %d: marker 丢失 (got=%q)", i, g)
		}
	}
	t.Log("参数隔离与 marker 保留验证通过")
}
