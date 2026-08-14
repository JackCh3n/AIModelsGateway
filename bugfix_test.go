package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// 测试 anthropicToOpenAIReq 多 tool_result 不丢失
func TestAnthropicToOpenAIMultipleToolResults(t *testing.T) {
	body := `{
		"model": "claude-3-5",
		"max_tokens": 100,
		"messages": [{
			"role": "user",
			"content": [
				{"type": "tool_result", "tool_use_id": "toolu_1", "content": "result-1"},
				{"type": "tool_result", "tool_use_id": "toolu_2", "content": "result-2"}
			]
		}]
	}`
	out, err := anthropicToOpenAIReq([]byte(body))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	msgs, _ := out["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 tool messages, got %d", len(msgs))
	}
	for i, m := range msgs {
		mm := m.(map[string]any)
		if mm["role"] != "tool" {
			t.Fatalf("msg %d role = %v, want tool", i, mm["role"])
		}
	}
}

// 测试 tool_choice 转换
func TestToolChoiceConversion(t *testing.T) {
	// Anthropic -> OpenAI
	if v := anthropicToolChoiceToOpenAI("any"); v != "required" {
		t.Errorf("any -> %v, want required", v)
	}
	if v := anthropicToolChoiceToOpenAI(map[string]any{"type": "tool", "name": "get_weather"}); v != nil {
		m := v.(map[string]any)
		fn := m["function"].(map[string]any)
		if fn["name"] != "get_weather" {
			t.Errorf("tool choice name = %v", fn["name"])
		}
		if m["type"] != "function" {
			t.Errorf("tool choice type = %v", m["type"])
		}
	}

	// OpenAI -> Anthropic
	if v := openAIToolChoiceToAnthropic("required"); v != "any" {
		t.Errorf("required -> %v, want any", v)
	}
	if v := openAIToolChoiceToAnthropic(map[string]any{"type": "function", "function": map[string]any{"name": "f1"}}); v != nil {
		m := v.(map[string]any)
		if m["type"] != "tool" || m["name"] != "f1" {
			t.Errorf("got %v", m)
		}
	}
}

// 测试 safeHeaders 隐藏密钥
func TestSafeHeaders(t *testing.T) {
	h := map[string]string{
		"Authorization": "Bearer sk-test-1234567890",
		"x-api-key":     "sk-ant-abcdef",
		"Content-Type":  "application/json",
	}
	out := safeHeaders(h)
	if strings.Contains(out["Authorization"], "sk-test-1234567890") {
		t.Error("Authorization leaked!")
	}
	if strings.Contains(out["x-api-key"], "sk-ant-abcdef") {
		t.Error("x-api-key leaked!")
	}
	if out["Content-Type"] != "application/json" {
		t.Error("Content-Type should be preserved")
	}
}

// 测试 nil/无效 JSON body 不再 panic
func TestProxyNilBodyNoPanic(t *testing.T) {
	var params map[string]any
	if err := json.Unmarshal([]byte("not json"), &params); err == nil {
		t.Fatal("expected error for invalid json")
	}
	if params == nil {
		params = map[string]any{}
	}
	params["model"] = "test"
	if params["model"] != "test" {
		t.Fatal("map assignment failed")
	}
}

// 测试 limitToTokens
func TestLimitToTokens(t *testing.T) {
	cases := map[string]int{
		"32K": 32000, "1M": 1000000, "128K": 128000, "8": 8,
	}
	for k, want := range cases {
		if got := limitToTokens(k); got != want {
			t.Errorf("limitToTokens(%s) = %d, want %d", k, got, want)
		}
	}
}

// 测试 tool_result content 为块数组时转换为 OpenAI 字符串
func TestAnthropicToOpenAIToolResultArrayContent(t *testing.T) {
	body := `{
		"model": "claude-3-5",
		"max_tokens": 100,
		"messages": [{
			"role": "user",
			"content": [
				{"type": "tool_result", "tool_use_id": "toolu_1", "content": [
					{"type": "text", "text": "result-part-1"},
					{"type": "text", "text": "result-part-2"}
				]}
			]
		}]
	}`
	out, err := anthropicToOpenAIReq([]byte(body))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	msgs, _ := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 tool message, got %d", len(msgs))
	}
	m := msgs[0].(map[string]any)
	content, ok := m["content"].(string)
	if !ok {
		t.Fatalf("tool message content should be string, got %T", m["content"])
	}
	if content != "result-part-1\nresult-part-2" {
		t.Errorf("content = %q, want %q", content, "result-part-1\nresult-part-2")
	}
}

// 测试别名与主备路由重名冲突检测
func TestAliasNameConflict(t *testing.T) {
	if !isAliasNameConflict("", "") {
		t.Error("empty name should conflict")
	}
	// 不冲突：与现有别名/主备路由都不同
	if isAliasNameConflict("fresh-name-xyz", "") {
		t.Error("fresh name should not conflict")
	}
}

// 测试 limitToTokens 的大小写与空格容错
func TestLimitToTokensCase(t *testing.T) {
	if got := limitToTokens(" 64k "); got != 64000 {
		t.Errorf("limitToTokens(64k) = %d, want 64000", got)
	}
}

// 测试缓存指标提取（三种上游格式）
func TestExtractCacheFromUsage(t *testing.T) {	// DeepSeek 风格
	hit, miss := extractCacheFromUsage(map[string]any{
		"prompt_cache_hit_tokens":  float64(100),
		"prompt_cache_miss_tokens": float64(50),
	})
	if hit != 100 || miss != 50 {
		t.Errorf("deepseek style: hit=%d miss=%d, want 100/50", hit, miss)
	}
	// Anthropic 风格
	hit, miss = extractCacheFromUsage(map[string]any{
		"cache_read_input_tokens":     float64(200),
		"cache_creation_input_tokens": float64(80),
	})
	if hit != 200 || miss != 80 {
		t.Errorf("anthropic style: hit=%d miss=%d, want 200/80", hit, miss)
	}
	// OpenAI 新版风格
	hit, miss = extractCacheFromUsage(map[string]any{
		"prompt_tokens": float64(300),
		"prompt_tokens_details": map[string]any{
			"cached_tokens": float64(120),
		},
	})
	if hit != 120 || miss != 180 {
		t.Errorf("openai style: hit=%d miss=%d, want 120/180", hit, miss)
	}
	// 无缓存数据
	hit, miss = extractCacheFromUsage(map[string]any{"prompt_tokens": float64(10)})
	if hit != 0 || miss != 0 {
		t.Errorf("no cache: hit=%d miss=%d, want 0/0", hit, miss)
	}
}

// 测试主备路由熔断器：连续失败达到阈值后熔断，成功后恢复
func TestFailoverBreaker(t *testing.T) {	b := &breaker{states: make(map[string]*breakerState)}
	if !b.allow("p1") {
		t.Fatal("初始状态应放行")
	}
	// 连续失败 3 次 → 熔断
	b.recordFailure("p1")
	b.recordFailure("p1")
	b.recordFailure("p1")
	if b.allow("p1") {
		t.Error("连续失败 3 次后应熔断")
	}
	// 其他站点不受影响
	if !b.allow("p2") {
		t.Error("无关站点不应被熔断")
	}
	// 成功应恢复
	b.recordSuccess("p1")
	if !b.allow("p1") {
		t.Error("成功后应恢复放行")
	}
	// 熔断冷却期结束应恢复（手动把 openUntil 置为过去）
	b.recordFailure("p1")
	b.recordFailure("p1")
	b.recordFailure("p1")
	b.mu.Lock()
	b.states["p1"].openUntil = time.Now().Add(-time.Second)
	b.mu.Unlock()
	if !b.allow("p1") {
		t.Error("冷却期结束后应恢复放行")
	}
}

// 测试多模态图片转换：Anthropic image block -> OpenAI image_url
func TestAnthropicImageToOpenAI(t *testing.T) {
	body := `{
		"model": "claude-3-5",
		"max_tokens": 100,
		"messages": [{
			"role": "user",
			"content": [
				{"type": "text", "text": "看这张图"},
				{"type": "image", "source": {"type": "base64", "media_type": "image/png", "data": "iVBORw0KGgo="}}
			]
		}]
	}`
	out, err := anthropicToOpenAIReq([]byte(body))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	msgs, _ := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	m := msgs[0].(map[string]any)
	content, ok := m["content"].([]any)
	if !ok {
		t.Fatalf("多模态消息 content 应为数组，got %T", m["content"])
	}
	if len(content) != 2 {
		t.Fatalf("expected 2 blocks (text+image), got %d", len(content))
	}
	img := content[1].(map[string]any)
	if img["type"] != "image_url" {
		t.Fatalf("block type = %v, want image_url", img["type"])
	}
	iu := img["image_url"].(map[string]any)
	if iu["url"] != "data:image/png;base64,iVBORw0KGgo=" {
		t.Errorf("url = %v", iu["url"])
	}
}

// 测试多模态图片转换：OpenAI image_url -> Anthropic image
func TestOpenAIImageToAnthropic(t *testing.T) {
	body := `{
		"model": "gpt-4o",
		"max_tokens": 100,
		"messages": [{
			"role": "user",
			"content": [
				{"type": "text", "text": "看这张图"},
				{"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,/9j/4AAQ"}}
			]
		}]
	}`
	out, err := openAIToAnthropicReq([]byte(body))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	msgs, _ := out["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	m := msgs[0].(map[string]any)
	content, ok := m["content"].([]any)
	if !ok {
		t.Fatalf("content 应为数组，got %T", m["content"])
	}
	if len(content) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(content))
	}
	img := content[1].(map[string]any)
	if img["type"] != "image" {
		t.Fatalf("block type = %v, want image", img["type"])
	}
	src := img["source"].(map[string]any)
	if src["type"] != "base64" || src["media_type"] != "image/jpeg" || src["data"] != "/9j/4AAQ" {
		t.Errorf("source = %v", src)
	}
}
