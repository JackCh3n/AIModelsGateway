package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 5 * time.Minute,
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
	json.Unmarshal(body, &params)
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

	if model == "" {
		model = getSettings().DefaultModel
		params["model"] = model
		body, _ = json.Marshal(params)
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
			return
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
				return
			}
			upstreamBody, _ = json.Marshal(converted)
		} else if clientFormat == "openai" && provider.Format == "anthropic" {
			// OpenAI 请求 -> Anthropic 请求
			converted, err := openAIToAnthropicReq(body)
			if err != nil {
				writeError(w, clientFormat, http.StatusBadRequest, "convert request: "+err.Error())
				return
			}
			upstreamBody, _ = json.Marshal(converted)
		}
	} else {
		upstreamBody = body
	}

	// 构建上游请求
	upstreamURL := provider.BaseURL
	if provider.Format == "openai" {
		upstreamURL = strings.TrimSuffix(upstreamURL, "/") + "/chat/completions"
	} else {
		upstreamURL = strings.TrimSuffix(upstreamURL, "/") + "/messages"
	}

	req, err := http.NewRequest("POST", upstreamURL, bytes.NewReader(upstreamBody))
	if err != nil {
		writeError(w, clientFormat, http.StatusInternalServerError, "create request: "+err.Error())
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	apiKey := pickAPIKey(provider)
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

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("  upstream error: %v", err)
		writeError(w, clientFormat, http.StatusBadGateway, "upstream request failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("  upstream %d: %s", resp.StatusCode, truncate(string(respBody), 500))
		writeError(w, clientFormat, resp.StatusCode, fmt.Sprintf("upstream %d: %s", resp.StatusCode, truncate(string(respBody), 300)))
		return
	}

	if isStream {
		handleStreamProxy(w, resp, clientFormat, provider.Format, provider, model)
	} else {
		handleNonStreamProxy(w, resp, clientFormat, provider.Format, provider, model)
	}
}

func handleNonStreamProxy(w http.ResponseWriter, resp *http.Response, clientFormat, providerFormat string, provider *Provider, model string) {
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		writeError(w, clientFormat, http.StatusInternalServerError, "decode response: "+err.Error())
		return
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
		// 直通流式响应，同时收集用量
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		reader := bufio.NewReader(resp.Body)
		var totalInput, totalOutput int
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
					}
				}
			}
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
