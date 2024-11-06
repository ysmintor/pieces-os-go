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
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// 全局变量
var (
	Version           = "dev"
	BuildTime         = "unknown"
	foolproofPathsMap map[string]bool
	initFoolproofOnce sync.Once
)

// getFoolproofPaths 获取防呆路由
func getFoolproofPaths(apiPrefix string) map[string]bool {
	if foolproofPathsMap != nil {
		return foolproofPathsMap
	}

	initFoolproofOnce.Do(func() {
		// 基础路径段
		segments := []string{
			"/chat/completions",
			"/completions",
		}

		// 使用 map 来去重
		pathsMap := make(map[string]bool)

		// 生成所有可能的路径组合
		for _, seg1 := range segments {
			// 添加基本路径
			pathsMap[seg1] = true

			// 添加带 API 前缀的路径
			if apiPrefix != "" {
				pathsMap[apiPrefix+seg1] = true
			}

			// 添加组合路径
			for _, seg2 := range segments {
				// 基本组合路径
				pathsMap[seg1+seg2] = true

				// 带 API 前缀的组合路径
				if apiPrefix != "" {
					pathsMap[apiPrefix+seg1+seg2] = true
					pathsMap[seg1+apiPrefix+seg2] = true
					pathsMap[apiPrefix+seg1+apiPrefix+seg2] = true
				}
			}
		}

		// 移除标准路由
		standardPath := "/chat/completions"
		if apiPrefix != "" {
			standardPath = apiPrefix + "/chat/completions"
		}
		delete(pathsMap, standardPath)

		foolproofPathsMap = pathsMap
	})
	return foolproofPathsMap
}

func main() {
	// 打印版本信息
	log.Printf("Starting Pieces-OS-Go Version: %s, BuildTime: %s", Version, BuildTime)

	if err := model.InitModels(); err != nil {
		log.Fatalf("Failed to initialize models: %v", err)
	}
	cfg := config.Load()
	if err := middleware.InitLogger(cfg.LogFile); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	if err := tokenizer.InitTokenizers(); err != nil {
		log.Fatalf("Failed to initialize tokenizers: %v", err)
	}
	if cfg.EnableFoolproofRoute && cfg.APIPrefix == "" {
		cfg.EnableFoolproofRoute = false
		log.Printf("Warning: Foolproof routing is not supported when APIPrefix is empty, automatically disabled. Recommend using /v1 as prefix")
	}

	r := chi.NewRouter()

	// 创建处理器实例
	chatHandler := handler.NewChatHandler(cfg)

	// 全局中间件
	r.Use(middleware.Logger(cfg))
	r.Use(middleware.CORS)

	// 自定义 404 处理器
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": string(model.ErrRouteNotFound),
			"path":  r.URL.Path,
		})
	})

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

	// 防呆路由
	if cfg.EnableFoolproofRoute {
		// 遍历预定义的路径
		for path := range getFoolproofPaths(cfg.APIPrefix) {
			path := path // 创建新的变量作用域
			r.Route(path, func(r chi.Router) {
				// 添加认证中间件
				if cfg.APIKey != "" {
					r.Use(middleware.Auth(cfg.APIKey))
				}

				r.Post("/", func(w http.ResponseWriter, r *http.Request) {
					standardPath := cfg.APIPrefix + "/chat/completions"
					w.Header().Set("X-Warning", "Non-standard path, please use: "+standardPath)
					chatHandler.HandleCompletion(w, r)
				})
			})
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
	log.Printf("本地可通过以下地址访问服务:")
	log.Printf("- http://localhost%s/", serverAddr)
	log.Printf("- http://127.0.0.1%s/", serverAddr)
	log.Printf("- http://[::1]%s/", serverAddr)
	log.Fatal(http.ListenAndServe(serverAddr, r))
}
