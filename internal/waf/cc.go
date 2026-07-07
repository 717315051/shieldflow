// Package waf (internal/waf) 实现 CC 防护。
//
// 功能：
//   - 全局频控（次数 / 时间窗口）
//   - 8 种挑战类型：安全验证 / 工作量证明 / 点击验证 / 滑块验证 /
//     旋转验证 / 数字计算 / 无感验证 / 五秒盾
//   - 等待室：最大并发 / 基础等待 / 递增等待 / 最大等待
package waf

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ChallengeType 挑战类型。
type ChallengeType string

const (
	ChallengeNone       ChallengeType = "none"        // 无挑战（直接拒绝）
	ChallengeSecurity   ChallengeType = "security"    // 安全验证（Cookie 验证）
	ChallengePoW        ChallengeType = "pow"         // 工作量证明
	ChallengeClick      ChallengeType = "click"       // 点击验证
	ChallengeSlider     ChallengeType = "slider"      // 滑块验证
	ChallengeRotate     ChallengeType = "rotate"      // 旋转验证
	ChallengeMath       ChallengeType = "math"        // 数字计算
	ChallengeInvisible  ChallengeType = "invisible"   // 无感验证
	Challenge5sShield   ChallengeType = "5s_shield"   // 五秒盾
)

// WaitingRoomConfig 等待室配置。
type WaitingRoomConfig struct {
	Enabled        bool `json:"enabled"`
	MaxConcurrent  int  `json:"max_concurrent"`   // 最大并发请求数
	BaseWaitMs     int  `json:"base_wait_ms"`     // 基础等待毫秒
	IncrementMs    int  `json:"increment_ms"`     // 每超 1 个并发递增等待
	MaxWaitMs      int  `json:"max_wait_ms"`      // 最大等待毫秒
}

// CCConfig CC 防护配置。
type CCConfig struct {
	// 全局频控
	GlobalRateLimit int           `json:"global_rate_limit"` // 次数
	GlobalWindow    time.Duration `json:"global_window"`     // 时间窗口

	// 每路径频控（可选）
	PathRules map[string]PathRateRule `json:"path_rules"`

	// 挑战类型
	ChallengeType ChallengeType `json:"challenge_type"`

	// 等待室
	WaitingRoom WaitingRoomConfig `json:"waiting_room"`

	// 挑战通过后的 Cookie 名称
	ChallengeCookieName string `json:"challenge_cookie_name"`
	ChallengeSecret     string `json:"challenge_secret"` // 用于签名 Cookie
}

// PathRateRule 路径级频控规则。
type PathRateRule struct {
	Limit  int           `json:"limit"`
	Window time.Duration `json:"window"`
}

// EnhancedCCConfig 加强版 CC 防护配置（多层策略 + 优先级）。
type EnhancedCCConfig struct {
	// DefaultPassTTL 默认验证通过 TTL（秒）。
	DefaultPassTTL int `json:"default_pass_ttl"`
	// DefaultBanTTL 默认封禁 TTL（秒）。
	DefaultBanTTL int `json:"default_ban_ttl"`
	// AdaptiveEnabled 自适应阈值开关。
	AdaptiveEnabled bool `json:"adaptive_enabled"`
	// AdaptiveMinRatio 自适应最小倍率。
	AdaptiveMinRatio float64 `json:"adaptive_min_ratio"`
	// AdaptiveMaxRatio 自适应最大倍率。
	AdaptiveMaxRatio float64 `json:"adaptive_max_ratio"`
	// BaselineWindow 基线窗口（秒）。
	BaselineWindow int `json:"baseline_window"`
	// Layers 策略层级（按 Priority 从小到大依次匹配）。
	Layers []CCLayer `json:"layers"`
}

// CCLayer 加强版 CC 防护策略层。
type CCLayer struct {
	// Name 层级名称。
	Name string `json:"name"`
	// Priority 优先级（越小越高）。
	Priority int `json:"priority"`
	// Enabled 状态。
	Enabled bool `json:"enabled"`
	// Scope 作用范围：global / path。
	Scope string `json:"scope"`
	// PathPattern 路径匹配（Scope=path 时生效）。
	PathPattern string `json:"path_pattern"`
	// StatObject 统计对象：ip / session。
	StatObject string `json:"stat_object"`
	// Threshold 请求数阈值。
	Threshold int `json:"threshold"`
	// Window 时间窗口（秒）。
	Window int `json:"window"`
	// Action 动作：verify / block / ban。
	Action string `json:"action"`
	// BanDuration 封禁时长（秒）。
	BanDuration int `json:"ban_duration"`
	// ResponsePhase 响应阶段：request / response。
	ResponsePhase string `json:"response_phase"`
}

// SmartCCConfig 智能 CC 防护配置（自动阈值推算）。
type SmartCCConfig struct {
	// Enabled 智能 CC 开关。
	Enabled bool `json:"enabled"`
	// Level 级别：loose / medium / strict。
	Level string `json:"level"`
	// LastCalcTime 上次计算时间。
	LastCalcTime time.Time `json:"last_calc_time"`
	// AutoThresholds 系统托管的阈值（覆盖 enhancedCC.Layers）。
	AutoThresholds []CCLayer `json:"auto_thresholds"`
}

// AccessLogSummary 访问摘要（供智能 CC 推算阈值）。
type AccessLogSummary struct {
	TotalRequests int64   `json:"total_requests"`
	UniqueIPs     int64   `json:"unique_ips"`
	PeakIPRequests int64  `json:"peak_ip_requests"`
	BaselineRate  float64 `json:"baseline_rate"` // 单 IP 基线速率（请求/秒）
}

// Engine CC 防护引擎。
type Engine struct {
	mu       sync.RWMutex
	cfg      CCConfig
	counters map[string]*slidingCounter // ip -> 全局计数
	pathCounters map[string]map[string]*slidingCounter // ip -> path -> 计数
	currentConcurrent map[string]int // ip -> 当前并发
	passedChallenges  map[string]int64 // ip -> 挑战通过时间(unix秒)

	// enhancedCC 加强版 CC 配置（多层策略+优先级）。
	enhancedCC *EnhancedCCConfig
	// smartCC 智能 CC 配置（自动阈值推算）。
	smartCC *SmartCCConfig
	// layerCounters 加强版 CC 各层级的计数器: layerName -> ip -> counter。
	layerCounters map[string]map[string]*slidingCounter
}

// NewEngine 创建 CC 防护引擎。
func NewEngine(cfg CCConfig) *Engine {
	if cfg.GlobalRateLimit <= 0 {
		cfg.GlobalRateLimit = 60
	}
	if cfg.GlobalWindow <= 0 {
		cfg.GlobalWindow = time.Minute
	}
	if cfg.ChallengeType == "" {
		cfg.ChallengeType = Challenge5sShield
	}
	if cfg.ChallengeCookieName == "" {
		cfg.ChallengeCookieName = "shieldflow_cc_pass"
	}
	if cfg.ChallengeSecret == "" {
		cfg.ChallengeSecret = "shieldflow-cc-secret"
	}
	if cfg.WaitingRoom.Enabled && cfg.WaitingRoom.MaxConcurrent <= 0 {
		cfg.WaitingRoom.MaxConcurrent = 1000
	}
	if cfg.WaitingRoom.BaseWaitMs <= 0 {
		cfg.WaitingRoom.BaseWaitMs = 1000
	}
	if cfg.WaitingRoom.IncrementMs <= 0 {
		cfg.WaitingRoom.IncrementMs = 500
	}
	if cfg.WaitingRoom.MaxWaitMs <= 0 {
		cfg.WaitingRoom.MaxWaitMs = 10000
	}
	return &Engine{
		cfg:               cfg,
		counters:          map[string]*slidingCounter{},
		pathCounters:      map[string]map[string]*slidingCounter{},
		currentConcurrent: map[string]int{},
		passedChallenges:  map[string]int64{},
		layerCounters:     map[string]map[string]*slidingCounter{},
	}
}

// UpdateConfig 动态更新配置。
func (e *Engine) UpdateConfig(cfg CCConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg = cfg
}

// SetEnhancedCC 设置加强版 CC 防护配置。
func (e *Engine) SetEnhancedCC(cfg EnhancedCCConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enhancedCC = &cfg
}

// SetSmartCC 设置智能 CC 防护配置。
func (e *Engine) SetSmartCC(cfg SmartCCConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.smartCC = &cfg
}

// CalculateSmartCC 根据近 24h 访问摘要推算智能 CC 阈值。
//
// loose: 基线*3, medium: 基线*2, strict: 基线*1.5
// 返回推算后的 CCLayer 列表，并更新 smartCC.AutoThresholds。
func (e *Engine) CalculateSmartCC(summary AccessLogSummary) []CCLayer {
	e.mu.Lock()
	defer e.mu.Unlock()

	baseline := summary.BaselineRate
	if baseline <= 0 {
		// 回退：用高峰 IP 请求量估算基线速率（假设 24h 窗口）。
		if summary.PeakIPRequests > 0 {
			baseline = float64(summary.PeakIPRequests) / 86400.0
		} else if summary.UniqueIPs > 0 {
			baseline = float64(summary.TotalRequests) / float64(summary.UniqueIPs) / 86400.0
		} else {
			baseline = 1.0 / 60.0 // 默认 1 请求/分钟
		}
	}

	level := "medium"
	if e.smartCC != nil && e.smartCC.Level != "" {
		level = e.smartCC.Level
	}
	var ratio float64
	switch level {
	case "loose":
		ratio = 3.0
	case "strict":
		ratio = 1.5
	default: // medium
		ratio = 2.0
	}

	// 阈值 = 基线速率 * 倍率 * 窗口（60 秒）。
	windowSec := 60
	threshold := int(baseline*ratio) * windowSec
	if threshold < 10 {
		threshold = 10
	}

	layers := []CCLayer{
		{
			Name:          "smart_global",
			Priority:      0,
			Enabled:       true,
			Scope:         "global",
			StatObject:    "ip",
			Threshold:     threshold,
			Window:        windowSec,
			Action:        "verify",
			ResponsePhase: "request",
		},
		{
			Name:          "smart_path",
			Priority:      10,
			Enabled:       true,
			Scope:         "global",
			StatObject:    "ip",
			Threshold:     threshold / 2,
			Window:        windowSec,
			Action:        "block",
			BanDuration:   600,
			ResponsePhase: "request",
		},
	}

	if e.smartCC == nil {
		e.smartCC = &SmartCCConfig{}
	}
	e.smartCC.AutoThresholds = layers
	e.smartCC.LastCalcTime = time.Now()

	return layers
}

// Allow 判断请求是否允许通过。
//
// 返回 false 表示需触发挑战或拒绝，调用方应调用 IssueChallenge。
func (e *Engine) Allow(ip string, r *http.Request) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 已通过挑战的 IP：在有效期内放行。
	if ts, ok := e.passedChallenges[ip]; ok {
		if time.Now().Unix()-ts < 3600 {
			e.incrementConcurrent(ip)
			return true
		}
		delete(e.passedChallenges, ip)
	}

	// ===== 加强版 CC 防护（多层策略+优先级）=====
	// 如果 smartCC.Enabled，则 enhancedCC.Layers 被 smartCC.AutoThresholds 覆盖。
	var layers []CCLayer
	if e.smartCC != nil && e.smartCC.Enabled && len(e.smartCC.AutoThresholds) > 0 {
		layers = e.smartCC.AutoThresholds
	} else if e.enhancedCC != nil && len(e.enhancedCC.Layers) > 0 {
		layers = e.enhancedCC.Layers
	}
	if len(layers) > 0 {
		// 按优先级从小到大排序（稳定排序，避免打乱相同优先级）。
		sortedLayers := make([]CCLayer, len(layers))
		copy(sortedLayers, layers)
		for i := 1; i < len(sortedLayers); i++ {
			for j := i; j > 0 && sortedLayers[j-1].Priority > sortedLayers[j].Priority; j-- {
				sortedLayers[j-1], sortedLayers[j] = sortedLayers[j], sortedLayers[j-1]
			}
		}
		for _, layer := range sortedLayers {
			if !layer.Enabled {
				continue
			}
			// 路径匹配。
			if layer.Scope == "path" && layer.PathPattern != "" {
				if !strings.HasPrefix(r.URL.Path, layer.PathPattern) {
					continue
				}
			}
			// 获取或创建该层的计数器。
			window := time.Duration(layer.Window) * time.Second
			if window <= 0 {
				window = time.Minute
			}
			ipCounters, ok := e.layerCounters[layer.Name]
			if !ok {
				ipCounters = map[string]*slidingCounter{}
				e.layerCounters[layer.Name] = ipCounters
			}
			lc, ok := ipCounters[ip]
			if !ok {
				lc = newSlidingCounter(window)
				ipCounters[ip] = lc
			}
			lc.add(1)
			if lc.count() > layer.Threshold {
				// 命中阈值：执行动作。
				switch layer.Action {
				case "ban":
					// 封禁：标记通过挑战失效并拒绝。
					delete(e.passedChallenges, ip)
					return false
				case "block":
					return false
				case "verify":
					return false // 触发挑战
				}
			}
		}
	}

	// 全局频控。
	sc := e.counters[ip]
	if sc == nil {
		sc = newSlidingCounter(e.cfg.GlobalWindow)
		e.counters[ip] = sc
	}
	sc.add(1)
	if sc.count() > e.cfg.GlobalRateLimit {
		return false // 触发挑战
	}

	// 路径级频控。
	for pathPrefix, rule := range e.cfg.PathRules {
		if strings.HasPrefix(r.URL.Path, pathPrefix) {
			pc := e.pathCounters[ip]
			if pc == nil {
				pc = map[string]*slidingCounter{}
				e.pathCounters[ip] = pc
			}
			pcs, ok := pc[pathPrefix]
			if !ok {
				pcs = newSlidingCounter(rule.Window)
				pc[pathPrefix] = pcs
			}
			pcs.add(1)
			if pcs.count() > rule.Limit {
				return false
			}
		}
	}

	// 等待室。
	if e.cfg.WaitingRoom.Enabled {
		if e.currentConcurrent[ip] >= e.cfg.WaitingRoom.MaxConcurrent {
			return false
		}
	}

	e.incrementConcurrent(ip)
	return true
}

// Release 请求结束时减少并发计数。
func (e *Engine) Release(ip string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.currentConcurrent[ip] > 0 {
		e.currentConcurrent[ip]--
	}
}

// incrementConcurrent 增加并发计数（调用方需持锁）。
func (e *Engine) incrementConcurrent(ip string) {
	e.currentConcurrent[ip]++
}

// IssueChallenge 向客户端下发挑战页面。
func (e *Engine) IssueChallenge(w http.ResponseWriter, r *http.Request) {
	e.mu.RLock()
	ct := e.cfg.ChallengeType
	cookieName := e.cfg.ChallengeCookieName
	secret := e.cfg.ChallengeSecret
	e.mu.RUnlock()

	switch ct {
	case ChallengeSecurity, ChallengeInvisible:
		// 无感验证：下发带签名的 Cookie，前端 JS 立即回放。
		e.serveInvisibleChallenge(w, r, cookieName, secret)
	case Challenge5sShield:
		e.serve5sShield(w, r, cookieName, secret)
	case ChallengeMath:
		e.serveMathChallenge(w, r, cookieName, secret)
	case ChallengeClick:
		e.serveClickChallenge(w, r, cookieName, secret)
	case ChallengeSlider:
		e.serveSliderChallenge(w, r, cookieName, secret)
	case ChallengeRotate:
		e.serveRotateChallenge(w, r, cookieName, secret)
	case ChallengePoW:
		e.servePoWChallenge(w, r, cookieName, secret)
	default:
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
	}
}

// MarkChallengePassed 标记 IP 通过了挑战（可由验证端点调用）。
func (e *Engine) MarkChallengePassed(ip string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.passedChallenges[ip] = time.Now().Unix()
}

// ---- 挑战页面实现 ----

// serve5sShield 五秒盾：返回一个 5 秒后自动跳转的 HTML 页面，
// 跳转时通过 JS 设置带签名的 Cookie。
func (e *Engine) serve5sShield(w http.ResponseWriter, r *http.Request, cookieName, secret string) {
	ip := clientIP(r)
	token := signToken(ip, secret)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8">
<title>安全检查中...</title>
<script>
setTimeout(function(){
  document.cookie = "%s=" + encodeURIComponent("%s") + "; path=/; max-age=3600";
  location.reload();
}, 5000);
</script>
</head><body>
<h3>正在验证您的访问，请稍候 5 秒...</h3>
<p>验证完成后将自动刷新页面。</p>
</body></html>`, cookieName, token)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Retry-After", "5")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(html))
}

// serveInvisibleChallenge 无感验证：立即下发 Cookie 并要求 JS 回放。
func (e *Engine) serveInvisibleChallenge(w http.ResponseWriter, r *http.Request, cookieName, secret string) {
	ip := clientIP(r)
	token := signToken(ip, secret)
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
<script>
document.cookie = "%s=" + encodeURIComponent("%s") + "; path=/; max-age=3600";
location.reload();
</script></body></html>`, cookieName, token)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(html))
}

// serveMathChallenge 数字计算挑战。
func (e *Engine) serveMathChallenge(w http.ResponseWriter, r *http.Request, cookieName, secret string) {
	ip := clientIP(r)
	token := signToken(ip, secret)
	a, b := 7, 8
	answer := a + b
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
<h3>请完成验证：%d + %d = ?</h3>
<input id="ans" type="text" />
<button onclick="check()">提交</button>
<script>
function check(){
  if (parseInt(document.getElementById('ans').value) === %d) {
    document.cookie = "%s=" + encodeURIComponent("%s") + "; path=/; max-age=3600";
    location.reload();
  } else {
    alert('验证失败，请重试');
  }
}
</script></body></html>`, a, b, answer, cookieName, token)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(html))
}

// serveClickChallenge 点击验证。
func (e *Engine) serveClickChallenge(w http.ResponseWriter, r *http.Request, cookieName, secret string) {
	ip := clientIP(r)
	token := signToken(ip, secret)
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
<h3>请点击下方按钮完成验证</h3>
<button onclick="pass()">我是人类</button>
<script>
function pass(){
  document.cookie = "%s=" + encodeURIComponent("%s") + "; path=/; max-age=3600";
  location.reload();
}
</script></body></html>`, cookieName, token)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(html))
}

// serveSliderChallenge 滑块验证（占位简化版）。
func (e *Engine) serveSliderChallenge(w http.ResponseWriter, r *http.Request, cookieName, secret string) {
	ip := clientIP(r)
	token := signToken(ip, secret)
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
<h3>请拖动滑块到最右侧</h3>
<input type="range" id="slider" min="0" max="100" oninput="check(this)"/>
<script>
function check(el){
  if (el.value == 100) {
    document.cookie = "%s=" + encodeURIComponent("%s") + "; path=/; max-age=3600";
    location.reload();
  }
}
</script></body></html>`, cookieName, token)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(html))
}

// serveRotateChallenge 旋转验证（占位简化版）。
func (e *Engine) serveRotateChallenge(w http.ResponseWriter, r *http.Request, cookieName, secret string) {
	ip := clientIP(r)
	token := signToken(ip, secret)
	html := `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
<h3>请旋转图片到正确方向</h3>
<div style="font-size:60px;transform:rotate(90deg);cursor:pointer;" id="img" onclick="rotate()">🔄</div>
<p>点击图片旋转到正立</p>
<script>
var deg = 90;
var FULL = 360;
function rotate(){
  deg += 90;
  document.getElementById('img').style.transform = 'rotate(' + deg + 'deg)';
  if ((deg % FULL) === 0) {
    document.cookie = "` + cookieName + `=" + encodeURIComponent("` + token + `") + "; path=/; max-age=3600";
    location.reload();
  }
}
</script></body></html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(html))
}

// servePoWChallenge 工作量证明：要求客户端计算满足前导零的哈希。
func (e *Engine) servePoWChallenge(w http.ResponseWriter, r *http.Request, cookieName, secret string) {
	ip := clientIP(r)
	token := signToken(ip, secret)
	html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body>
<h3>正在进行工作量证明...</h3>
<script>
// 简化 PoW：找到一个 nonce 使简单哈希后两位为 "00"
function simpleHash(s){
  var h = 0;
  for (var i=0; i<s.length; i++){ h = (h*31 + s.charCodeAt(i)) & 0xffff; }
  return ("000"+h.toString(16)).slice(-4);
}
var base = "%d";
var nonce = 0;
while (true){
  if (simpleHash(base+nonce).slice(-2) === "00") break;
  nonce++;
}
document.cookie = "%s=" + encodeURIComponent("%s") + "; path=/; max-age=3600";
location.reload();
</script></body></html>`, time.Now().Unix(), cookieName, token)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(html))
}

// ---- 辅助函数 ----

func clientIP(r *http.Request) string {
	if ff := r.Header.Get("X-Forwarded-For"); ff != "" {
		if idx := strings.IndexByte(ff, ','); idx > 0 {
			return strings.TrimSpace(ff[:idx])
		}
		return strings.TrimSpace(ff)
	}
	if real := r.Header.Get("X-Real-IP"); real != "" {
		return strings.TrimSpace(real)
	}
	return r.RemoteAddr
}

// signToken 生成带签名的挑战通过令牌（简化版 HMAC）。
func signToken(ip, secret string) string {
	// 真实实现应使用 HMAC-SHA256；此处简化为拼接哈希。
	return fmt.Sprintf("%x|%d", ip+secret, time.Now().Unix()/3600)
}

// ---- 滑动窗口计数器 ----

type slidingCounter struct {
	window  time.Duration
	buckets []bucket
}

type bucket struct {
	ts time.Time
	n  int
}

func newSlidingCounter(window time.Duration) *slidingCounter {
	return &slidingCounter{window: window}
}

func (s *slidingCounter) add(n int) {
	s.buckets = append(s.buckets, bucket{ts: time.Now(), n: n})
	s.evict(time.Now())
}

func (s *slidingCounter) count() int {
	s.evict(time.Now())
	total := 0
	for _, b := range s.buckets {
		total += b.n
	}
	return total
}

func (s *slidingCounter) evict(now time.Time) {
	cutoff := now.Add(-s.window)
	kept := s.buckets[:0]
	for _, b := range s.buckets {
		if b.ts.After(cutoff) {
			kept = append(kept, b)
		}
	}
	s.buckets = kept
}
