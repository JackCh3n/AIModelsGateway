package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// newTransport 创建 Transport，参数化连接池大小以支持普通/狂暴两种模式
func newTransport(maxIdle, maxIdlePerHost int) *http.Transport {
	return &http.Transport{
		MaxIdleConns:          maxIdle,
		MaxIdleConnsPerHost:   maxIdlePerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
}

// 普通模式连接池参数
const (
	normalMaxIdle        = 500
	normalMaxIdlePerHost = 100
	rageMaxIdle          = 10000 // 狂暴模式：全局最大空闲连接
	rageMaxIdlePerHost   = 5000  // 狂暴模式：单 host 最大空闲连接（支持万级并发到同一上游）
)

var httpClient = &http.Client{
	Timeout:   5 * time.Minute,
	Transport: newTransport(normalMaxIdle, normalMaxIdlePerHost),
}

// applyRageModePool 狂暴模式：加大 HTTP 连接池，支持更多并发
func applyRageModePool() {
	httpClient.Transport = newTransport(rageMaxIdle, rageMaxIdlePerHost)
	// 清空代理 client 缓存，让它们用新参数重建
	proxyClients.Lock()
	proxyClients.m = make(map[string]*http.Client)
	proxyClients.Unlock()
	log.Printf("[rage] HTTP 连接池已加大: MaxIdle=%d PerHost=%d", rageMaxIdle, rageMaxIdlePerHost)
}

// restoreNormalPool 恢复普通模式连接池
func restoreNormalPool() {
	httpClient.Transport = newTransport(normalMaxIdle, normalMaxIdlePerHost)
	proxyClients.Lock()
	proxyClients.m = make(map[string]*http.Client)
	proxyClients.Unlock()
	log.Printf("[rage] HTTP 连接池已恢复普通模式: MaxIdle=%d PerHost=%d", normalMaxIdle, normalMaxIdlePerHost)
}

// 代理 HTTP Client 缓存：按 (enabled,type,addr) 复用 Transport，避免每次请求新建连接
var proxyClients = struct {
	sync.Mutex
	m map[string]*http.Client
}{m: make(map[string]*http.Client)}

// getHTTPClient 返回配置了代理的 HTTP client
func getHTTPClient(p *Provider) *http.Client {
	if !p.ProxyEnabled || p.ProxyAddr == "" {
		return httpClient
	}
	cacheKey := p.ProxyType + "|" + p.ProxyAddr
	proxyClients.Lock()
	defer proxyClients.Unlock()
	if c, ok := proxyClients.m[cacheKey]; ok {
		return c
	}
	transport := newTransport(normalMaxIdle, normalMaxIdlePerHost)
	switch p.ProxyType {
	case "http", "https":
		proxyURL, err := url.Parse(p.ProxyType + "://" + p.ProxyAddr)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	case "socks5":
		dialer, err := proxy.SOCKS5("tcp", p.ProxyAddr, nil, proxy.Direct)
		if err == nil {
			transport.Dial = dialer.Dial
		}
	}
	c := &http.Client{
		Timeout:   5 * time.Minute,
		Transport: transport,
	}
	proxyClients.m[cacheKey] = c
	return c
}

// proxyRequest 处理所有代理请求
// clientFormat: "openai" 或 "anthropic" (客户端发来的格式)
// providerOverride: 指定 provider（来自 URL 路径），为空则用活跃站点
func proxyRequest(w http.ResponseWriter, r *http.Request, clientFormat string, providerOverride string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, clientFormat, http.StatusBadRequest, "read body: "+err.Error())
		return
	}

	// 解析请求判断是否流式
	var params map[string]any
	if err := json.Unmarshal(body, &params); err != nil {
		writeError(w, clientFormat, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if params == nil {
		params = map[string]any{}
	}
	isStream, _ := params["stream"].(bool)
	model, _ := params["model"].(string)

	// 模型别名路由：如果模型名匹配别名，自动路由到指定站点+模型
	if model != "" {
		if alias := getAliasByModel(model); alias != nil {
			log.Printf("[alias] %s -> provider=%s model=%s", alias.Name, alias.ProviderID, alias.Model)
			providerOverride = alias.ProviderID
			model = alias.Model
			params["model"] = model
			body, _ = json.Marshal(params)
		}
	}

	// 主备路由：模型名命中主备路由时，按优先级依次尝试站点（3次失败顺延下一个）
	if model != "" && providerOverride == "" {
		if fo := getFailoverByName(model); fo != nil {
			log.Printf("[failover] 命中主备路由 %s (entries=%d)", fo.Name, len(fo.Entries))
			proxyFailoverRequest(w, r, clientFormat, fo, body, params, isStream)
			return
		}
	}

	var provider *Provider
	if providerOverride != "" {
		provider = getProvider(providerOverride)
		if provider == nil {
			writeError(w, clientFormat, http.StatusNotFound, "指定的中转站不存在: "+providerOverride)
			return
		}
		if provider.Status != "active" {
			writeError(w, clientFormat, http.StatusServiceUnavailable, "该中转站已禁用: "+provider.Name)
			return
		}
	} else {
		provider = getActiveProvider()
	}
	if provider == nil {
		writeError(w, clientFormat, http.StatusServiceUnavailable, "没有可用的中转站，请在管理后台添加")
		return
	}

	// 转发到单个站点（普通模式：默认活跃站点或指定站点）
	forwardToProvider(w, r, clientFormat, provider, model, body, params, isStream)
}

// forwardToProvider 转发请求到单个站点（含格式转换、重试、响应处理）。
// 返回 handled=false 表示该站点调用失败且错误未写入响应（可顺延到备用站点）；
// handled=true 表示响应已写入（成功或已返回错误给客户端）。
func forwardToProvider(w http.ResponseWriter, r *http.Request, clientFormat string, provider *Provider, model string, body []byte, params map[string]any, isStream bool) (handled bool) {
	// 模型为空或 "all" 时，使用站点的默认模型
	if model == "" || model == "all" {
		if provider.DefaultModel != "" {
			model = provider.DefaultModel
			params["model"] = model
			body, _ = json.Marshal(params)
		}
	}

	// 校验该模型是否在该站点已启用（仅当站点有模型列表时校验）
	if len(provider.Models) > 0 {
		inList := false
		for _, m := range provider.Models {
			if m == model {
				inList = true
				break
			}
		}
		if inList && !isModelEnabled(provider, model) {
			writeError(w, clientFormat, http.StatusForbidden, "该模型已被禁用: "+model+"（请在管理后台启用）")
			return true
		}
	}

	// 应用模型上下文配置：如果后台配置了输出限制，覆盖请求中的 max_tokens
	if mc := getModelConfig(provider, model); mc != nil && mc.OutputLimit != "" {
		maxTokens := limitToTokens(mc.OutputLimit)
		if maxTokens > 0 {
			params["max_tokens"] = maxTokens
			// OpenAI 也可能用 max_completion_tokens
			params["max_completion_tokens"] = maxTokens
			body, _ = json.Marshal(params)
			log.Printf("  [model-config] %s: outputLimit=%s -> max_tokens=%d", model, mc.OutputLimit, maxTokens)
		}
	}

	log.Printf("[%s] provider=%s format=%s model=%s stream=%v",
		clientFormat, provider.Name, provider.Format, model, isStream)

	// 根据客户端格式和站点格式决定转换策略
	needConvert := clientFormat != provider.Format

	var upstreamBody []byte
	if needConvert {
		if clientFormat == "anthropic" && provider.Format == "openai" {
			// Anthropic 请求 -> OpenAI 请求
			converted, err := anthropicToOpenAIReq(body)
			if err != nil {
				writeError(w, clientFormat, http.StatusBadRequest, "convert request: "+err.Error())
				return true
			}
			upstreamBody, _ = json.Marshal(converted)
		} else if clientFormat == "openai" && provider.Format == "anthropic" {
			// OpenAI 请求 -> Anthropic 请求
			converted, err := openAIToAnthropicReq(body)
			if err != nil {
				writeError(w, clientFormat, http.StatusBadRequest, "convert request: "+err.Error())
				return true
			}
			upstreamBody, _ = json.Marshal(converted)
		}
	} else {
		upstreamBody = body
	}

	// 注入缺失的 reasoning_content（DeepSeek 思维模式兼容）
	// 仅对 OpenAI 格式上游生效，因为 reasoning_content 是 OpenAI 兼容协议字段
	if provider.Format == "openai" {
		if newBody, n := injectReasoningIntoRequestBody(upstreamBody); n > 0 {
			upstreamBody = newBody
			log.Printf("[reasoning] injected %d reasoning_content(s) into upstream request", n)
		}
	}

	// 构建上游请求
	upstreamURL := provider.BaseURL
	if provider.Format == "openai" {
		upstreamURL = strings.TrimSuffix(upstreamURL, "/") + "/chat/completions"
	} else {
		upstreamURL = strings.TrimSuffix(upstreamURL, "/") + "/messages"
	}

	// 上游请求重试：上游繁忙（503 SERVICE_BUSY）或限流（429）时自动重试最多 3 次
	// 重试间隔逐次递增（500ms / 1s / 1.5s），3 次仍失败才将错误返回客户端
	const maxUpstreamRetries = 3
	apiKey := pickAPIKey(provider)

	var resp *http.Response
	var lastStatus int
	var lastBody []byte

	for attempt := 0; attempt <= maxUpstreamRetries; attempt++ {
		// 每次重试重新构建请求（body 为 bytes.Reader 需重建）
		req, err := http.NewRequest("POST", upstreamURL, bytes.NewReader(upstreamBody))
		if err != nil {
			writeError(w, clientFormat, http.StatusInternalServerError, "create request: "+err.Error())
			return true
		}
		// 绑定客户端 Context：客户端断开时自动取消上游请求，避免资源泄漏
		req = req.WithContext(r.Context())

		// 设置请求头
		req.Header.Set("Content-Type", "application/json")
		if provider.Format == "anthropic" {
			req.Header.Set("x-api-key", apiKey)
			req.Header.Set("anthropic-version", "2023-06-01")
		} else {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		// 应用自定义请求头（覆盖同名默认头）
		for k, v := range provider.CustomHeaders {
			req.Header.Set(k, v)
		}

		resp, err = getHTTPClient(provider).Do(req)
		if err != nil {
			log.Printf("  upstream error: %v", err)
			writeError(w, clientFormat, http.StatusBadGateway, "upstream request failed: "+err.Error())
			return true
		}

		if resp.StatusCode == http.StatusOK {
			break // 成功，进入后续处理
		}

		// 读响应体，判断是否可重试
		lastBody, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		lastStatus = resp.StatusCode

		// 仅上游繁忙(503)或限流(429)时重试；其余错误直接返回
		if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
			if attempt < maxUpstreamRetries {
				delay := time.Duration(attempt+1) * 500 * time.Millisecond
				log.Printf("  upstream %d (SERVICE_BUSY), retry %d/%d after %v: %s",
					resp.StatusCode, attempt+1, maxUpstreamRetries, delay, truncate(string(lastBody), 200))
				select {
				case <-time.After(delay):
				case <-r.Context().Done():
					return true // 客户端已断开，放弃重试
				}
				continue
			}
		}

		// 不可重试错误，或重试次数耗尽：
		// 返回 handled=false 让主备路由顺延到下一个站点
		log.Printf("  upstream %d: %s", lastStatus, truncate(string(lastBody), 500))
		return false
	}
	defer resp.Body.Close()

	if isStream {
		handleStreamProxy(w, resp, clientFormat, provider.Format, provider, model)
	} else {
		handleNonStreamProxy(w, resp, clientFormat, provider.Format, provider, model)
	}
	return true
}

// proxyFailoverRequest 主备路由转发：按优先级依次尝试 entries 中的站点。
// 站点调用失败（3 次重试后仍失败）时顺延下一个站点，最多 3 个站点；
// 全部失败时返回最后一个站点的错误信息。
func proxyFailoverRequest(w http.ResponseWriter, r *http.Request, clientFormat string, fo *FailoverRoute, body []byte, params map[string]any, isStream bool) {
	if len(fo.Entries) == 0 {
		writeError(w, clientFormat, http.StatusServiceUnavailable, "主备路由无可用站点: "+fo.Name)
		return
	}

	var lastErr string
	for i, entry := range fo.Entries {
		provider := getProvider(entry.ProviderID)
		if provider == nil {
			lastErr = fmt.Sprintf("主备路由 %s 站点 %s 不存在", fo.Name, entry.ProviderID)
			log.Printf("[failover] %s: %s", fo.Name, lastErr)
			continue
		}
		if provider.Status != "active" {
			lastErr = fmt.Sprintf("主备路由 %s 站点 %s 已禁用", fo.Name, provider.Name)
			log.Printf("[failover] %s: %s", fo.Name, lastErr)
			continue
		}

		log.Printf("[failover] %s: 尝试站点 %s (order=%d, model=%s)", fo.Name, provider.Name, entry.Order, entry.Model)
		handled := forwardToProvider(w, r, clientFormat, provider, entry.Model, body, params, isStream)
		if handled {
			return // 该站点成功或错误已写入响应
		}
		lastErr = fmt.Sprintf("主备路由 %s 全部站点失败，最后尝试: %s", fo.Name, provider.Name)
		if i < len(fo.Entries)-1 {
			log.Printf("[failover] %s: 站点 %s 失败，顺延下一个", fo.Name, provider.Name)
		}
	}

	// 全部站点失败，返回最后错误（HTTP 502）
	writeError(w, clientFormat, http.StatusBadGateway, lastErr)
}

func handleNonStreamProxy(w http.ResponseWriter, resp *http.Response, clientFormat, providerFormat string, provider *Provider, model string) {
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		writeError(w, clientFormat, http.StatusInternalServerError, "decode response: "+err.Error())
		return
	}

	// 缓存 reasoning_content（DeepSeek 思维模式兼容）
	// 仅 OpenAI 格式上游响应包含 reasoning_content 字段
	if providerFormat == "openai" {
		if reasoning, toolIDs := extractReasoningFromOpenAIResponse(raw); reasoning != "" && len(toolIDs) > 0 {
			cacheReasoningByToolCalls(toolIDs, reasoning)
			log.Printf("[reasoning] cached from non-stream response (len=%d, tool_calls=%d)", len(reasoning), len(toolIDs))
		}
	}

	// 如果需要格式转换
	var out map[string]any
	if clientFormat != providerFormat {
		if providerFormat == "openai" {
			// OpenAI 响应 -> Anthropic 响应
			out = openAIToAnthropicResp(raw)
		} else {
			// Anthropic 响应 -> OpenAI 响应
			out = anthropicToOpenAIResp(raw)
		}
	} else {
		out = raw
	}

	// 提取 token 用量并记录
	respFormat := providerFormat
	input, output := extractUsage(raw, respFormat)
	if input == 0 && output == 0 {
		input, output = extractUsage(out, clientFormat)
	}
	logEntry := newUsageLog(provider.ID, provider.Name, model, input, output, clientFormat)
	addUsageLog(logEntry)

	writeJSON(w, http.StatusOK, out)
}

func handleStreamProxy(w http.ResponseWriter, resp *http.Response, clientFormat, providerFormat string, provider *Provider, model string) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, clientFormat, http.StatusInternalServerError, "streaming not supported")
		return
	}

	if clientFormat == providerFormat {
		// 直通流式响应，同时收集用量和 reasoning_content
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		reader := bufio.NewReader(resp.Body)
		var totalInput, totalOutput int
		var reasoningBuf strings.Builder
		var toolCallIDs []string
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF && line != "" {
					w.Write([]byte(line + "\n"))
					flusher.Flush()
				}
				break
			}
			w.Write([]byte(line))
			flusher.Flush()
			// 尝试从 data: 行提取用量
			trimmed := strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(trimmed, "data: ") {
				payload := strings.TrimSpace(trimmed[6:])
				if payload != "" && payload != "[DONE]" {
					var obj map[string]any
					if json.Unmarshal([]byte(payload), &obj) == nil {
						if u, ok := obj["usage"].(map[string]any); ok {
							if pt, ok := u["prompt_tokens"].(float64); ok {
								totalInput = int(pt)
							}
							if ct, ok := u["completion_tokens"].(float64); ok {
								totalOutput = int(ct)
							}
						}
						// Anthropic 流式用量
						if obj["type"] == "message_start" {
							if msg, ok := obj["message"].(map[string]any); ok {
								if u, ok := msg["usage"].(map[string]any); ok {
									if it, ok := u["input_tokens"].(float64); ok {
										totalInput = int(it)
									}
								}
							}
						}
						if obj["type"] == "message_delta" {
							if u, ok := obj["usage"].(map[string]any); ok {
								if ot, ok := u["output_tokens"].(float64); ok {
									totalOutput = int(ot)
								}
							}
						}
						// OpenAI 流式：提取 reasoning_content 和 tool_call ids
						if providerFormat == "openai" {
							extractReasoningAndToolIDsFromOpenAIChunk(obj, &reasoningBuf, &toolCallIDs)
						}
					}
				}
			}
		}
		// 缓存 reasoning_content（DeepSeek 思维模式兼容）
		if providerFormat == "openai" && reasoningBuf.Len() > 0 && len(toolCallIDs) > 0 {
			cacheReasoningByToolCalls(toolCallIDs, reasoningBuf.String())
			log.Printf("[reasoning] cached from stream passthrough (len=%d, tool_calls=%d)", reasoningBuf.Len(), len(toolCallIDs))
		}
		// 记录用量
		logEntry := newUsageLog(provider.ID, provider.Name, model, totalInput, totalOutput, clientFormat)
		addUsageLog(logEntry)
		return
	}

	if clientFormat == "anthropic" && providerFormat == "openai" {
		// OpenAI SSE -> Anthropic SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		openAISSEToAnthropicSSE(w, flusher, resp.Body, provider, model)
		return
	}

	if clientFormat == "openai" && providerFormat == "anthropic" {
		// Anthropic SSE -> OpenAI SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		anthropicSSEToOpenAISSE(w, flusher, resp.Body, provider, model)
		return
	}
}

// openAISSEToAnthropicSSE 将 OpenAI SSE 流转换为 Anthropic SSE 流
func openAISSEToAnthropicSSE(w http.ResponseWriter, flusher http.Flusher, body io.Reader, provider *Provider, model string) {
	msgID := "msg_" + fmt.Sprintf("%x", time.Now().UnixMilli())
	stopReason := "end_turn"

	emit := func(event string, data any) {
		d, _ := json.Marshal(data)
		w.Write([]byte(fmt.Sprintf("event: %s\n", event)))
		w.Write([]byte(fmt.Sprintf("data: %s\n\n", string(d))))
		flusher.Flush()
	}

	emit("message_start", map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":          msgID,
			"type":        "message",
			"role":        "assistant",
			"content":     []any{},
			"model":       model,
			"stop_reason": nil,
		},
	})

	textIndex := -1
	hasText := false
	pendingTools := map[int]*streamToolAcc{}
	var totalInput, totalOutput int
	var reasoningBuf strings.Builder
	var toolCallIDs []string

	emitToolBlock := func(acc *streamToolAcc) {
		acc.emitted = true
		var argsObj any
		json.Unmarshal([]byte(acc.args), &argsObj)
		if argsObj == nil {
			argsObj = map[string]any{}
		}
		emit("content_block_start", map[string]any{
			"type":  "content_block_start",
			"index": acc.index,
			"content_block": map[string]any{
				"type":  "tool_use",
				"id":    acc.id,
				"name":  acc.name,
				"input": argsObj,
			},
		})
		emit("content_block_stop", map[string]any{
			"type":  "content_block_stop",
			"index": acc.index,
		})
	}

	reader := bufio.NewReader(body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(line[5:])
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var obj map[string]any
		if err := json.Unmarshal([]byte(payload), &obj); err != nil {
			continue
		}

		// 提取用量
		if u, ok := obj["usage"].(map[string]any); ok {
			if pt, ok := u["prompt_tokens"].(float64); ok {
				totalInput = int(pt)
			}
			if ct, ok := u["completion_tokens"].(float64); ok {
				totalOutput = int(ct)
			}
		}

		choices, _ := obj["choices"].([]any)
		if len(choices) == 0 {
			continue
		}
		choice, _ := choices[0].(map[string]any)
		if choice == nil {
			continue
		}

		delta, ok := choice["delta"].(map[string]any)
		if !ok {
			delta = choice
		}

		// 提取 reasoning_content（DeepSeek 思维模式兼容）
		if rc, ok := delta["reasoning_content"].(string); ok && rc != "" {
			reasoningBuf.WriteString(rc)
		}

		// 文本内容
		if c, ok := delta["content"].(string); ok && c != "" {
			if !hasText {
				hasText = true
				textIndex++
				emit("content_block_start", map[string]any{
					"type":          "content_block_start",
					"index":         textIndex,
					"content_block": map[string]any{"type": "text", "text": ""},
				})
			}
			emit("content_block_delta", map[string]any{
				"type":  "content_block_delta",
				"index": textIndex,
				"delta": map[string]any{"type": "text_delta", "text": c},
			})
		}

		// tool_calls
		if tcRaw, ok := delta["tool_calls"].([]any); ok {
			for _, tc := range tcRaw {
				tcMap, _ := tc.(map[string]any)
				if tcMap == nil {
					continue
				}
				idx := 0
				if i, ok := tcMap["index"].(float64); ok {
					idx = int(i)
				}
				// 使用 textIndex+1+idx 作为 content block index
				blkIdx := idx
				if hasText {
					blkIdx = idx + 1
				}
				acc, exists := pendingTools[blkIdx]
				if !exists {
					acc = &streamToolAcc{index: blkIdx}
					pendingTools[blkIdx] = acc
				}
				if id, ok := tcMap["id"].(string); ok && id != "" {
					acc.id = id
					toolCallIDs = append(toolCallIDs, id)
				}
				if fn, ok := tcMap["function"].(map[string]any); ok {
					if name, ok := fn["name"].(string); ok && name != "" {
						acc.name = name
					}
					if args, ok := fn["arguments"].(string); ok && args != "" {
						acc.args += args
					}
				}
				if acc.id != "" && acc.name != "" && acc.args != "" && !acc.emitted {
					emitToolBlock(acc)
				}
			}
		}

		// finish_reason
		if fr, ok := choice["finish_reason"].(string); ok && fr != "" {
			switch fr {
			case "length":
				stopReason = "max_tokens"
			case "tool_calls":
				stopReason = "tool_use"
			}
		}
	}

	if hasText {
		emit("content_block_stop", map[string]any{
			"type":  "content_block_stop",
			"index": textIndex,
		})
	}

	for _, acc := range pendingTools {
		if !acc.emitted {
			emitToolBlock(acc)
		}
	}

	emit("message_delta", map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   stopReason,
			"stop_sequence": nil,
		},
		"usage": map[string]any{"output_tokens": totalOutput},
	})

	emit("message_stop", map[string]any{"type": "message_stop"})

	// 缓存 reasoning_content（DeepSeek 思维模式兼容）
	if reasoningBuf.Len() > 0 && len(toolCallIDs) > 0 {
		cacheReasoningByToolCalls(toolCallIDs, reasoningBuf.String())
		log.Printf("[reasoning] cached from OpenAI->Anthropic stream (len=%d, tool_calls=%d)", reasoningBuf.Len(), len(toolCallIDs))
	}

	// 记录用量
	logEntry := newUsageLog(provider.ID, provider.Name, model, totalInput, totalOutput, "anthropic")
	addUsageLog(logEntry)
}

// anthropicSSEToOpenAISSE 将 Anthropic SSE 流转换为 OpenAI SSE 流
func anthropicSSEToOpenAISSE(w http.ResponseWriter, flusher http.Flusher, body io.Reader, provider *Provider, model string) {
	chatID := "chatcmpl-" + fmt.Sprintf("%x", time.Now().UnixMilli())
	created := time.Now().Unix()
	var totalInput, totalOutput int
	finishReason := "stop"

	reader := bufio.NewReader(body)
	currentEventType := ""

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\r\n")

		if strings.HasPrefix(line, "event: ") {
			currentEventType = strings.TrimSpace(line[7:])
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimSpace(line[6:])
		if payload == "" {
			continue
		}

		var obj map[string]any
		if err := json.Unmarshal([]byte(payload), &obj); err != nil {
			continue
		}

		switch currentEventType {
		case "message_start":
			if msg, ok := obj["message"].(map[string]any); ok {
				if u, ok := msg["usage"].(map[string]any); ok {
					if it, ok := u["input_tokens"].(float64); ok {
						totalInput = int(it)
					}
				}
			}
			// 发送初始 chunk
			chunk := map[string]any{
				"id":      chatID,
				"object":  "chat.completion.chunk",
				"created": created,
				"model":   model,
				"choices": []any{map[string]any{
					"index":         0,
					"delta":         map[string]any{"role": "assistant"},
					"finish_reason": nil,
				}},
			}
			d, _ := json.Marshal(chunk)
			w.Write([]byte("data: " + string(d) + "\n\n"))
			flusher.Flush()

		case "content_block_delta":
			delta, _ := obj["delta"].(map[string]any)
			if delta == nil {
				continue
			}
			if delta["type"] == "text_delta" {
				if t, ok := delta["text"].(string); ok {
					chunk := map[string]any{
						"id":      chatID,
						"object":  "chat.completion.chunk",
						"created": created,
						"model":   model,
						"choices": []any{map[string]any{
							"index":         0,
							"delta":         map[string]any{"content": t},
							"finish_reason": nil,
						}},
					}
					d, _ := json.Marshal(chunk)
					w.Write([]byte("data: " + string(d) + "\n\n"))
					flusher.Flush()
				}
			}
			if delta["type"] == "input_json_delta" {
				// tool_use 参数增量 - 暂时跳过细粒度处理
				if pj, ok := delta["partial_json"].(string); ok {
					chunk := map[string]any{
						"id":      chatID,
						"object":  "chat.completion.chunk",
						"created": created,
						"model":   model,
						"choices": []any{map[string]any{
							"index": 0,
							"delta": map[string]any{
								"tool_calls": []any{map[string]any{
									"index":    0,
									"function": map[string]any{"arguments": pj},
								}},
							},
							"finish_reason": nil,
						}},
					}
					d, _ := json.Marshal(chunk)
					w.Write([]byte("data: " + string(d) + "\n\n"))
					flusher.Flush()
				}
			}

		case "content_block_start":
			blk, _ := obj["content_block"].(map[string]any)
			if blk != nil && blk["type"] == "tool_use" {
				tcID, _ := blk["id"].(string)
				tcName, _ := blk["name"].(string)
				chunk := map[string]any{
					"id":      chatID,
					"object":  "chat.completion.chunk",
					"created": created,
					"model":   model,
					"choices": []any{map[string]any{
						"index": 0,
						"delta": map[string]any{
							"tool_calls": []any{map[string]any{
								"index": 0,
								"id":    tcID,
								"type":  "function",
								"function": map[string]any{
									"name":      tcName,
									"arguments": "",
								},
							}},
						},
						"finish_reason": nil,
					}},
				}
				d, _ := json.Marshal(chunk)
				w.Write([]byte("data: " + string(d) + "\n\n"))
				flusher.Flush()
			}

		case "message_delta":
			if delta, ok := obj["delta"].(map[string]any); ok {
				if sr, ok := delta["stop_reason"].(string); ok {
					switch sr {
					case "end_turn":
						finishReason = "stop"
					case "max_tokens":
						finishReason = "length"
					case "tool_use":
						finishReason = "tool_calls"
					}
				}
			}
			if u, ok := obj["usage"].(map[string]any); ok {
				if ot, ok := u["output_tokens"].(float64); ok {
					totalOutput = int(ot)
				}
			}

		case "message_stop":
			// 发送结束 chunk
			chunk := map[string]any{
				"id":      chatID,
				"object":  "chat.completion.chunk",
				"created": created,
				"model":   model,
				"choices": []any{map[string]any{
					"index":         0,
					"delta":         map[string]any{},
					"finish_reason": finishReason,
				}},
			}
			d, _ := json.Marshal(chunk)
			w.Write([]byte("data: " + string(d) + "\n\n"))
			flusher.Flush()
			w.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
		}
	}

	// 记录用量
	logEntry := newUsageLog(provider.ID, provider.Name, model, totalInput, totalOutput, "openai")
	addUsageLog(logEntry)
}

// --- 辅助函数 ---

type streamToolAcc struct {
	index   int
	id      string
	name    string
	args    string
	emitted bool
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, format string, status int, message string) {
	if format == "anthropic" {
		writeJSON(w, status, map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    "api_error",
				"message": message,
			},
		})
	} else {
		writeJSON(w, status, map[string]any{
			"error": map[string]any{
				"message": message,
				"type":    "api_error",
			},
		})
	}
}
