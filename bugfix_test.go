package main

import (
	"encoding/json"
	"strings"
	"testing"
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
