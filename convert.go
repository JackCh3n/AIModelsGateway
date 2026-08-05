package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ============================================================
// 协议转换：OpenAI Chat Completions ↔ Anthropic Messages
// 支持四种场景：
//   1. OpenAI in  -> OpenAI out  (直通)
//   2. OpenAI in  -> Anthropic out (请求: openAIToAnthropicReq, 响应: anthropicToOpenAIResp)
//   3. Anthropic in -> OpenAI out  (请求: anthropicToOpenAIReq, 响应: openAIToAnthropicResp)
//   4. Anthropic in -> Anthropic out (直通)
// ============================================================

// --- Anthropic 请求 -> OpenAI 请求 ---

func anthropicToOpenAIReq(body []byte) (map[string]any, error) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	openAI := map[string]any{
		"model": req["model"],
	}
	if mt, ok := req["max_tokens"].(float64); ok {
		openAI["max_tokens"] = int(mt)
	}
	if s, ok := req["stream"].(bool); ok {
		openAI["stream"] = s
	}
	if t, ok := req["temperature"].(float64); ok && t > 0 {
		openAI["temperature"] = t
	}
	if tp, ok := req["top_p"].(float64); ok && tp > 0 {
		openAI["top_p"] = tp
	}
	if tk, ok := req["top_k"].(float64); ok && tk > 0 {
		openAI["top_k"] = int(tk)
	}
	// stop_sequences -> stop
	if ss, ok := req["stop_sequences"]; ok {
		openAI["stop"] = ss
	}

	// system 字段 -> system message
	msgs := []any{}
	if sys, ok := req["system"]; ok {
		sysText := extractStringContent(sys)
		if sysText != "" {
			msgs = append(msgs, map[string]any{"role": "system", "content": sysText})
		}
	}

	// 转换 messages
	if rawMsgs, ok := req["messages"].([]any); ok {
		for _, m := range rawMsgs {
			msg, ok := m.(map[string]any)
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)
			content := msg["content"]

			switch c := content.(type) {
			case string:
				msgs = append(msgs, map[string]any{"role": role, "content": c})
			case []any:
				textParts := []string{}
				var toolCalls []any
				var toolResults []any

				for _, block := range c {
					b, ok := block.(map[string]any)
					if !ok {
						continue
					}
					switch b["type"] {
					case "text":
						if t, ok := b["text"].(string); ok {
							textParts = append(textParts, t)
						}
					case "tool_use":
						argsStr := "{}"
						if input, ok := b["input"]; ok && input != nil {
							if s, ok := input.(string); ok {
								argsStr = s
							} else if bts, err := json.Marshal(input); err == nil {
								argsStr = string(bts)
							}
						}
						tc := map[string]any{
							"id":   b["id"],
							"type": "function",
							"function": map[string]any{
								"name":      b["name"],
								"arguments": argsStr,
							},
						}
						toolCalls = append(toolCalls, tc)
					case "tool_result":
						tr := map[string]any{
							"role":         "tool",
							"content":      b["content"],
							"tool_call_id": b["tool_use_id"],
						}
						toolResults = append(toolResults, tr)
					}
				}

				if role == "assistant" && len(toolCalls) > 0 {
					m := map[string]any{
						"role":       "assistant",
						"content":    strings.Join(textParts, "\n"),
						"tool_calls": toolCalls,
					}
					msgs = append(msgs, m)
				} else if role == "user" && len(toolResults) > 0 {
					// 多个 tool_result 需拆成多条独立 tool 消息，避免覆盖丢失
					msgs = append(msgs, toolResults...)
				} else {
					msgs = append(msgs, map[string]any{"role": role, "content": strings.Join(textParts, "\n")})
				}
			}
		}
	}
	openAI["messages"] = msgs

	// tools 转换
	if tools, ok := req["tools"].([]any); ok {
		openAI["tools"] = anthropicToolsToOpenAI(tools)
	}
	// tool_choice: Anthropic 格式转换为 OpenAI 格式
	// Anthropic: {"type": "auto"|"any"|"tool", "name": "xxx"} / "auto" / "none" / "any"
	// OpenAI: {"type": "function", "function": {"name": "xxx"}} / "auto" / "none" / "required"
	if tc, ok := req["tool_choice"]; ok {
		openAI["tool_choice"] = anthropicToolChoiceToOpenAI(tc)
	}

	return openAI, nil
}

// --- OpenAI 请求 -> Anthropic 请求 ---

func openAIToAnthropicReq(body []byte) (map[string]any, error) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	anthropic := map[string]any{
		"model": req["model"],
	}
	if mt, ok := req["max_tokens"].(float64); ok {
		anthropic["max_tokens"] = int(mt)
	} else if mt, ok := req["max_completion_tokens"].(float64); ok {
		anthropic["max_tokens"] = int(mt)
	} else {
		anthropic["max_tokens"] = 4096
	}
	if s, ok := req["stream"].(bool); ok {
		anthropic["stream"] = s
	}
	if t, ok := req["temperature"].(float64); ok && t > 0 {
		anthropic["temperature"] = t
	}
	if tp, ok := req["top_p"].(float64); ok && tp > 0 {
		anthropic["top_p"] = tp
	}
	if stop, ok := req["stop"]; ok {
		anthropic["stop_sequences"] = stop
	}

	msgs := []any{}
	var systemText string

	if rawMsgs, ok := req["messages"].([]any); ok {
		for _, m := range rawMsgs {
			msg, ok := m.(map[string]any)
			if !ok {
				continue
			}
			role, _ := msg["role"].(string)

			if role == "system" {
				if s, ok := msg["content"].(string); ok {
					if systemText != "" {
						systemText += "\n"
					}
					systemText += s
				}
				continue
			}

			if role == "tool" {
				// OpenAI tool result -> Anthropic user message with tool_result block
				content := msg["content"]
				toolCallID, _ := msg["tool_call_id"].(string)
				msgs = append(msgs, map[string]any{
					"role": "user",
					"content": []any{map[string]any{
						"type":        "tool_result",
						"tool_use_id": toolCallID,
						"content":     content,
					}},
				})
				continue
			}

			// 普通消息
			content := msg["content"]
			toolCalls, _ := msg["tool_calls"].([]any)

			if len(toolCalls) > 0 && role == "assistant" {
				// assistant with tool_calls -> Anthropic content blocks
				blocks := []any{}
				if s, ok := content.(string); ok && s != "" {
					blocks = append(blocks, map[string]any{"type": "text", "text": s})
				}
				for _, tc := range toolCalls {
					tcMap, ok := tc.(map[string]any)
					if !ok {
						continue
					}
					fn, _ := tcMap["function"].(map[string]any)
					argsStr, _ := fn["arguments"].(string)
					var argsObj any
					if argsStr != "" {
						json.Unmarshal([]byte(argsStr), &argsObj)
					}
					if argsObj == nil {
						argsObj = map[string]any{}
					}
					blocks = append(blocks, map[string]any{
						"type":  "tool_use",
						"id":    tcMap["id"],
						"name":  fn["name"],
						"input": argsObj,
					})
				}
				msgs = append(msgs, map[string]any{"role": role, "content": blocks})
			} else if s, ok := content.(string); ok {
				msgs = append(msgs, map[string]any{"role": role, "content": s})
			} else if arr, ok := content.([]any); ok {
				// 已经是 content blocks 格式
				msgs = append(msgs, map[string]any{"role": role, "content": arr})
			}
		}
	}

	if systemText != "" {
		anthropic["system"] = systemText
	}
	anthropic["messages"] = msgs

	// tools 转换
	if tools, ok := req["tools"].([]any); ok {
		anthropic["tools"] = openAIToolsToAnthropic(tools)
	}
	// tool_choice: OpenAI 格式转换为 Anthropic 格式
	if tc, ok := req["tool_choice"]; ok {
		anthropic["tool_choice"] = openAIToolChoiceToAnthropic(tc)
	}

	return anthropic, nil
}

// --- OpenAI 响应 -> Anthropic 响应 ---

func openAIToAnthropicResp(openAI map[string]any) map[string]any {
	out := map[string]any{
		"id":   "msg_" + fmt.Sprintf("%x", time.Now().UnixMilli()),
		"type": "message",
		"role": "assistant",
	}
	if m, ok := openAI["model"].(string); ok {
		out["model"] = m
	}

	choices, _ := openAI["choices"].([]any)
	if len(choices) == 0 {
		out["content"] = []any{map[string]any{"type": "text", "text": ""}}
		out["stop_reason"] = "end_turn"
		out["usage"] = map[string]any{"input_tokens": 0, "output_tokens": 0}
		return out
	}

	choice, _ := choices[0].(map[string]any)
	msg, _ := choice["message"].(map[string]any)
	if msg == nil {
		msg, _ = choice["delta"].(map[string]any)
	}

	text := ""
	if msg != nil {
		if c, ok := msg["content"].(string); ok {
			text = c
		}
	}

	contentBlocks := []any{map[string]any{"type": "text", "text": text}}

	// tool_calls -> tool_use blocks
	if msg != nil {
		if tc, ok := msg["tool_calls"].([]any); ok && len(tc) > 0 {
			contentBlocks = []any{}
			if text != "" {
				contentBlocks = append(contentBlocks, map[string]any{"type": "text", "text": text})
			}
			for _, tcItem := range tc {
				tcMap, ok := tcItem.(map[string]any)
				if !ok {
					continue
				}
				fn, _ := tcMap["function"].(map[string]any)
				input := fn["arguments"]
				if argsStr, ok := input.(string); ok {
					var argsObj any
					if json.Unmarshal([]byte(argsStr), &argsObj) == nil {
						input = argsObj
					}
				}
				contentBlocks = append(contentBlocks, map[string]any{
					"type":  "tool_use",
					"id":    tcMap["id"],
					"name":  fn["name"],
					"input": input,
				})
			}
		}
	}

	out["content"] = contentBlocks

	switch getNested(choice, "finish_reason") {
	case "stop":
		out["stop_reason"] = "end_turn"
	case "length":
		out["stop_reason"] = "max_tokens"
	case "tool_calls":
		out["stop_reason"] = "tool_use"
	default:
		out["stop_reason"] = "end_turn"
	}

	usage := map[string]any{}
	if u, ok := openAI["usage"].(map[string]any); ok {
		if pt, ok := u["prompt_tokens"].(float64); ok {
			usage["input_tokens"] = int(pt)
		}
		if ct, ok := u["completion_tokens"].(float64); ok {
			usage["output_tokens"] = int(ct)
		}
	}
	out["usage"] = usage

	return out
}

// --- Anthropic 响应 -> OpenAI 响应 ---

func anthropicToOpenAIResp(anthropic map[string]any) map[string]any {
	out := map[string]any{
		"id":      anthropic["id"],
		"object":  "chat.completion",
		"created": time.Now().Unix(),
	}
	if m, ok := anthropic["model"].(string); ok {
		out["model"] = m
	}

	content, _ := anthropic["content"].([]any)
	text := ""
	var toolCalls []any

	for i, block := range content {
		b, ok := block.(map[string]any)
		if !ok {
			continue
		}
		switch b["type"] {
		case "text":
			if t, ok := b["text"].(string); ok {
				text += t
			}
		case "tool_use":
			argsStr := "{}"
			if input, ok := b["input"]; ok && input != nil {
				if bts, err := json.Marshal(input); err == nil {
					argsStr = string(bts)
				}
			}
			tc := map[string]any{
				"id":   b["id"],
				"type": "function",
				"function": map[string]any{
					"name":      b["name"],
					"arguments": argsStr,
				},
			}
			toolCalls = append(toolCalls, tc)
			_ = i
		}
	}

	message := map[string]any{
		"role":    "assistant",
		"content": text,
	}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
	}

	finishReason := "stop"
	switch anthropic["stop_reason"] {
	case "end_turn":
		finishReason = "stop"
	case "max_tokens":
		finishReason = "length"
	case "tool_use":
		finishReason = "tool_calls"
	}

	out["choices"] = []any{map[string]any{
		"index":         0,
		"message":       message,
		"finish_reason": finishReason,
	}}

	usage := map[string]any{}
	if u, ok := anthropic["usage"].(map[string]any); ok {
		if it, ok := u["input_tokens"].(float64); ok {
			usage["prompt_tokens"] = int(it)
		}
		if ot, ok := u["output_tokens"].(float64); ok {
			usage["completion_tokens"] = int(ot)
		}
		usage["total_tokens"] = 0
		if pt, ok := usage["prompt_tokens"].(int); ok {
			if ct, ok := usage["completion_tokens"].(int); ok {
				usage["total_tokens"] = pt + ct
			}
		}
	}
	out["usage"] = usage

	return out
}

// --- 工具函数 ---

func extractStringContent(raw any) string {
	switch v := raw.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case json.RawMessage:
		var s string
		if json.Unmarshal(v, &s) == nil {
			return s
		}
		var blocks []map[string]any
		if json.Unmarshal(v, &blocks) == nil {
			parts := []string{}
			for _, b := range blocks {
				if b["type"] == "text" {
					if t, ok := b["text"].(string); ok {
						parts = append(parts, t)
					}
				}
			}
			return strings.Join(parts, "\n")
		}
	case []any:
		parts := []string{}
		for _, b := range v {
			if bm, ok := b.(map[string]any); ok {
				if bm["type"] == "text" {
					if t, ok := bm["text"].(string); ok {
						parts = append(parts, t)
					}
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func anthropicToolsToOpenAI(tools []any) []any {
	out := make([]any, 0, len(tools))
	for _, t := range tools {
		tMap, ok := t.(map[string]any)
		if !ok {
			continue
		}
		if tMap["type"] == "function" {
			out = append(out, t)
			continue
		}
		oai := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tMap["name"],
				"description": tMap["description"],
				"parameters":  tMap["input_schema"],
			},
		}
		out = append(out, oai)
	}
	return out
}

func openAIToolsToAnthropic(tools []any) []any {
	out := make([]any, 0, len(tools))
	for _, t := range tools {
		tMap, ok := t.(map[string]any)
		if !ok {
			continue
		}
		// 已经是 Anthropic 格式
		if _, ok := tMap["input_schema"]; ok {
			out = append(out, t)
			continue
		}
		fn, _ := tMap["function"].(map[string]any)
		if fn == nil {
			continue
		}
		ant := map[string]any{
			"name":         fn["name"],
			"description":  fn["description"],
			"input_schema": fn["parameters"],
		}
		out = append(out, ant)
	}
	return out
}

// anthropicToolChoiceToOpenAI 将 Anthropic tool_choice 转换为 OpenAI 格式
func anthropicToolChoiceToOpenAI(tc any) any {
	switch v := tc.(type) {
	case string:
		// "auto" / "none" / "any"
		if v == "any" {
			return "required"
		}
		return v
	case map[string]any:
		if t, _ := v["type"].(string); t == "tool" {
			name, _ := v["name"].(string)
			return map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": name,
				},
			}
		}
		// {"type": "auto"|"any"}
		if t, _ := v["type"].(string); t == "any" {
			return "required"
		}
		return "auto"
	default:
		return tc
	}
}

// openAIToolChoiceToAnthropic 将 OpenAI tool_choice 转换为 Anthropic 格式
func openAIToolChoiceToAnthropic(tc any) any {
	switch v := tc.(type) {
	case string:
		// "auto" / "none" / "required"
		if v == "required" {
			return "any"
		}
		return v
	case map[string]any:
		if t, _ := v["type"].(string); t == "function" {
			if fn, ok := v["function"].(map[string]any); ok {
				name, _ := fn["name"].(string)
				return map[string]any{
					"type": "tool",
					"name": name,
				}
			}
		}
		return "auto"
	default:
		return tc
	}
}

func getNested(obj map[string]any, keys ...any) any {
	current := any(obj)
	for _, key := range keys {
		switch k := key.(type) {
		case string:
			if m, ok := current.(map[string]any); ok {
				current = m[k]
			} else {
				return nil
			}
		case int:
			if arr, ok := current.([]any); ok && k < len(arr) {
				current = arr[k]
			} else {
				return nil
			}
		default:
			return nil
		}
	}
	return current
}

// extractUsage 从响应中提取 token 用量
func extractUsage(resp map[string]any, format string) (input, output int) {
	if format == "anthropic" {
		if u, ok := resp["usage"].(map[string]any); ok {
			if it, ok := u["input_tokens"].(float64); ok {
				input = int(it)
			}
			if ot, ok := u["output_tokens"].(float64); ok {
				output = int(ot)
			}
		}
	} else {
		if u, ok := resp["usage"].(map[string]any); ok {
			if pt, ok := u["prompt_tokens"].(float64); ok {
				input = int(pt)
			}
			if ct, ok := u["completion_tokens"].(float64); ok {
				output = int(ct)
			}
		}
	}
	return
}
