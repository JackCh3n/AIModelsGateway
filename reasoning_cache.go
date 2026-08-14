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

// lastReasoning 最近缓存的多条 reasoning_content（按时间从新到旧，去重）。
// 兜底场景：部分客户端（如 LiteLLM）在后续请求中会重新生成 tool_call_id，
// 导致按 id 精确匹配失败。此时从最近列表中按顺序分配不同的 reasoning 注入，
// 避免多条不同轮次的消息拿到同一条 reasoning。
var lastReasoning = struct {
	sync.RWMutex
	seq []reasoningSeqItem
}{}

// reasoningRecentMaxSize 最近 reasoning 兜底列表最大保留条数
const reasoningRecentMaxSize = 20

// reasoningSeqItem 最近 reasoning 列表中的一条
type reasoningSeqItem struct {
	content string
	ts      time.Time
}

// cacheReasoningByToolCalls 缓存 reasoning_content，按 tool_call_id 索引
// 同时存储一份按 reasoning_content 自身的索引（用于无 tool_calls 时回退查找）
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

	// 更新最近 reasoning 兜底列表（去重，新的放最前）
	lastReasoning.Lock()
	// 移除同内容的旧条目，避免重复
	filtered := lastReasoning.seq[:0]
	for _, it := range lastReasoning.seq {
		if it.content != reasoningContent {
			filtered = append(filtered, it)
		}
	}
	lastReasoning.seq = filtered
	lastReasoning.seq = append([]reasoningSeqItem{{content: reasoningContent, ts: time.Now()}}, lastReasoning.seq...)
	if len(lastReasoning.seq) > reasoningRecentMaxSize {
		lastReasoning.seq = lastReasoning.seq[:reasoningRecentMaxSize]
	}
	lastReasoning.Unlock()
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

// lookupRecentReasonings 返回最近缓存的 reasoning_content 列表（从新到旧，TTL 内有效）。
// 用于一次性取多条不同的 reasoning，供兜底注入时分配，避免多条消息拿到相同内容。
func lookupRecentReasonings() []string {
	lastReasoning.RLock()
	defer lastReasoning.RUnlock()
	out := make([]string, 0, len(lastReasoning.seq))
	for _, it := range lastReasoning.seq {
		if it.content != "" && time.Since(it.ts) <= reasoningCacheTTL {
			out = append(out, it.content)
		}
	}
	return out
}

// injectReasoningIntoRequestBody 扫描请求 body 中的 messages，
// 对包含 tool_calls 但缺少 reasoning_content 的 assistant 消息，
// 从缓存中查找并注入 reasoning_content。
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

	// 兜底分配用的最近 reasoning 列表与游标。
	// 同一请求内多条消息兜底时，按顺序分配给不同的 reasoning，避免重复。
	recentPool := lookupRecentReasonings()
	recentIdx := 0
	takeRecent := func() string {
		if len(recentPool) == 0 {
			return ""
		}
		// 优先取当前游标，游标越界则重头循环
		if recentIdx >= len(recentPool) {
			recentIdx = 0
		}
		r := recentPool[recentIdx]
		recentIdx++
		return r
	}

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

		// 必须有 tool_calls 才需要注入
		toolCalls, ok := msg["tool_calls"].([]any)
		hasToolCalls := ok && len(toolCalls) > 0
		if !hasToolCalls {
			// 无 tool_calls 的 assistant 消息也可能需要注入 reasoning：
			// DeepSeek 思维模式下，若历史 assistant 消息曾在思维中产出 reasoning，
			// 客户端(如 LiteLLM)在后续轮次仍要求将其传回，否则 400。
			// 此处用最近缓存的 reasoning 兜底注入（仅在消息为 assistant 且无 reasoning 时）。
			if r := takeRecent(); r != "" {
				msg["reasoning_content"] = r
				messages[i] = msg
				injected++
				modified = true
				log.Printf("[reasoning] injected via recent reasoning into plain assistant msg #%d (no tool_calls)", i)
			}
			continue
		}

		// 从 tool_calls 中取 id 查找缓存的 reasoning_content
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

		// 兜底：id 精确匹配失败（客户端重新生成 tool_call_id）时，
		// 从最近列表按顺序取不同的 reasoning 注入，避免 upstream 400 报错
		if reasoning == "" {
			if r := takeRecent(); r != "" {
				reasoning = r
				log.Printf("[reasoning] fallback inject via recent reasoning (tool_calls=%d)", len(toolCalls))
			}
		}

		if reasoning != "" {
			msg["reasoning_content"] = reasoning
			messages[i] = msg
			injected++
			modified = true
			log.Printf("[reasoning] injected reasoning_content for assistant msg #%d (tool_calls=%d)", i, len(toolCalls))
		}
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
