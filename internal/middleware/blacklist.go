package middleware

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"pieces-os-go/internal/config"
	"strings"
	"sync"
	"time"
)

type BlacklistManager struct {
	mu            sync.RWMutex
	blacklist     map[string]bool       // IP黑名单
	subnetList    map[string]*net.IPNet // IP段黑名单
	violations    map[string]int        // IP违规计数
	configuredIPs map[string]bool       // 配置文件中的黑名单
	threshold     int
	mode          string
	blacklistFile string
	ipv4Mask      int // 添加IPv4掩码配置
	ipv6Mask      int // 添加IPv6掩码配置
}

type BlacklistEntry struct {
	IP         string    `json:"ip"`
	Type       string    `json:"type"` // "ip" 或 "subnet"
	AddedAt    time.Time `json:"added_at"`
	Violations int       `json:"violations"`
}

// 添加黑名单统计
type BlacklistStats struct {
	TotalBlocked    int
	BlockedSubnets  int
	ActiveViolators int
}

// 添加掩码验证方法
func (bm *BlacklistManager) validateMasks() {
	if bm.ipv4Mask < config.MinIPv4Mask || bm.ipv4Mask > config.MaxIPv4Mask {
		log.Printf("Warning: Invalid IPv4 mask %d, using default %d",
			bm.ipv4Mask, config.DefaultIPv4Mask)
		bm.ipv4Mask = config.DefaultIPv4Mask
	}

	if bm.ipv6Mask < config.MinIPv6Mask || bm.ipv6Mask > config.MaxIPv6Mask {
		log.Printf("Warning: Invalid IPv6 mask %d, using default %d",
			bm.ipv6Mask, config.DefaultIPv6Mask)
		bm.ipv6Mask = config.DefaultIPv6Mask
	}
}

func NewBlacklistManager(cfg *config.Config) *BlacklistManager {
	bm := &BlacklistManager{
		blacklist:     make(map[string]bool),
		subnetList:    make(map[string]*net.IPNet),
		violations:    make(map[string]int),
		configuredIPs: make(map[string]bool),
		threshold:     cfg.BlacklistThreshold,
		mode:          cfg.BlacklistMode,
		blacklistFile: cfg.BlacklistFile,
		ipv4Mask:      cfg.IPv4Mask,
		ipv6Mask:      cfg.IPv6Mask,
	}

	// 验证掩码值
	bm.validateMasks()

	// 加载配置的黑名单
	for _, ip := range cfg.IPBlacklist {
		bm.configuredIPs[ip] = true
		bm.blacklist[ip] = true
	}

	// 从文件加载自动生成的黑名单
	bm.loadFromFile()

	// 每24小时清理一次违规记录
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			bm.cleanupViolations()
		}
	}()

	return bm
}

func (bm *BlacklistManager) loadFromFile() error {
	file, err := os.OpenFile(bm.blacklistFile, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry BlacklistEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.Type == "subnet" {
			_, ipnet, err := net.ParseCIDR(entry.IP)
			if err == nil {
				bm.subnetList[entry.IP] = ipnet
			}
		} else {
			bm.blacklist[entry.IP] = true
		}
	}
	return scanner.Err()
}

func (bm *BlacklistManager) saveToFile() error {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	file, err := os.Create(bm.blacklistFile)
	if err != nil {
		return err
	}
	defer file.Close()

	entries := []BlacklistEntry{}
	for ip := range bm.blacklist {
		if !bm.configuredIPs[ip] {
			entries = append(entries, BlacklistEntry{
				IP:         ip,
				Type:       "ip",
				AddedAt:    time.Now(),
				Violations: bm.violations[ip],
			})
		}
	}
	for cidr := range bm.subnetList {
		entries = append(entries, BlacklistEntry{
			IP:      cidr,
			Type:    "subnet",
			AddedAt: time.Now(),
		})
	}

	encoder := json.NewEncoder(file)
	return encoder.Encode(entries)
}

// HandleBlacklist 处理黑名单查询API
func (bm *BlacklistManager) HandleBlacklist(w http.ResponseWriter, r *http.Request) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	entries := []BlacklistEntry{}
	for ip := range bm.blacklist {
		if !bm.configuredIPs[ip] {
			entries = append(entries, BlacklistEntry{
				IP:         ip,
				Type:       "ip",
				Violations: bm.violations[ip],
			})
		}
	}
	for cidr := range bm.subnetList {
		entries = append(entries, BlacklistEntry{
			IP:   cidr,
			Type: "subnet",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// 添加一个辅助函数来处理IP地址
func (bm *BlacklistManager) parseIPAndSubnet(ip string) (net.IP, *net.IPNet, error) {
	// 尝试解析CIDR
	if strings.Contains(ip, "/") {
		_, subnet, err := net.ParseCIDR(ip)
		if err == nil {
			// 验证CIDR掩码是否在允许范围内
			ones, bits := subnet.Mask.Size()
			if bits == 32 && (ones < config.MinIPv4Mask || ones > config.MaxIPv4Mask) {
				return nil, nil, fmt.Errorf("invalid IPv4 mask: /%d", ones)
			}
			if bits == 128 && (ones < config.MinIPv6Mask || ones > config.MaxIPv6Mask) {
				return nil, nil, fmt.Errorf("invalid IPv6 mask: /%d", ones)
			}
			return nil, subnet, nil
		}
	}

	// 解析普通IP地址
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	// 根据IP类型返回对应的子网掩码
	var mask net.IPMask
	if parsedIP.To4() != nil {
		// IPv4使用配置的掩码
		mask = net.CIDRMask(bm.ipv4Mask, 32)
	} else {
		// IPv6使用配置的掩码
		mask = net.CIDRMask(bm.ipv6Mask, 128)
	}

	return parsedIP, &net.IPNet{IP: parsedIP.Mask(mask), Mask: mask}, nil
}

func (bm *BlacklistManager) IsBlocked(ipStr string) bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	// 检查IP是否在黑名单中
	if bm.blacklist[ipStr] {
		return true
	}

	// 解析IP地址
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// 检查IP是否在任何被封禁的子网中
	for _, subnet := range bm.subnetList {
		if subnet.Contains(ip) {
			return true
		}
	}

	return false
}

func (bm *BlacklistManager) RecordViolation(ip string) {
	if bm.mode == "off" {
		return
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.violations[ip]++
	if bm.violations[ip] >= bm.threshold {
		if bm.mode == "subnet" {
			parsedIP, _, err := bm.parseIPAndSubnet(ip)
			if err == nil {
				var cidr string
				if parsedIP.To4() != nil {
					// IPv4使用配置的掩码
					cidr = fmt.Sprintf("%s/%d", parsedIP.Mask(net.CIDRMask(bm.ipv4Mask, 32)).String(), bm.ipv4Mask)
				} else {
					// IPv6使用配置的掩码
					cidr = fmt.Sprintf("%s/%d", parsedIP.Mask(net.CIDRMask(bm.ipv6Mask, 128)).String(), bm.ipv6Mask)
				}
				_, ipnet, err := net.ParseCIDR(cidr)
				if err == nil {
					bm.subnetList[cidr] = ipnet
				}
			}
		} else if bm.mode == "single" {
			bm.blacklist[ip] = true
		}
		bm.saveToFile()
	}
}

// 获取黑名单统计信息
func (bm *BlacklistManager) GetStats() BlacklistStats {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	stats := BlacklistStats{
		TotalBlocked:   len(bm.blacklist),
		BlockedSubnets: len(bm.subnetList),
	}

	// 统计活跃违规者
	for _, count := range bm.violations {
		if count > 0 {
			stats.ActiveViolators++
		}
	}

	return stats
}

// 添加清理过期违规记录的方法
func (bm *BlacklistManager) cleanupViolations() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// 清理24小时前的违规记录
	for ip, count := range bm.violations {
		if count == 0 {
			delete(bm.violations, ip)
		}
	}
}
