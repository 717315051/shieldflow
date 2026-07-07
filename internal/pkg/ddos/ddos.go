// Package ddos 实现基于配置管理与 nftables 规则的 DDoS 防护。
//
// 注意：eBPF/XDP 数据面由 Rust 程序实现（见 ebpf/ 目录），本包负责：
//   - 配置管理（连接数/PPS/自动封禁阈值）
//   - 黑白名单管理
//   - nftables 规则下发（Go 层 stub）
//   - 与 Rust eBPF 程序通过共享 map 通信（CGO / 配置文件）
//
// 设计参考：
//   max_connections_per_ip   单 IP 最大并发连接数
//   new_connections_per_sec  单 IP 新建连接速率
//   max_packets_per_sec      单 IP 最大包速率
//   auto_ban_enabled         自动封禁开关
//   ban_threshold_connections 触发封禁的连接数阈值
//   ban_threshold_packets      触发封禁的包速率阈值
//   ban_duration_seconds       封禁时长
package ddos

import (
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"
)

// Config DDoS 防护配置。
type Config struct {
	// eBPF / 内核级限制
	MaxConnectionsPerIP  int   `json:"max_connections_per_ip"`
	NewConnectionsPerSec int   `json:"new_connections_per_sec"`
	MaxPacketsPerSec     int   `json:"max_packets_per_sec"`

	// 自动封禁
	AutoBanEnabled           bool `json:"auto_ban_enabled"`
	BanThresholdConnections  int  `json:"ban_threshold_connections"`
	BanThresholdPackets      int  `json:"ban_threshold_packets"`
	BanDurationSeconds       int  `json:"ban_duration_seconds"`

	// 黑白名单
	Blacklist []string `json:"blacklist"`
	Whitelist []string `json:"whitelist"`

	// UseIPSet 是否使用 ipset 管理黑白名单（而非 nftables）。
	UseIPSet bool `json:"use_ipset"`
}

// ipsetBlacklistName ipset 黑名单集合名称。
const ipsetBlacklistName = "shieldflow-blacklist"

// Guard DDoS 防护管理器。
type Guard struct {
	mu        sync.RWMutex
	cfg       Config
	banned    map[string]*banEntry // ip -> 封禁信息
	connCount map[string]int       // ip -> 当前并发连接数（应用层近似）
	pktCount  map[string]*slidingCounter // ip -> 滑动窗口计数器
}

type banEntry struct {
	unbanAt time.Time
	reason  string
}

// NewGuard 创建 DDoS 防护管理器。
func NewGuard(cfg Config) *Guard {
	if cfg.MaxConnectionsPerIP <= 0 {
		cfg.MaxConnectionsPerIP = 1000
	}
	if cfg.NewConnectionsPerSec <= 0 {
		cfg.NewConnectionsPerSec = 50
	}
	if cfg.MaxPacketsPerSec <= 0 {
		cfg.MaxPacketsPerSec = 2000
	}
	if cfg.BanThresholdConnections <= 0 {
		cfg.BanThresholdConnections = 2000
	}
	if cfg.BanThresholdPackets <= 0 {
		cfg.BanThresholdPackets = 5000
	}
	if cfg.BanDurationSeconds <= 0 {
		cfg.BanDurationSeconds = 3600
	}
	g := &Guard{
		cfg:       cfg,
		banned:    map[string]*banEntry{},
		connCount: map[string]int{},
		pktCount:  map[string]*slidingCounter{},
	}
	// 如果启用 ipset，初始化 ipset 集合。
	if cfg.UseIPSet {
		g.initIPSet()
	}
	// 应用初始黑白名单到 nftables / ipset。
	g.applyBlacklist()
	g.applyWhitelist()
	// 启动封禁过期清理。
	go g.evictor()
	return g
}

// Allow 判断某 IP 的请求是否允许通过（应用层近似判定）。
//
// 实际的 L3/L4 限速由 eBPF 程序在内核态完成，这里提供应用层的补充判断，
// 并在达到阈值时下发 nftables 封禁规则。
func (g *Guard) Allow(ip string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 白名单永远放行。
	if g.isWhitelistedLocked(ip) {
		return true
	}
	// 黑名单永远拒绝。
	if g.isBlacklistedLocked(ip) {
		return false
	}
	// 已封禁：未过期则拒绝。
	if entry, ok := g.banned[ip]; ok {
		if time.Now().Before(entry.unbanAt) {
			return false
		}
		delete(g.banned, ip)
	}

	// 统计包速率（应用层近似：每次请求 +1）。
	sc := g.pktCount[ip]
	if sc == nil {
		sc = newSlidingCounter(time.Second)
		g.pktCount[ip] = sc
	}
	sc.add(1)

	// 连接数近似：+1（实际应由 eBPF 提供）。
	g.connCount[ip]++
	if g.connCount[ip] > g.cfg.MaxConnectionsPerIP {
		if g.cfg.AutoBanEnabled {
			g.banLocked(ip, fmt.Sprintf("connections=%d > %d", g.connCount[ip], g.cfg.MaxConnectionsPerIP))
		}
		return false
	}
	if pps := sc.count(); pps > g.cfg.MaxPacketsPerSec {
		if g.cfg.AutoBanEnabled {
			g.banLocked(ip, fmt.Sprintf("pps=%d > %d", pps, g.cfg.MaxPacketsPerSec))
		}
		return false
	}
	return true
}

// Release 请求结束时释放连接计数。
func (g *Guard) Release(ip string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.connCount[ip] > 0 {
		g.connCount[ip]--
	}
}

// UpdateConfig 动态更新配置。
func (g *Guard) UpdateConfig(cfg Config) {
	g.mu.Lock()
	g.cfg = cfg
	g.mu.Unlock()
	g.applyBlacklist()
	g.applyWhitelist()
}

// Ban 手动封禁某 IP。
func (g *Guard) Ban(ip, reason string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.banLocked(ip, reason)
}

// banLocked 加锁版本封禁。
func (g *Guard) banLocked(ip, reason string) {
	dur := time.Duration(g.cfg.BanDurationSeconds) * time.Second
	g.banned[ip] = &banEntry{
		unbanAt: time.Now().Add(dur),
		reason:  reason,
	}
	// 下发封禁规则：ipset 或 nftables。
	if g.cfg.UseIPSet {
		g.ipsetBan(ip, dur)
	} else {
		g.nftBan(ip, dur)
	}
}

// isBlacklistedLocked 判断 IP 是否在配置黑名单（支持 CIDR）。
func (g *Guard) isBlacklistedLocked(ip string) bool {
	return matchList(g.cfg.Blacklist, ip)
}

// isWhitelistedLocked 判断 IP 是否在配置白名单（支持 CIDR）。
func (g *Guard) isWhitelistedLocked(ip string) bool {
	return matchList(g.cfg.Whitelist, ip)
}

func (g *Guard) applyBlacklist() {
	// 实际下发 nftables / ipset：nft add element ip shieldflow-ddos blacklist { ip }
	// 这里是 stub，记录到日志即可。
	for _, ip := range g.cfg.Blacklist {
		if g.cfg.UseIPSet {
			g.ipsetBan(ip, 0)
		} else {
			g.nftBan(ip, 0)
		}
	}
}

func (g *Guard) applyWhitelist() {
	// 实际下发 nftables / ipset 白名单集合。
	for _, ip := range g.cfg.Whitelist {
		if g.cfg.UseIPSet {
			g.ipsetAllow(ip)
		} else {
			g.nftAllow(ip)
		}
	}
}

// nftBan 下发 nftables 封禁规则（stub，实际需调用 nft 命令）。
//
// 真实命令示例：
//   nft add element inet shieldflow-filter blacklist_v4 { <ip> timeout <dur>s }
func (g *Guard) nftBan(ip string, dur time.Duration) {
	// stub：在生产环境中执行 nft 命令。
	_ = exec.Command("nft", "list", "tables").Run()
	_ = ip
	_ = dur
}

// nftAllow 下发 nftables 白名单规则（stub）。
func (g *Guard) nftAllow(ip string) {
	_ = ip
}

// initIPSet 初始化 ipset 黑名单集合。
//
// 执行: ipset create shieldflow-blacklist hash:ip timeout 0
// 若集合已存在则忽略错误（ipset restore 幂等）。
func (g *Guard) initIPSet() {
	// 尝试创建集合，若已存在则忽略错误。
	_ = exec.Command("ipset", "create", ipsetBlacklistName, "hash:ip", "timeout", "0").Run()
}

// ipsetBan 通过 ipset 封禁 IP。
//
// 执行: ipset add shieldflow-blacklist <ip> [timeout <dur>s]
// dur > 0 时附带超时（自动解封），dur == 0 表示永久封禁。
func (g *Guard) ipsetBan(ip string, dur time.Duration) {
	if dur > 0 {
		_ = exec.Command("ipset", "add", ipsetBlacklistName, ip,
			"timeout", fmt.Sprintf("%d", int(dur.Seconds())),
			"-!").Run()
	} else {
		_ = exec.Command("ipset", "add", ipsetBlacklistName, ip, "-!").Run()
	}
}

// ipsetAllow 通过 ipset 解封 IP（从黑名单集合移除）。
//
// 执行: ipset del shieldflow-blacklist <ip>
func (g *Guard) ipsetAllow(ip string) {
	_ = exec.Command("ipset", "del", ipsetBlacklistName, ip, "-!").Run()
}

// evictor 定期清理过期封禁。
func (g *Guard) evictor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		g.mu.Lock()
		now := time.Now()
		for ip, entry := range g.banned {
			if now.After(entry.unbanAt) {
				delete(g.banned, ip)
			}
		}
		g.mu.Unlock()
	}
}

// Stats 返回当前防护统计。
type Stats struct {
	BannedIPs     int `json:"banned_ips"`
	TrackedIPs    int `json:"tracked_ips"`
	BlacklistSize int `json:"blacklist_size"`
	WhitelistSize int `json:"whitelist_size"`
}

// Stats 返回统计快照。
func (g *Guard) Stats() Stats {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return Stats{
		BannedIPs:     len(g.banned),
		TrackedIPs:    len(g.connCount),
		BlacklistSize: len(g.cfg.Blacklist),
		WhitelistSize: len(g.cfg.Whitelist),
	}
}

// ---- CIDR 匹配 ----

func matchList(list []string, ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, entry := range list {
		if entry == ip {
			return true
		}
		if _, cidr, err := net.ParseCIDR(entry); err == nil {
			if cidr.Contains(parsed) {
				return true
			}
		}
	}
	return false
}

// ---- 滑动窗口计数器 ----

type slidingCounter struct {
	window time.Duration
	buckets []bucket
	mu      sync.Mutex
}

type bucket struct {
	timestamp time.Time
	count     int
}

func newSlidingCounter(window time.Duration) *slidingCounter {
	return &slidingCounter{window: window}
}

func (s *slidingCounter) add(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.evictLocked(now)
	s.buckets = append(s.buckets, bucket{timestamp: now, count: n})
}

func (s *slidingCounter) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evictLocked(time.Now())
	total := 0
	for _, b := range s.buckets {
		total += b.count
	}
	return total
}

func (s *slidingCounter) evictLocked(now time.Time) {
	cutoff := now.Add(-s.window)
	kept := s.buckets[:0]
	for _, b := range s.buckets {
		if b.timestamp.After(cutoff) {
			kept = append(kept, b)
		}
	}
	s.buckets = kept
}

// ebpfAttach 是 Rust eBPF 程序加载的占位函数。
//
// 实际实现：Rust 程序编译为 .o，通过 libbpf 加载到 XDP/tc hook。
// Go 通过 CGO 或配置文件向 Rust 程序传递限速参数。
// 占位函数仅返回 nil，表示需要 Rust 实现。
func ebpfAttach() error {
	// TODO: 由 Rust 实现 eBPF/XDP 程序加载。
	return nil
}
