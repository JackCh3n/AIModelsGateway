package main

import (
	"encoding/json"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

// ============================================================
// reasoning_content 缓存与注入
//
// 背景：DeepSeek V4 系列模型默认开启思维模式，响应中包含 reasoning_content。
// 当对话包含工具调用（tool_calls）时，后续请求必须将对应的 reasoning_content
// 完整传回 API，否则返回 400 错误：
//   "The `reasoning_content` in the thinking mode must be passed back to the API."
//
// 许多客户端（Cherry Studio / NextChat 等）在多轮对话中会丢失这个字段，
// 网关层通过按 tool_call_id 缓存 reasoning_content，并在后续请求中自动注入，
// 解决客户端不兼容的问题。
// ============================================================

const (
	reasoningCacheTTL       = 15 * time.Minute // 单条缓存有效期
	reasoningCacheMaxSize   = 10000            // 最大缓存条数
	reasoningCacheCleanSize = 2000             // 清理时保留的条数
)

// reasoningEntry 单条缓存
type reasoningEntry struct {
	content   string
	toolIDs   []string
	createdAt time.Time
}

// reasoningCache 全局 reasoning_content 缓存
var reasoningCache = struct {
	sync.RWMutex
	entries map[string]*reasoningEntry // key: tool_call_id
}{
	entries: make(map[string]*reasoningEntry),
}

// cacheReasoningByToolCalls 缓存 reasoning_content，按 tool_call_id 索引。
// 只按 tool_call_id 精确索引，不做「最近列表」兜底：
// 兜底注入会把 A 会话的思维链塞进 B 会话的消息，造成内容串台/答非所问，
// 宁可让上游返回 400 也不注入错误内容。
func cacheReasoningByToolCalls(toolCallIDs []string, reasoningContent string) {
	if reasoningContent == "" || len(toolCallIDs) == 0 {
		return
	}

	entry := &reasoningEntry{
		content:   reasoningContent,
		toolIDs:   toolCallIDs,
		createdAt: time.Now(),
	}

	reasoningCache.Lock()
	defer reasoningCache.Unlock()

	// 容量控制
	if len(reasoningCache.entries) >= reasoningCacheMaxSize {
		cleanReasoningCacheLocked()
	}

	for _, id := range toolCallIDs {
		if id != "" {
			reasoningCache.entries[id] = entry
		}
	}
}

// cleanReasoningCacheLocked 清理过期或超量的缓存（调用前需持锁）
func cleanReasoningCacheLocked() {
	now := time.Now()
	for id, e := range reasoningCache.entries {
		if now.Sub(e.createdAt) > reasoningCacheTTL {
			delete(reasoningCache.entries, id)
		}
	}

	// 如果仍然超量，按创建时间清理最旧的
	if len(reasoningCache.entries) >= reasoningCacheMaxSize {
		type kv struct {
			id string
			ts time.Time
		}
		all := make([]kv, 0, len(reasoningCache.entries))
		for id, e := range reasoningCache.entries {
			all = append(all, kv{id, e.createdAt})
		}
		sort.Slice(all, func(i, j int) bool { return all[i].ts.Before(all[j].ts) })
		// 保留最新的 reasoningCacheCleanSize 条
		remove := len(all) - reasoningCacheCleanSize
		for i := 0; i < remove && i < len(all); i++ {
			delete(reasoningCache.entries, all[i].id)
		}
	}
}

// lookupReasoningByToolCallID 按 tool_call_id 查找缓存的 reasoning_content
func lookupReasoningByToolCallID(toolCallID string) string {
	if toolCallID == "" {
		return ""
	}
	reasoningCache.RLock()
	defer reasoningCache.RUnlock()
	if e, ok := reasoningCache.entries[toolCallID]; ok {
		if time.Since(e.createdAt) <= reasoningCacheTTL {
			return e.content
		}
	}
	return ""
}

// injectReasoningIntoRequestBody 扫描请求 body 中的 messages，
// 对「带 tool_calls 且缺少 reasoning_content」的 assistant 消息，
// 按 tool_call_id 精确匹配缓存并注入 reasoning_content。
//
// 安全策略：
//   - 只对带 tool_calls 的 assistant 消息注入（DeepSeek 思维模式约束场景）
//   - 只按 tool_call_id 精确匹配；匹配不到不注入（宁可上游 400，
//     也不把其他会话/轮次的思维链注入进来导致内容串台）
//   - 无 tool_calls 的 assistant 消息一律不注入
//
// 返回可能被修改的 body 和注入的条数。
func injectReasoningIntoRequestBody(body []byte) ([]byte, int) {
	var params map[string]any
	if err := json.Unmarshal(body, &params); err != nil {
		return body, 0
	}

	messages, ok := params["messages"].([]any)
	if !ok || len(messages) == 0 {
		return body, 0
	}

	injected := 0
	modified := false

	for i, m := range messages {
		msg, ok := m.(map[string]any)
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		if role != "assistant" {
			continue
		}

		// 已有 reasoning_content（非空）则跳过
		if rc, exists := msg["reasoning_content"]; exists {
			if rcStr, ok := rc.(string); ok && rcStr != "" {
				continue
			}
		}

		// 只处理带 tool_calls 的 assistant 消息；无 tool_calls 的不注入
		toolCalls, ok := msg["tool_calls"].([]any)
		if !ok || len(toolCalls) == 0 {
			continue
		}

		// 从 tool_calls 中取 id 精确查找缓存的 reasoning_content
		var reasoning string
		for _, tc := range toolCalls {
			tcMap, ok := tc.(map[string]any)
			if !ok {
				continue
			}
			id, _ := tcMap["id"].(string)
			if id == "" {
				continue
			}
			if r := lookupReasoningByToolCallID(id); r != "" {
				reasoning = r
				break
			}
		}

		// 精确匹配失败：不注入（避免跨会话污染）
		if reasoning == "" {
			continue
		}

		msg["reasoning_content"] = reasoning
		messages[i] = msg
		injected++
		modified = true
		log.Printf("[reasoning] injected reasoning_content for assistant msg #%d (tool_calls=%d)", i, len(toolCalls))
	}

	if !modified {
		return body, 0
	}

	params["messages"] = messages
	newBody, err := json.Marshal(params)
	if err != nil {
		return body, injected
	}
	return newBody, injected
}

// extractReasoningFromOpenAIResponse 从 OpenAI 格式响应中提取 reasoning_content 和 tool_call_ids
// 适用于非流式响应
func extractReasoningFromOpenAIResponse(raw map[string]any) (string, []string) {
	choices, ok := raw["choices"].([]any)
	if !ok || len(choices) == 0 {
		return "", nil
	}
	choice, ok := choices[0].(map[string]any)
	if !ok {
		return "", nil
	}
	msg, ok := choice["message"].(map[string]any)
	if !ok {
		return "", nil
	}

	reasoning, _ := msg["reasoning_content"].(string)
	if reasoning == "" {
		return "", nil
	}

	var toolIDs []string
	if toolCalls, ok := msg["tool_calls"].([]any); ok {
		for _, tc := range toolCalls {
			tcMap, ok := tc.(map[string]any)
			if !ok {
				continue
			}
			id, _ := tcMap["id"].(string)
			if id != "" {
				toolIDs = append(toolIDs, id)
			}
		}
	}

	return reasoning, toolIDs
}

// extractReasoningAndToolIDsFromOpenAIChunk 从 OpenAI 流式 chunk 中提取
// reasoning_content（增量累积）和 tool_call ids（首次出现时收集）。
// reasoningBuf 和 toolCallIDs 由调用者维护，函数会追加写入。
func extractReasoningAndToolIDsFromOpenAIChunk(obj map[string]any, reasoningBuf *strings.Builder, toolCallIDs *[]string) {
	choices, ok := obj["choices"].([]any)
	if !ok || len(choices) == 0 {
		return
	}
	choice, ok := choices[0].(map[string]any)
	if !ok {
		return
	}

	// 流式 delta
	if delta, ok := choice["delta"].(map[string]any); ok {
		if rc, ok := delta["reasoning_content"].(string); ok && rc != "" {
			reasoningBuf.WriteString(rc)
		}
		if tcs, ok := delta["tool_calls"].([]any); ok {
			for _, tc := range tcs {
				tcMap, ok := tc.(map[string]any)
				if !ok {
					continue
				}
				if id, ok := tcMap["id"].(string); ok && id != "" {
					*toolCallIDs = append(*toolCallIDs, id)
				}
			}
		}
	}

	// 非流式 message（部分上游会在流式响应中返回完整 message）
	if msg, ok := choice["message"].(map[string]any); ok {
		if reasoningBuf.Len() == 0 {
			if rc, ok := msg["reasoning_content"].(string); ok && rc != "" {
				reasoningBuf.WriteString(rc)
			}
		}
		if tcs, ok := msg["tool_calls"].([]any); ok {
			for _, tc := range tcs {
				tcMap, ok := tc.(map[string]any)
				if !ok {
					continue
				}
				if id, ok := tcMap["id"].(string); ok && id != "" {
					*toolCallIDs = append(*toolCallIDs, id)
				}
			}
		}
	}
}
