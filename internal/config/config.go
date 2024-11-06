package config

import (
	"log"
	"os"
	"pieces-os-go/internal/model"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
	APIKey               string
	VertexGRPCAddr       string
	GPTGRPCAddr          string
	DefaultModel         string
	MaxRetries           int
	Timeout              int
	Debug                bool
	APIPrefix            string
	LogFile              string
	MinPoolSize          int           `yaml:"min_pool_size"`          // 最小连接数
	MaxPoolSize          int           `yaml:"max_pool_size"`          // 最大连接数
	ScaleInterval        time.Duration `yaml:"scale_interval"`         // 扩缩容检查间隔
	EnableModelRoute     bool          `yaml:"enable_model_route"`     // 是否启用模型路由
	EnableFoolproofRoute bool          `yaml:"enable_foolproof_route"` // 是否启用防呆路由
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	defaultModel := getEnv("DEFAULT_MODEL", "")
	// 检查默认模型是否支持
	if defaultModel != "" && !model.IsModelSupported(defaultModel) {
		log.Printf("Warning: DEFAULT_MODEL '%s' is not supported, setting to empty", defaultModel)
		defaultModel = ""
	}

	apiPrefix := getEnv("API_PREFIX", "/v1")
	// 确保APIPrefix以/开头
	if !strings.HasPrefix(apiPrefix, "/") {
		apiPrefix = "/" + apiPrefix
		log.Printf("Warning: APIPrefix should start with /, auto fixed to: %s", apiPrefix)
	}
	// 确保APIPrefix不以/结尾
	if strings.HasSuffix(apiPrefix, "/") {
		apiPrefix = strings.TrimSuffix(apiPrefix, "/")
		log.Printf("Warning: APIPrefix should not end with /, auto fixed to: %s", apiPrefix)
	}

	return &Config{
		Port:                 getEnv("PORT", "8787"),
		APIKey:               getEnv("API_KEY", ""),
		VertexGRPCAddr:       getEnv("VERTEX_GRPC_ADDR", "runtime-native-io-vertex-inference-grpc-service-lmuw6mcn3q-ul.a.run.app:443"),
		GPTGRPCAddr:          getEnv("GPT_GRPC_ADDR", "runtime-native-io-gpt-inference-grpc-service-lmuw6mcn3q-ul.a.run.app:443"),
		DefaultModel:         defaultModel,
		MaxRetries:           getEnvAsInt("MAX_RETRIES", 3),
		Timeout:              getEnvAsInt("TIMEOUT", 30),
		Debug:                getEnvAsBool("DEBUG", false),
		APIPrefix:            apiPrefix,
		LogFile:              getEnv("LOG_FILE", ""),
		MinPoolSize:          getEnvAsInt("MIN_POOL_SIZE", 5),
		MaxPoolSize:          getEnvAsInt("MAX_POOL_SIZE", 20),
		ScaleInterval:        time.Duration(getEnvAsInt("SCALE_INTERVAL", 30)) * time.Second,
		EnableModelRoute:     getEnvAsBool("ENABLE_MODEL_ROUTE", false),
		EnableFoolproofRoute: getEnvAsBool("ENABLE_FOOLPROOF_ROUTE", false),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
