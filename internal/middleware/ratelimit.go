package middleware

import (
	"net"
	"net/http"
	"pieces-os-go/internal/config"
	"pieces-os-go/internal/model"
	"strings"
	"sync"
	"time"
)

type visitor struct {
	firstSeen time.Time
	counts    map[string]int
}

type RateLimiter struct {
	visitors  map[string]*visitor
	mu        sync.Mutex
	rules     map[string]config.RateLimitRule
	whitelist map[string]bool
	blacklist *BlacklistManager
}

func NewRateLimiter(cfg *config.Config) *RateLimiter {
	whitelist := make(map[string]bool)
	for _, ip := range cfg.IPWhitelist {
		whitelist[strings.TrimSpace(ip)] = true
	}

	rl := &RateLimiter{
		visitors:  make(map[string]*visitor),
		rules:     cfg.RateLimits,
		whitelist: whitelist,
		blacklist: NewBlacklistManager(cfg),
	}

	// 每小时清理一次过期记录
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

// NewStrictRateLimiter 创建一个只应用严格规则的限流器
func NewStrictRateLimiter(cfg *config.Config) *RateLimiter {
	whitelist := make(map[string]bool)
	for _, ip := range cfg.IPWhitelist {
		whitelist[strings.TrimSpace(ip)] = true
	}

	// 只使用严格规则
	strictRules := map[string]config.RateLimitRule{
		"strict": cfg.RateLimits["strict"],
	}

	return &RateLimiter{
		visitors:  make(map[string]*visitor),
		rules:     strictRules,
		whitelist: whitelist,
		blacklist: NewBlacklistManager(cfg),
	}
}

func (rl *RateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetRealIP(r)

		// 检查是否在黑名单中
		if rl.blacklist.IsBlocked(ip) {
			writeError(w, model.NewAPIError(model.ErrIPBlocked, "IP has been blocked", http.StatusForbidden))
			return
		}

		// 检查白名单
		if rl.whitelist[ip] {
			next.ServeHTTP(w, r)
			return
		}

		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		if !exists {
			v = &visitor{
				firstSeen: time.Now(),
				counts:    make(map[string]int),
			}
			rl.visitors[ip] = v
		}

		// 检查所有启用的规则
		for ruleName, rule := range rl.rules {
			if !rule.Enabled {
				continue
			}

			// 检查是否需要重置计数器
			if time.Since(v.firstSeen) > rule.Window {
				v.firstSeen = time.Now()
				v.counts = make(map[string]int)
			}

			// 检查是否超过限制
			if v.counts[ruleName] >= rule.Limit {
				rl.mu.Unlock()
				// 记录违规
				rl.blacklist.RecordViolation(ip)
				writeError(w, model.NewAPIError(model.ErrRateLimitExceeded, "Rate limit exceeded", http.StatusTooManyRequests))
				return
			}

			v.counts[ruleName]++
		}

		rl.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

// GetRealIP 获取真实IP地址，支持IPv4和IPv6
func GetRealIP(r *http.Request) string {
	// 按优先级尝试获取真实IP
	for _, h := range []string{"X-Real-Ip", "X-Forwarded-For"} {
		addresses := strings.Split(r.Header.Get(h), ",")
		if len(addresses) > 0 {
			// 获取第一个非空地址
			for _, addr := range addresses {
				trimmedAddr := strings.TrimSpace(addr)
				if trimmedAddr != "" {
					return trimmedAddr
				}
			}
		}
	}

	// 如果都没有，则使用RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// 如果分割失败，可能是因为没有端口号
		return r.RemoteAddr
	}
	return ip
}

// GetBlacklist 返回黑名单管理器实例
func (rl *RateLimiter) GetBlacklist() *BlacklistManager {
	return rl.blacklist
}

// 添加清理过期访问者的方法
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, v := range rl.visitors {
		// 清理超过最大窗口期的记录
		maxWindow := time.Duration(0)
		for _, rule := range rl.rules {
			if rule.Window > maxWindow {
				maxWindow = rule.Window
			}
		}
		if now.Sub(v.firstSeen) > maxWindow {
			delete(rl.visitors, ip)
		}
	}
}
