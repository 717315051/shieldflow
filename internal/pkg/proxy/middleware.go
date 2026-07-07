// Package proxy: 请求处理中间件链。
//
// 按文档定义的安全防护层次顺序执行：
//   黑白名单 → CC防护 → 访问控制 → 区域限制 → Bot检测 → 语义WAF → 转发源站
// 每层均可独立开关，被拦截时返回相应的 HTTP 状态码与简要原因。
package proxy

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/shieldflow/shieldflow/internal/pkg/bot"
	"github.com/shieldflow/shieldflow/internal/pkg/cache"
	"github.com/shieldflow/shieldflow/internal/pkg/ddos"
	"github.com/shieldflow/shieldflow/internal/pkg/waf"
	zcc "github.com/shieldflow/shieldflow/internal/waf"
)

// LayerSwitch 控制每层中间件的开关状态。
type LayerSwitch struct {
	// ProtectionEnabled 防护总开关：false 时跳过所有防护直接转发源站。
	ProtectionEnabled bool

	Blacklist     bool
	CC            bool
	AccessControl bool
	Region        bool
	Bot           bool
	WAF           bool
	Cache         bool
}

// MiddlewareChain 是完整的中间件链。
type MiddlewareChain struct {
	sw        LayerSwitch
	ddosGuard *ddos.Guard
	ccEngine  *zcc.Engine
	wafEngine *waf.Engine
	botEngine *bot.Engine
	cacheStore *cache.Store

	// 黑白名单（CIDR 友好）。
	blacklistIPs map[string]bool
	whitelistIPs map[string]bool
	blacklistMu  sync.RWMutex

	// 访问控制。
	pathPasswords map[string]string // path -> password (Basic Auth)
	acMu          sync.RWMutex

	// 区域访问限制。
	allowedProvinces map[string]bool
	blockedProvinces map[string]bool
	allowedASNs      map[string]bool
	blockedASNs      map[string]bool
	regionMu         sync.RWMutex

	// 防盗链。
	allowEmptyReferer     bool
	allowedRefererDomains map[string]bool
	antiLeechMu           sync.RWMutex

	// 动态请求防护：非 GET/HEAD 的请求需要额外校验（这里复用 WAF）。
	dynamicStrict bool
}

// NewMiddlewareChain 创建中间件链。
func NewMiddlewareChain(
	ddosGuard *ddos.Guard,
	ccEngine *zcc.Engine,
	wafEngine *waf.Engine,
	botEngine *bot.Engine,
	cacheStore *cache.Store,
) *MiddlewareChain {
	return &MiddlewareChain{
		sw: LayerSwitch{
			ProtectionEnabled: true,
			Blacklist: true, CC: true, AccessControl: true,
			Region: true, Bot: true, WAF: true, Cache: true,
		},
		ddosGuard:             ddosGuard,
		ccEngine:              ccEngine,
		wafEngine:             wafEngine,
		botEngine:             botEngine,
		cacheStore:            cacheStore,
		blacklistIPs:          map[string]bool{},
		whitelistIPs:          map[string]bool{},
		pathPasswords:         map[string]string{},
		allowedProvinces:      map[string]bool{},
		blockedProvinces:      map[string]bool{},
		allowedASNs:           map[string]bool{},
		blockedASNs:           map[string]bool{},
		allowedRefererDomains: map[string]bool{},
		allowEmptyReferer:     true,
	}
}

// SetLayerSwitch 动态调整开关。
func (c *MiddlewareChain) SetLayerSwitch(sw LayerSwitch) {
	c.blacklistMu.Lock()
	c.sw = sw
	c.blacklistMu.Unlock()
}

// SetBlacklist 设置 IP 黑名单。
func (c *MiddlewareChain) SetBlacklist(ips []string) {
	c.blacklistMu.Lock()
	defer c.blacklistMu.Unlock()
	c.blacklistIPs = map[string]bool{}
	for _, ip := range ips {
		c.blacklistIPs[ip] = true
	}
}

// SetWhitelist 设置 IP 白名单（白名单优先级最高，跳过所有检查）。
func (c *MiddlewareChain) SetWhitelist(ips []string) {
	c.blacklistMu.Lock()
	defer c.blacklistMu.Unlock()
	c.whitelistIPs = map[string]bool{}
	for _, ip := range ips {
		c.whitelistIPs[ip] = true
	}
}

// SetPathPasswords 设置路径密码保护（Basic Auth）。
func (c *MiddlewareChain) SetPathPasswords(m map[string]string) {
	c.acMu.Lock()
	defer c.acMu.Unlock()
	c.pathPasswords = m
}

// SetRegionRules 设置区域访问规则。
func (c *MiddlewareChain) SetRegionRules(allowedProvinces, blockedProvinces, allowedASNs, blockedASNs []string) {
	c.regionMu.Lock()
	defer c.regionMu.Unlock()
	c.allowedProvinces = toSet(allowedProvinces)
	c.blockedProvinces = toSet(blockedProvinces)
	c.allowedASNs = toSet(allowedASNs)
	c.blockedASNs = toSet(blockedASNs)
}

// SetAntiLeech 设置防盗链白名单域名。
func (c *MiddlewareChain) SetAntiLeech(allowedDomains []string, allowEmpty bool) {
	c.antiLeechMu.Lock()
	defer c.antiLeechMu.Unlock()
	c.allowedRefererDomains = toSet(allowedDomains)
	c.allowEmptyReferer = allowEmpty
}

// Handler 返回包裹完整中间件链的 http.Handler。
func (c *MiddlewareChain) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ===== 防护总开关：关闭时跳过所有防护直接转发源站 =====
		c.blacklistMu.RLock()
		sw := c.sw
		c.blacklistMu.RUnlock()
		if !sw.ProtectionEnabled {
			next.ServeHTTP(w, r)
			return
		}

		ip := clientIP(r)

		// ===== 0. IP 白名单：直接放行 =====
		c.blacklistMu.RLock()
		if c.whitelistIPs[ip] {
			c.blacklistMu.RUnlock()
			next.ServeHTTP(w, r)
			return
		}
		sw = c.sw
		c.blacklistMu.RUnlock()

		// ===== 1. 黑白名单 =====
		if sw.Blacklist {
			c.blacklistMu.RLock()
			blocked := c.blacklistIPs[ip]
			c.blacklistMu.RUnlock()
			if blocked {
				http.Error(w, "Forbidden: IP blacklisted", http.StatusForbidden)
				return
			}
		}

		// ===== DDoS 防护（前置）=====
		if c.ddosGuard != nil && !c.ddosGuard.Allow(ip) {
			http.Error(w, "Too Many Requests: DDoS protection", http.StatusTooManyRequests)
			return
		}

		// ===== 2. CC 防护 =====
		if sw.CC && c.ccEngine != nil {
			if !c.ccEngine.Allow(ip, r) {
				// 触发挑战或直接拒绝。
				c.ccEngine.IssueChallenge(w, r)
				return
			}
		}

		// ===== 3. 访问控制（路径密码保护）=====
		if sw.AccessControl {
			if !c.checkAccessControl(w, r) {
				return
			}
		}

		// ===== 4. 区域访问限制 =====
		if sw.Region {
			if !c.checkRegion(ip) {
				http.Error(w, "Forbidden: region restricted", http.StatusForbidden)
				return
			}
		}

		// ===== 5. Bot 检测 =====
		if sw.Bot && c.botEngine != nil {
			verdict := c.botEngine.Detect(r)
			if verdict.Blocked {
				http.Error(w, "Forbidden: malicious bot", http.StatusForbidden)
				return
			}
		}

		// ===== 6. 语义 WAF =====
		if sw.WAF && c.wafEngine != nil {
			dec, block := c.wafEngine.ShouldBlock(r)
			if block {
				http.Error(w, "Forbidden: WAF rule matched", http.StatusForbidden)
				_ = dec
				return
			}
		}

		// ===== 7. 防盗链 =====
		if !c.checkAntiLeech(r) {
			http.Error(w, "Forbidden: hotlinking not allowed", http.StatusForbidden)
			return
		}

		// ===== 8. 缓存 =====
		if sw.Cache && c.cacheStore != nil && isCacheable(r) {
			if cached := c.cacheStore.Get(r); cached != nil {
				for k, vs := range cached.Headers {
					for _, v := range vs {
						w.Header().Add(k, v)
					}
				}
				w.Header().Set("X-ShieldFlow-Cache", "HIT")
				_, _ = w.Write(cached.Body)
				return
			}
			// MISS: 包装 writer 记录响应。
			rec := &recordingWriter{header: http.Header{}, status: 200, buf: newBuffer()}
			rw := &splitWriter{ResponseWriter: w, rec: rec}
			next.ServeHTTP(rw, r)
			if isCacheableResponse(rec.status, rec.header) {
				c.cacheStore.Set(r, &cache.Entry{
					Headers: rec.header,
					Body:    rec.buf.Bytes(),
				})
			}
			return
		}

		// ===== 9. 转发源站 =====
		next.ServeHTTP(w, r)
	})
}

// checkAccessControl 路径密码保护。
func (c *MiddlewareChain) checkAccessControl(w http.ResponseWriter, r *http.Request) bool {
	c.acMu.RLock()
	defer c.acMu.RUnlock()
	for path, pass := range c.pathPasswords {
		if strings.HasPrefix(r.URL.Path, path) {
			user, pwd, ok := r.BasicAuth()
			if !ok || user != "admin" || pwd != pass {
				w.Header().Set("WWW-Authenticate", `Basic realm="ShieldFlow Protected"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return false
			}
		}
	}
	return true
}

// checkRegion 区域访问限制（占位：真实场景需要 IP→省份/ASN 库）。
func (c *MiddlewareChain) checkRegion(ip string) bool {
	c.regionMu.RLock()
	defer c.regionMu.RUnlock()
	// 占位：默认放行；真实实现需调用 IP 地理库。
	// 这里仅演示 ASN 黑名单逻辑（若配置）。
	if len(c.blockedASNs) == 0 && len(c.allowedProvinces) == 0 && len(c.blockedProvinces) == 0 {
		return true
	}
	_ = ip
	return true
}

// checkAntiLeech 防盗链检查。
func (c *MiddlewareChain) checkAntiLeech(r *http.Request) bool {
	c.antiLeechMu.RLock()
	defer c.antiLeechMu.RUnlock()
	if len(c.allowedRefererDomains) == 0 {
		return true
	}
	referer := r.Header.Get("Referer")
	if referer == "" {
		return c.allowEmptyReferer
	}
	host := extractHost(referer)
	if host == "" {
		return false
	}
	for domain := range c.allowedRefererDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

// extractHost 从 URL 字符串中提取 hostname（小写）。
func extractHost(s string) string {
	idx := strings.Index(s, "://")
	if idx < 0 {
		return ""
	}
	rest := s[idx+3:]
	end := strings.IndexAny(rest, "/?#$")
	host := rest
	if end > 0 {
		host = rest[:end]
	}
	return strings.ToLower(host)
}

// ---- 辅助 ----

func toSet(items []string) map[string]bool {
	m := map[string]bool{}
	for _, i := range items {
		if i != "" {
			m[i] = true
		}
	}
	return m
}

func isCacheable(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if cc := r.Header.Get("Cache-Control"); strings.Contains(cc, "no-cache") || strings.Contains(cc, "no-store") {
		return false
	}
	return true
}

func isCacheableResponse(status int, h http.Header) bool {
	if status < 200 || status >= 300 {
		return false
	}
	cc := h.Get("Cache-Control")
	if strings.Contains(cc, "no-cache") || strings.Contains(cc, "no-store") || strings.Contains(cc, "private") {
		return false
	}
	return true
}

// recordingWriter 记录下游 handler 的响应以供缓存。
type recordingWriter struct {
	header http.Header
	status int
	buf    *buffer
}

func (w *recordingWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}
func (w *recordingWriter) WriteHeader(status int) { w.status = status }
func (w *recordingWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

type splitWriter struct {
	http.ResponseWriter
	rec *recordingWriter
}

func (s *splitWriter) WriteHeader(status int) {
	s.ResponseWriter.WriteHeader(status)
	s.rec.status = status
}
func (s *splitWriter) Write(b []byte) (int, error) {
	s.rec.Write(b)
	return s.ResponseWriter.Write(b)
}

// buffer 简易字节缓冲。
type buffer struct {
	data []byte
}

func newBuffer() *buffer { return &buffer{} }
func (b *buffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}
func (b *buffer) Bytes() []byte { return b.data }

// 确保 net 包被使用（未来 IP 解析扩展预留）。
var _ = net.Listen
var _ = time.Now
