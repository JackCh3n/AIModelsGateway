package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func corsHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-api-key, anthropic-version, anthropic-beta")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h(w, r)
	}
}

func startServer(port int) error {
	// 预加载配置
	cfg := loadConfig()
	log.Printf("已加载: %d 个中转站, %d 个 API Key", len(cfg.Providers), len(cfg.APIKeys))
	if cfg.Settings.ActiveProviderID != "" {
		if p := getProvider(cfg.Settings.ActiveProviderID); p != nil {
			log.Printf("当前活跃站点: %s (%s)", p.Name, p.Format)
		}
	}

	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("/health", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"version": "1.0",
		})
	}))

	// 管理后台
	registerAdminRoutes(mux)

	// API Key 鉴权中间件
	authMiddleware := func(next http.HandlerFunc, clientFormat string) http.HandlerFunc {
		return corsHandler(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				writeError(w, clientFormat, http.StatusMethodNotAllowed, "method not allowed")
				return
			}

			// 提取 API Key
			key := r.Header.Get("x-api-key")
			if key == "" {
				if b := r.Header.Get("Authorization"); len(b) > 7 && b[:7] == "Bearer " {
					key = b[7:]
				}
			}

			if !validateAPIKey(key) {
				writeError(w, clientFormat, http.StatusUnauthorized, "无效的 API Key，请在管理后台生成")
				return
			}

			next(w, r)
		})
	}

	// OpenAI Chat Completions
	chatHandler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(w, r, "openai", "")
	}, "openai")
	mux.HandleFunc("/v1/chat/completions", chatHandler)
	mux.HandleFunc("/chat/completions", chatHandler)

	// 指定 provider 的 OpenAI 路径: /v1/chat/completions/p/{providerId}
	chatProviderHandler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		pid := strings.TrimPrefix(r.URL.Path, "/v1/chat/completions/p/")
		pid = strings.TrimPrefix(pid, "/chat/completions/p/")
		proxyRequest(w, r, "openai", pid)
	}, "openai")
	mux.HandleFunc("/v1/chat/completions/p/", chatProviderHandler)
	mux.HandleFunc("/chat/completions/p/", chatProviderHandler)

	// Anthropic Messages
	anthropicHandler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(w, r, "anthropic", "")
	}, "anthropic")
	mux.HandleFunc("/v1/messages", anthropicHandler)
	mux.HandleFunc("/messages", anthropicHandler)

	// 指定 provider 的 Anthropic 路径: /v1/messages/p/{providerId}
	anthropicProviderHandler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		pid := strings.TrimPrefix(r.URL.Path, "/v1/messages/p/")
		pid = strings.TrimPrefix(pid, "/messages/p/")
		proxyRequest(w, r, "anthropic", pid)
	}, "anthropic")
	mux.HandleFunc("/v1/messages/p/", anthropicProviderHandler)
	mux.HandleFunc("/messages/p/", anthropicProviderHandler)

	// 模型列表
	modelsHandler := authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
			return
		}
		provider := getActiveProvider()
		models := []map[string]any{}
		if provider != nil {
			for _, m := range provider.Models {
				if !isModelEnabled(provider, m) {
					continue
				}
				models = append(models, map[string]any{
					"id":       m,
					"object":   "model",
					"created":  time.Now().UnixMilli(),
					"owned_by": provider.Name,
				})
			}
		}
		if len(models) == 0 {
			// 返回默认模型列表
			for _, m := range []string{"gpt-4o-mini", "gpt-4o", "claude-3-5-sonnet-20241022", "deepseek-chat"} {
				models = append(models, map[string]any{
					"id":       m,
					"object":   "model",
					"created":  time.Now().UnixMilli(),
					"owned_by": "aimodels",
				})
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": models})
	}, "openai")
	mux.HandleFunc("/v1/models", modelsHandler)
	mux.HandleFunc("/models", modelsHandler)

	// 启动信息
	addr := fmt.Sprintf(":%d", port)
	fmt.Println()
	fmt.Println(strings.Repeat("=", 56))
	fmt.Println("  AI Models Gateway v1.0")
	fmt.Println(strings.Repeat("=", 56))
	fmt.Printf("  服务地址:   http://127.0.0.1:%d\n", port)
	fmt.Printf("  OpenAI:    http://127.0.0.1:%d/v1/chat/completions\n", port)
	fmt.Printf("  Anthropic: http://127.0.0.1:%d/v1/messages\n", port)
	fmt.Printf("  管理后台:   http://127.0.0.1:%d/admin/\n", port)
	fmt.Printf("  中转站:     %d 个", len(cfg.Providers))
	if cfg.Settings.ActiveProviderID != "" {
		if p := getProvider(cfg.Settings.ActiveProviderID); p != nil {
			fmt.Printf(" (活跃: %s)", p.Name)
		}
	}
	fmt.Println()
	fmt.Println(strings.Repeat("=", 56))

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("服务器启动: %s", addr)
	return server.ListenAndServe()
}

// adminPageHandler 返回管理后台页面
func adminPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(adminHTML))
}

// 确保 json 包被使用
var _ = json.Marshal
