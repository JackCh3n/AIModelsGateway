package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Version 版本号，编译时通过 ldflags 注入
var Version = "dev"

// exeDir 返回可执行文件所在目录（与数据目录解析逻辑保持一致）
func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

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

	// 如果启动时已配置狂暴模式，连接 Redis
	if cfg.Settings.RageMode && cfg.Settings.RedisAddr != "" {
		initRedis(cfg.Settings.RedisAddr, cfg.Settings.RedisPassword, cfg.Settings.RedisDB)
	}

	mux := http.NewServeMux()

	// 管理后台登录/登出（无需鉴权，登录接口本身校验密码）
	mux.HandleFunc("/admin/api/login", corsHandler(adminLoginHandler))
	mux.HandleFunc("/admin/api/logout", corsHandler(adminLogoutHandler))

	// 健康检查
	mux.HandleFunc("/health", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"version": Version,
		})
	}))

	// 管理后台
	registerAdminRoutes(mux)

	// 静态文件服务（Chart.js 等）：路径基于可执行文件目录解析，
	// 避免从其他工作目录启动时静态资源 404
	staticDir := filepath.Join(exeDir(), "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// API Key 鉴权中间件
	// allowMethods: 允许的 HTTP 方法（默认 POST）
	authMiddleware := func(next http.HandlerFunc, clientFormat string, allowMethods ...string) http.HandlerFunc {
		if len(allowMethods) == 0 {
			allowMethods = []string{"POST"}
		}
		return corsHandler(func(w http.ResponseWriter, r *http.Request) {
			methodOK := false
			for _, m := range allowMethods {
				if r.Method == m {
					methodOK = true
					break
				}
			}
			if !methodOK {
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
					"created":  time.Now().Unix(), // OpenAI 规范为 Unix 秒
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
					"created":  time.Now().Unix(),
					"owned_by": "aimodels",
				})
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": models})
	}, "openai", "GET", "POST")
	mux.HandleFunc("/v1/models", modelsHandler)
	mux.HandleFunc("/models", modelsHandler)

	// 启动信息
	addr := fmt.Sprintf(":%d", port)
	// 控制台横幅版本：优先使用编译时注入的版本（-ldflags -X main.Version=...），
	// 未注入（go run 调试）时显示 v1.0 兜底
	dispVer := Version
	if dispVer == "" || dispVer == "dev" {
		dispVer = "v1.0"
	}
	fmt.Println()
	fmt.Println(strings.Repeat("=", 56))
	fmt.Printf("  AI Models Gateway %s\n", dispVer)
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
		Addr:              addr,
		Handler:           adminAuth(mux), // 管理后台 API 可选登录保护
		ReadHeaderTimeout: 10 * time.Second, // 防止 slowloris 攻击
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      5 * time.Minute,  // 兼容流式响应
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,          // 1MB
	}

	// 优雅关闭：收到信号后等待活跃连接结束，刷盘用量日志
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Printf("收到关闭信号，正在优雅关闭...")
		flushUsageLogs()
		flushErrorLogs()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("强制关闭: %v", err)
		}
	}()

	log.Printf("服务器启动: %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	log.Printf("服务器已关闭")
	return nil
}

// adminPageHandler 返回管理后台页面
func adminPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	html := strings.Replace(adminHTML, "{{VERSION}}", Version, 1)
	w.Write([]byte(html))
}

// 确保 json 包被使用
var _ = json.Marshal
