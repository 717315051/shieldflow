// Package waf: 托管规则集。
//
// 提供开箱即用的规则集：SQL注入 / XSS / 命令注入 / 路径遍历 / 扫描器检测 /
// XXE / SSRF / SSTI / CRLF / 开放重定向 / PHP注入 / Java攻击 / 协议违规 等。
// 规则可在拦截模式(block)与观察模式(detect)之间切换。
package waf

// ManagedRules 返回内置托管规则集。规则ID使用前缀分类：
//   SQLI-*, XSI-*, CMD-*, LFI-*, SCN-*, XXE-*, SSRF-*, SSTI-*, CRLF-*,
//   REDIR-*, PHP-*, JAVA-*, PROTO-*, NOSQL-*, LDAP-*。
func ManagedRules() []Rule {
	return []Rule{
		// ===== SQL 注入 =====
		{
			ID: "SQLI-001", Name: "Union-based SQLi", Kind: ThreatSQLi,
			Pattern: `union\s+select`, Score: 80,
			Scopes: []Scope{ScopeURLParam, ScopeBody, ScopeHeader, ScopeCookie},
		},
		{
			ID: "SQLI-002", Name: "Boolean-based SQLi", Kind: ThreatSQLi,
			Pattern: `(or|and)\s+1=1`, Score: 70,
			Scopes: []Scope{ScopeURLParam, ScopeBody},
		},
		{
			ID: "SQLI-003", Name: "SQL comment sequence", Kind: ThreatSQLi,
			Pattern: `(--|#|/\*)`, Score: 30,
			Scopes: []Scope{ScopeURLParam, ScopeBody, ScopeCookie},
		},
		{
			ID: "SQLI-004", Name: "Time-based SQLi", Kind: ThreatSQLi,
			Pattern: `sleep\s*\(|benchmark\s*\(|pg_sleep\s*\(`, Score: 75,
			Scopes: []Scope{ScopeURLParam, ScopeBody},
		},
		{
			ID: "SQLI-005", Name: "Information schema access", Kind: ThreatSQLi,
			Pattern: `information_schema\.`, Score: 85,
		},
		{
			ID: "SQLI-006", Name: "Load_file / outfile", Kind: ThreatSQLi,
			Pattern: `load_file\s*\(|into\s+outfile`, Score: 90,
		},

		// ===== XSS =====
		{
			ID: "XSI-001", Name: "Script tag injection", Kind: ThreatXSS,
			Pattern: `<script`, Score: 85,
			Scopes: []Scope{ScopeURLParam, ScopeBody, ScopeHeader},
		},
		{
			ID: "XSI-002", Name: "JavaScript URI", Kind: ThreatXSS,
			Pattern: `javascript:`, Score: 70,
			Scopes: []Scope{ScopeURLParam, ScopeBody, ScopeHeader, ScopeCookie},
		},
		{
			ID: "XSI-003", Name: "Event handler injection", Kind: ThreatXSS,
			Pattern: `on(error|load|click|mouseover|focus|submit)\s*=`, Score: 75,
		},
		{
			ID: "XSI-004", Name: "Img/VBS/SVG tag injection", Kind: ThreatXSS,
			Pattern: `<(img|vbscript|svg|iframe|object|embed)[^>]*src`, Score: 70,
		},
		{
			ID: "XSI-005", Name: "Encoded script tag", Kind: ThreatXSS,
			Pattern: `(&#x?0*(73|106|60);|%3c%73cript)`, Score: 65,
		},

		// ===== 命令注入 / RCE =====
		{
			ID: "CMD-001", Name: "Shell metacharacters", Kind: ThreatCmdInjection,
			Pattern: `[;&|` + "`" + `$]`, Score: 40,
			Scopes: []Scope{ScopeURLParam, ScopeBody},
		},
		{
			ID: "CMD-002", Name: "Command execution functions", Kind: ThreatRCE,
			Pattern: `(system|exec|popen|passthru|shell_exec|eval)\s*\(`, Score: 90,
		},
		{
			ID: "CMD-003", Name: "Reverse shell patterns", Kind: ThreatRCE,
			Pattern: `(bash -i|nc -e|/bin/sh|/bin/bash)`, Score: 95,
		},

		// ===== 路径遍历 =====
		{
			ID: "LFI-001", Name: "Directory traversal", Kind: ThreatPathTraversal,
			Pattern: `(\.\./|\.\.\\|%2e%2e%2f|%2e%2e/|%2e%2e%5c)`, Score: 80,
			Scopes: []Scope{ScopeURLParam, ScopePath, ScopeBody},
		},
		{
			ID: "LFI-002", Name: "Absolute path access", Kind: ThreatPathTraversal,
			Pattern: `(/etc/passwd|/etc/shadow|/proc/self|c:\\windows\\system32)`, Score: 90,
		},
		{
			ID: "LFI-003", Name: "PHP wrapper abuse", Kind: ThreatPathTraversal,
			Pattern: `(php://filter|php://input|expect://|data://)`, Score: 85,
		},

		// ===== 扫描器检测 =====
		{
			ID: "SCN-001", Name: "Scanner User-Agent", Kind: ThreatScanner,
			Pattern: `(nikto|sqlmap|nmap|masscan|acunetix|nessus|w3af|dirbuster|gobuster|hydra)`, Score: 60,
			Scopes: []Scope{ScopeHeader},
		},
		{
			ID: "SCN-002", Name: "Common scan paths", Kind: ThreatScanner,
			Pattern: `(/\.*\.git|/\.env|/\.svn|/wp-admin|/phpmyadmin|/admin\.php)`, Score: 50,
			Scopes: []Scope{ScopePath},
		},

		// ===== XXE =====
		{
			ID: "XXE-001", Name: "XML entity injection", Kind: ThreatXXE,
			Pattern: `<!entity`, Score: 90,
			Scopes: []Scope{ScopeBody, ScopeHeader},
		},
		{
			ID: "XXE-002", Name: "SYSTEM keyword in XML", Kind: ThreatXXE,
			Pattern: `system\s+["']`, Score: 85,
			Scopes: []Scope{ScopeBody},
		},

		// ===== SSRF =====
		{
			ID: "SSRF-001", Name: "Internal IP access", Kind: ThreatSSRF,
			Pattern: `(127\.0\.0\.1|localhost|0\.0\.0\.0|10\.|172\.(1[6-9]|2[0-9]|3[01])\.|192\.168\.)`, Score: 60,
			Scopes: []Scope{ScopeURLParam, ScopeBody, ScopeHeader},
		},
		{
			ID: "SSRF-002", Name: "Cloud metadata endpoint", Kind: ThreatSSRF,
			Pattern: `169\.254\.169\.254`, Score: 95,
		},

		// ===== SSTI =====
		{
			ID: "SSTI-001", Name: "Template injection Jinja/Twig", Kind: ThreatSSTI,
			Pattern: `\{\{.*\}\}|\{%.*%\}`, Score: 80,
			Scopes: []Scope{ScopeURLParam, ScopeBody},
		},
		{
			ID: "SSTI-002", Name: "Freemarker/Velocity", Kind: ThreatSSTI,
			Pattern: `\$\{.*\}`, Score: 50,
		},

		// ===== CRLF / HTTP 走私 =====
		{
			ID: "CRLF-001", Name: "CRLF injection", Kind: ThreatCRLF,
			Pattern: `(\r\n|%0d%0a|%0a)`, Score: 60,
			Scopes: []Scope{ScopeURLParam, ScopeHeader, ScopeBody},
		},
		{
			ID: "CRLF-002", Name: "Transfer-Encoding smuggling", Kind: ThreatHTTPSmuggling,
			Pattern: `transfer-encoding\s*:`, Score: 70,
			Scopes: []Scope{ScopeHeader},
		},

		// ===== 开放重定向 =====
		{
			ID: "REDIR-001", Name: "Open redirect", Kind: ThreatOpenRedirect,
			Pattern: `(redirect=|url=|next=|return=|goto=)`, Score: 45,
			Scopes: []Scope{ScopeURLParam, ScopeBody},
		},

		// ===== PHP 注入 =====
		{
			ID: "PHP-001", Name: "PHP tags", Kind: ThreatPHPInjection,
			Pattern: `<\?php|<\?=`, Score: 85,
			Scopes: []Scope{ScopeURLParam, ScopeBody},
		},
		{
			ID: "PHP-002", Name: "PHP superglobals", Kind: ThreatPHPInjection,
			Pattern: `\$_(get|post|request|cookie|server|files)\s*\[`, Score: 70,
		},

		// ===== Java 攻击 =====
		{
			ID: "JAVA-001", Name: "JNDI injection", Kind: ThreatJavaAttack,
			Pattern: `\$\{jndi:`, Score: 95,
		},
		{
			ID: "JAVA-002", Name: "Log4Shell style", Kind: ThreatJavaAttack,
			Pattern: `\$\{.*:.*\}`, Score: 75,
		},
		{
			ID: "JAVA-003", Name: "Spring SpEL injection", Kind: ThreatJavaAttack,
			Pattern: `#\(this\)|T\(java\.lang\.Runtime\)`, Score: 90,
		},

		// ===== NoSQL 注入 =====
		{
			ID: "NOSQL-001", Name: "MongoDB operator injection", Kind: ThreatNoSQL,
			Pattern: `(\$where|\$ne|\$gt|\$lt|\$regex)\s*:`, Score: 75,
			Scopes: []Scope{ScopeURLParam, ScopeBody, ScopeCookie},
		},

		// ===== LDAP 注入 =====
		{
			ID: "LDAP-001", Name: "LDAP injection", Kind: ThreatLDAP,
			Pattern: `\*\)|\(\|`, Score: 60,
			Scopes: []Scope{ScopeURLParam, ScopeBody},
		},

		// ===== 协议违规 =====
		{
			ID: "PROTO-001", Name: "Null byte injection", Kind: ThreatProtocolViol,
			Pattern: `%00|\x00`, Score: 70,
		},
		{
			ID: "PROTO-002", Name: "Oversized method", Kind: ThreatProtocolViol,
			Pattern: `^(put|delete|connect|trace|options)$`, Score: 20,
			Scopes: []Scope{ScopePath},
		},
	}
}

// RuleSetByID 返回指定 ID 的规则（便于调试/展示）。
func RuleSetByID(id string) *Rule {
	for _, r := range ManagedRules() {
		if r.ID == id {
			return &r
		}
	}
	return nil
}

// ManagedRuleCount 返回托管规则总数。
func ManagedRuleCount() int {
	return len(ManagedRules())
}
