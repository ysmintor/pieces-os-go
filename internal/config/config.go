package config

import (
	"log"
	"math/rand"
	"os"
	"pieces-os-go/internal/model"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type RateLimitRule struct {
	Limit   int           `yaml:"limit"`
	Window  time.Duration `yaml:"window"`
	Enabled bool          `yaml:"enabled"`
}

type Config struct {
	Port                 string
	APIKey               string
	AdminKey             string
	VertexGRPCAddr       string
	GPTGRPCAddr          string
	DefaultModel         string
	MaxRetries           int
	Timeout              int
	Debug                bool
	APIPrefix            string
	LogFile              string
	MinPoolSize          int                      `yaml:"min_pool_size"`          // 最小连接数
	MaxPoolSize          int                      `yaml:"max_pool_size"`          // 最大连接数
	ScaleInterval        time.Duration            `yaml:"scale_interval"`         // 扩缩容检查间隔
	EnableModelRoute     bool                     `yaml:"enable_model_route"`     // 是否启用模型路由
	EnableFoolproofRoute bool                     `yaml:"enable_foolproof_route"` // 是否启用防呆路由
	RequestTimeout       time.Duration            `yaml:"request_timeout"`        // 普通请求超时时间
	StreamTimeout        time.Duration            `yaml:"stream_timeout"`         // 流式请求超时时间
	RateLimits           map[string]RateLimitRule `yaml:"rate_limits"`            // 多个限流规则
	IPWhitelist          []string                 `yaml:"ip_whitelist"`           // IP白名单
	IPBlacklist          []string                 `yaml:"ip_blacklist"`           // 配置的IP黑名单
	BlacklistMode        string                   `yaml:"blacklist_mode"`         // 黑名单模式：off/single/subnet
	BlacklistThreshold   int                      `yaml:"blacklist_threshold"`    // 触发自动拉黑的阈值
	BlacklistFile        string                   `yaml:"blacklist_file"`         // 黑名单文件路径
	IPv4Mask             int                      `yaml:"ipv4_mask"`              // 默认24
	IPv6Mask             int                      `yaml:"ipv6_mask"`              // 默认48
}

// 添加新的辅助函数用于生成随机字符串
func generateRandomString(minLength, maxLength int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	length := minLength + rand.Intn(maxLength-minLength+1)
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// 添加掩码范围常量
const (
	MinIPv4Mask = 8   // 最小 /8，避免封禁过大网段
	MaxIPv4Mask = 32  // 最大 /32，单个IP
	MinIPv6Mask = 32  // 最小 /32，避免封禁过大网段
	MaxIPv6Mask = 128 // 最大 /128，单个IP

	DefaultIPv4Mask = 24 // 默认 /24
	DefaultIPv6Mask = 48 // 默认 /48
)

// 添加掩码验证函数
func validateMask(mask int, min, max int, defaultValue int) int {
	if mask < min || mask > max {
		log.Printf("Warning: Invalid mask value %d, should be between %d and %d, using default value %d",
			mask, min, max, defaultValue)
		return defaultValue
	}
	return mask
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

	// 获取或生成ADMIN_KEY
	adminKey := getEnv("ADMIN_KEY", "")
	if adminKey == "" {
		adminKey = generateRandomString(32, 64)
		log.Printf("Generated random ADMIN_KEY: %s", adminKey)
	}

	// 定义默认的限流规则
	defaultRateLimits := map[string]RateLimitRule{
		"default": {
			Limit:   getEnvAsInt("RATE_LIMIT", 60),
			Window:  time.Duration(getEnvAsInt("RATE_LIMIT_WINDOW", 60)) * time.Second,
			Enabled: getEnvAsBool("RATE_LIMIT_ENABLED", true),
		},
		"strict": {
			Limit:   getEnvAsInt("STRICT_RATE_LIMIT", 10),
			Window:  time.Duration(getEnvAsInt("STRICT_RATE_LIMIT_WINDOW", 60)) * time.Second,
			Enabled: getEnvAsBool("STRICT_RATE_LIMIT_ENABLED", false),
		},
		"burst": {
			Limit:   getEnvAsInt("BURST_RATE_LIMIT", 100),
			Window:  time.Duration(getEnvAsInt("BURST_RATE_LIMIT_WINDOW", 1)) * time.Second,
			Enabled: getEnvAsBool("BURST_RATE_LIMIT_ENABLED", false),
		},
	}

	// 验证并设置掩码值
	ipv4Mask := validateMask(
		getEnvAsInt("IPV4_MASK", DefaultIPv4Mask),
		MinIPv4Mask,
		MaxIPv4Mask,
		DefaultIPv4Mask,
	)

	ipv6Mask := validateMask(
		getEnvAsInt("IPV6_MASK", DefaultIPv6Mask),
		MinIPv6Mask,
		MaxIPv6Mask,
		DefaultIPv6Mask,
	)

	return &Config{
		Port:                 getEnv("PORT", "8787"),
		APIKey:               getEnv("API_KEY", ""),
		AdminKey:             adminKey,
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
		RequestTimeout:       time.Duration(getEnvAsInt("REQUEST_TIMEOUT", 30)) * time.Second,
		StreamTimeout:        time.Duration(getEnvAsInt("STREAM_TIMEOUT", 300)) * time.Second,
		RateLimits:           defaultRateLimits,
		IPWhitelist:          getEnvAsStringSlice("IP_WHITELIST", []string{}),
		IPBlacklist:          getEnvAsStringSlice("IP_BLACKLIST", []string{}),
		BlacklistMode:        getEnv("BLACKLIST_MODE", "single"),
		BlacklistThreshold:   getEnvAsInt("BLACKLIST_THRESHOLD", 100),
		BlacklistFile:        getEnv("BLACKLIST_FILE", "blacklist.txt"),
		IPv4Mask:             ipv4Mask,
		IPv6Mask:             ipv6Mask,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return strings.TrimSpace(value)
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

func getEnvAsStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		// 处理每个部分的空格
		result := make([]string, len(parts))
		for i, part := range parts {
			result[i] = strings.TrimSpace(part)
		}
		return result
	}
	return defaultValue
}
