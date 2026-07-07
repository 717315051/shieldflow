// Package waf implements the semantic Web Application Firewall engine.
//
// 检测范围: URL参数 / 路径 / Headers / 请求体 / Cookies
// 威胁类型: SQLi / XSS / RCE / XXE / SSRF / LDAP / NoSQL / 路径遍历 /
//           CRLF / HTTP走私 / SSTI / 扫描器 / 协议违规 / PHP注入 / Java攻击 /
//           命令注入 / 开放重定向
// 威胁阈值: 0-100
package waf

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// ThreatKind 描述命中的威胁大类。
type ThreatKind string

const (
	ThreatSQLi           ThreatKind = "sqli"
	ThreatXSS            ThreatKind = "xss"
	ThreatRCE            ThreatKind = "rce"
	ThreatXXE            ThreatKind = "xxe"
	ThreatSSRF           ThreatKind = "ssrf"
	ThreatLDAP           ThreatKind = "ldap"
	ThreatNoSQL          ThreatKind = "nosql"
	ThreatPathTraversal  ThreatKind = "path_traversal"
	ThreatCRLF           ThreatKind = "crlf"
	ThreatHTTPSmuggling  ThreatKind = "http_smuggling"
	ThreatSSTI           ThreatKind = "ssti"
	ThreatScanner        ThreatKind = "scanner"
	ThreatProtocolViol   ThreatKind = "protocol_violation"
	ThreatPHPInjection   ThreatKind = "php_injection"
	ThreatJavaAttack     ThreatKind = "java_attack"
	ThreatCmdInjection   ThreatKind = "cmd_injection"
	ThreatOpenRedirect   ThreatKind = "open_redirect"
)

// Mode 控制 WAF 的工作模式：拦截(block) 或 观察(detect/observe)。
type Mode string

const (
	ModeBlock  Mode = "block"   // 拦截模式：命中即拒绝
	ModeDetect Mode = "detect"  // 观察模式：仅记录不拦截
)

// Rule 表示一条 WAF 检测规则。
type Rule struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Kind     ThreatKind `json:"kind"`
	Pattern  string     `json:"pattern"`
	regex    *regexp.Regexp
	Score    int        `json:"score"`     // 命中后累加的分值 0-100
	Scopes   []Scope    `json:"scopes"`    // 适用的检测范围；空表示全部
}

// Scope 描述规则适用的请求位置。
type Scope string

const (
	ScopeURLParam Scope = "url_param"
	ScopePath     Scope = "path"
	ScopeHeader   Scope = "header"
	ScopeBody     Scope = "body"
	ScopeCookie   Scope = "cookie"
)

// Threat 描述一次命中。
type Threat struct {
	RuleID   string     `json:"rule_id"`
	Kind     ThreatKind `json:"kind"`
	Score    int        `json:"score"`
	Scope    Scope      `json:"scope"`
	Match    string     `json:"match"`     // 命中的数据片段
	Field    string     `json:"field"`     // 命中的字段名
}

// Decision 是 WAF 对单个请求的判决结果。
type Decision struct {
	Blocked     bool     `json:"blocked"`
	TotalScore  int      `json:"total_score"`
	Threats     []Threat `json:"threats"`
}

// Whitelist 白名单：IP / URL 前缀 / 规则ID。
type Whitelist struct {
	IPs      []string `json:"ips"`        // 精确 IP 或 CIDR
	URLs     []string `json:"urls"`       // URL 前缀
	RuleIDs  []string `json:"rule_ids"`   // 规则ID
}

// Engine 是 WAF 语义分析引擎。
type Engine struct {
	mu         sync.RWMutex
	rules      []Rule
	threshold  int        // 威胁阈值 0-100，总分>=阈值即判为威胁
	mode       Mode
	whitelist  Whitelist
	bodyLimit  int64      // 请求体最大读取字节数，防止 OOM
}

// NewEngine 创建 WAF 引擎。
func NewEngine(mode Mode, threshold int) *Engine {
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 100 {
		threshold = 100
	}
	e := &Engine{
		threshold: threshold,
		mode:      mode,
		bodyLimit: 1 << 20, // 1MB
	}
	e.rules = ManagedRules()
	e.compile()
	return e
}

// compile 预编译所有规则正则。
func (e *Engine) compile() {
	for i := range e.rules {
		if e.rules[i].Pattern == "" {
			continue
		}
		if re, err := regexp.Compile("(?i)" + e.rules[i].Pattern); err == nil {
			e.rules[i].regex = re
		}
	}
}

// SetWhitelist 设置白名单。
func (e *Engine) SetWhitelist(w Whitelist) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.whitelist = w
}

// SetMode 动态切换工作模式。
func (e *Engine) SetMode(m Mode) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.mode = m
}

// SetThreshold 动态调整威胁阈值。
func (e *Engine) SetThreshold(t int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if t < 0 {
		t = 0
	}
	if t > 100 {
		t = 100
	}
	e.threshold = t
}

// isIPWhitelisted 判断 IP 是否在白名单中（支持精确匹配与 CIDR）。
func (e *Engine) isIPWhitelisted(ip string) bool {
	for _, entry := range e.whitelist.IPs {
		if entry == ip {
			return true
		}
		if strings.Contains(entry, "/") && cidrContains(entry, ip) {
			return true
		}
	}
	return false
}

// isURLWhitelisted 判断 URL 是否命中白名单前缀。
func (e *Engine) isURLWhitelisted(url string) bool {
	for _, prefix := range e.whitelist.URLs {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

// isRuleWhitelisted 判断规则ID是否在白名单。
func (e *Engine) isRuleWhitelisted(id string) bool {
	for _, rid := range e.whitelist.RuleIDs {
		if rid == id {
			return true
		}
	}
	return false
}

// Check 对请求执行 WAF 检测，返回判决结果。
// 注意：该方法会消耗请求体（最多 bodyLimit 字节），调用方应在转发前保存原始 body。
func (e *Engine) Check(r *http.Request) *Decision {
	e.mu.RLock()
	defer e.mu.RUnlock()

	dec := &Decision{}

	clientIP := clientIPFromRequest(r)
	if e.isIPWhitelisted(clientIP) {
		return dec // IP 白名单：直接放行
	}
	if e.isURLWhitelisted(r.URL.RequestURI()) {
		return dec // URL 白名单：直接放行
	}

	// 收集各检测范围的数据。
	scoped := map[Scope]map[string]string{
		ScopeURLParam: collectURLParams(r),
		ScopePath:     {"path": r.URL.Path},
		ScopeHeader:   collectHeaders(r),
		ScopeCookie:   collectCookies(r),
		ScopeBody:     collectBody(r, e.bodyLimit),
	}

	for _, rule := range e.rules {
		if rule.regex == nil {
			continue
		}
		if e.isRuleWhitelisted(rule.ID) {
			continue
		}
		scopes := rule.Scopes
		if len(scopes) == 0 {
			scopes = []Scope{ScopeURLParam, ScopePath, ScopeHeader, ScopeBody, ScopeCookie}
		}
		for _, sc := range scopes {
			fields, ok := scoped[sc]
			if !ok {
				continue
			}
			for field, val := range fields {
				if loc := rule.regex.FindStringIndex(val); loc != nil {
					match := val[loc[0]:loc[1]]
					if len(match) > 128 {
						match = match[:128]
					}
					dec.Threats = append(dec.Threats, Threat{
						RuleID: rule.ID,
						Kind:   rule.Kind,
						Score:  rule.Score,
						Scope:  sc,
						Match:  match,
						Field:  field,
					})
					dec.TotalScore += rule.Score
				}
			}
		}
	}

	if dec.TotalScore >= e.threshold {
		dec.Blocked = true
	}
	// 观察模式永远不拦截，只记录。
	if e.mode == ModeDetect {
		dec.Blocked = false
	}
	return dec
}

// ShouldBlock 返回是否应当拦截该请求。
func (e *Engine) ShouldBlock(r *http.Request) (*Decision, bool) {
	dec := e.Check(r)
	return dec, dec.Blocked
}

// ---- 辅助函数 ----

func collectURLParams(r *http.Request) map[string]string {
	m := map[string]string{}
	for k, vs := range r.URL.Query() {
		for _, v := range vs {
			m[k] = v
		}
	}
	m["__query__"] = r.URL.RawQuery
	return m
}

func collectHeaders(r *http.Request) map[string]string {
	m := map[string]string{}
	for k, vs := range r.Header {
		for _, v := range vs {
			m[k] = v
		}
	}
	return m
}

func collectCookies(r *http.Request) map[string]string {
	m := map[string]string{}
	for _, c := range r.Cookies() {
		m[c.Name] = c.Value
	}
	return m
}

func collectBody(r *http.Request, limit int64) map[string]string {
	m := map[string]string{}
	if r.Body == nil {
		return m
	}
	var buf bytes.Buffer
	reader := bufio.NewReader(io.LimitReader(r.Body, limit))
	if _, err := buf.ReadFrom(reader); err != nil && err != io.EOF {
		return m
	}
	// 恢复 body 供后续转发使用。
	r.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
	r.ContentLength = int64(buf.Len())
	m["__body__"] = buf.String()
	return m
}

func clientIPFromRequest(r *http.Request) string {
	if ff := r.Header.Get("X-Forwarded-For"); ff != "" {
		if idx := strings.Index(ff, ","); idx > 0 {
			return strings.TrimSpace(ff[:idx])
		}
		return strings.TrimSpace(ff)
	}
	if real := r.Header.Get("X-Real-IP"); real != "" {
		return strings.TrimSpace(real)
	}
	// 去除端口。
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx > 0 {
		return addr[:idx]
	}
	return addr
}

// cidrContains 简单 CIDR 包含判断（仅 IPv4）。
func cidrContains(cidr, ip string) bool {
	if !strings.Contains(cidr, "/") {
		return cidr == ip
	}
	parts := strings.SplitN(cidr, "/", 2)
	network := parts[0]
	bits := 0
	for _, ch := range parts[1] {
		bits = bits*10 + int(ch-'0')
	}
	mask := uint32(0xFFFFFFFF << (32 - bits))
	return (ipv4ToUint(network) & mask) == (ipv4ToUint(ip) & mask)
}

func ipv4ToUint(s string) uint32 {
	var a, b, c, d uint32
	_, _ = fmtSscanf(s, &a, &b, &c, &d)
	return (a << 24) | (b << 16) | (c << 8) | d
}

// fmtSscanf 避免引入额外包的简易解析。
func fmtSscanf(s string, a, b, c, d *uint32) (int, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return 0, nil
	}
	*a = atoiSafe(parts[0])
	*b = atoiSafe(parts[1])
	*c = atoiSafe(parts[2])
	*d = atoiSafe(parts[3])
	return 4, nil
}

func atoiSafe(s string) uint32 {
	var n uint32
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + uint32(ch-'0')
	}
	return n
}
