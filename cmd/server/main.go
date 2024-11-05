package main

import (
	"encoding/json"
	"log"
	"net/http"
	"pieces-os-go/internal/config"
	"pieces-os-go/internal/handler"
	"pieces-os-go/internal/middleware"
	"pieces-os-go/internal/model"
	"pieces-os-go/pkg/tokenizer"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// 添加版本和构建时间变量
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// 打印版本信息
	log.Printf("Starting Pieces-OS-Go Version: %s, BuildTime: %s", Version, BuildTime)

	cfg := config.Load()
	if err := middleware.InitLogger(cfg.LogFile); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	if err := tokenizer.InitTokenizers(); err != nil {
		log.Fatalf("Failed to initialize tokenizers: %v", err)
	}

	r := chi.NewRouter()

	// 自定义 404 处理器
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "路由不存在",
			"path":  r.URL.Path,
		})
	})

	// 创建处理器实例
	chatHandler := handler.NewChatHandler(cfg)

	// 全局中间件
	r.Use(middleware.Logger)
	r.Use(middleware.CORS)

	// 健康检查路由
	r.Get("/", handler.HealthCheck)
	r.Get("/ping", handler.Ping)

	// API路由组
	r.Route(cfg.APIPrefix, func(r chi.Router) {
		// API认证中间件只应用于此路由组
		if cfg.APIKey != "" {
			r.Use(middleware.Auth(cfg.APIKey))
		}

		// API endpoints
		r.Get("/models", handler.ListModels)
		r.Post("/chat/completions", chatHandler.HandleCompletion)
	})

	// 如果启用了模型路由，添加带模型名的路由
	if cfg.EnableModelRoute {
		for model := range model.SupportedModels {
			// 使用标准化的模型名称作为路由
			modelPath := "/" + model + cfg.APIPrefix
			r.Route(modelPath, func(r chi.Router) {
				if cfg.APIKey != "" {
					r.Use(middleware.Auth(cfg.APIKey))
				}
				r.Post("/chat/completions", handler.WithModel(chatHandler.HandleCompletion, model))
			})

			// 如果是 Claude 模型，添加使用 "-" 格式的路由
			if strings.HasPrefix(model, "claude-") && strings.Contains(model, "@") {
				// 将 "@" 转换为 "-"
				legacyModel := strings.Replace(model, "@", "-", 1)
				legacyPath := "/" + legacyModel + cfg.APIPrefix
				r.Route(legacyPath, func(r chi.Router) {
					if cfg.APIKey != "" {
						r.Use(middleware.Auth(cfg.APIKey))
					}
					r.Post("/chat/completions", handler.WithModel(chatHandler.HandleCompletion, legacyModel))
				})
			}
		}
	}

	// 每秒重置RPS计数器
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			handler.ResetSecondCounters()
		}
	}()

	// 每分钟重置RPM计数器
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			handler.ResetMinuteCounters()
		}
	}()

	// 分别打印服务器地址和API端点信息
	serverAddr := ":" + cfg.Port
	apiEndpoint := cfg.APIPrefix
	log.Printf("Server starting on port %s", cfg.Port)
	log.Printf("API endpoint available at %s", apiEndpoint)
	log.Fatal(http.ListenAndServe(serverAddr, r))
}
