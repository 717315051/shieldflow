// Package bot 实现爬虫/Bot 检测与分类。
//
// 识别维度：
//   - User-Agent 识别
//   - 搜索引擎爬虫（Googlebot/Bingbot/Baiduspider/360Spider/Sogou 等）→ 放行
//   - 恶意扫描器（sqlmap/nikto/nmap/masscan/acunetix 等）→ 拦截
//   - 数据采集器（scrapy/curl/python-requests/wget 等）→ 标记/限速
//   - 行为特征（无 UA / 异常 UA）→ 标记可疑
package bot

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// BotKind Bot 分类。
type BotKind string

const (
	KindSearchEngine BotKind = "search_engine" // 合法搜索引擎
	KindScanner      BotKind = "scanner"       // 恶意扫描器
	KindScraper      BotKind = "scraper"       // 数据采集器
	KindUnknown      BotKind = "unknown"       // 未知/可疑
	KindHuman        BotKind = "human"         // 真实用户
)

// Verdict Bot 检测判决。
type Verdict struct {
	Kind     BotKind `json:"kind"`
	Name     string  `json:"name"`       // 识别出的 bot 名称
	Blocked  bool    `json:"blocked"`    // 是否应当拦截
	Score    int     `json:"score"`      // 恶意分值 0-100
	Reason   string  `json:"reason"`
}

// Engine Bot 检测引擎。
type Engine struct {
	mu              sync.RWMutex
	searchEngines   []*botPattern
	scanners        []*botPattern
	scrapers        []*botPattern
	allowSearchEngines bool
	blockScanners     bool
	blockScrapers     bool
	blockNoUA         bool
}

type botPattern struct {
	name    string
	pattern *regexp.Regexp
}

// NewEngine 创建 Bot 检测引擎，加载默认规则。
func NewEngine() *Engine {
	e := &Engine{
		allowSearchEngines: true,
		blockScanners:      true,
		blockScrapers:      false,
		blockNoUA:          true,
	}
	e.searchEngines = compilePatterns(searchEngineSignatures)
	e.scanners = compilePatterns(scannerSignatures)
	e.scrapers = compilePatterns(scraperSignatures)
	return e
}

// SetPolicy 配置拦截策略。
func (e *Engine) SetPolicy(allowSearchEngines, blockScanners, blockScrapers, blockNoUA bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.allowSearchEngines = allowSearchEngines
	e.blockScanners = blockScanners
	e.blockScrapers = blockScrapers
	e.blockNoUA = blockNoUA
}

// Detect 对请求执行 Bot 检测。
func (e *Engine) Detect(r *http.Request) Verdict {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ua := r.UserAgent()

	// 无 UA / 空 UA
	if strings.TrimSpace(ua) == "" {
		return Verdict{
			Kind:    KindUnknown,
			Name:    "no-ua",
			Blocked: e.blockNoUA,
			Score:   40,
			Reason:  "empty User-Agent",
		}
	}

	uaLower := strings.ToLower(ua)

	// 1. 搜索引擎爬虫
	for _, p := range e.searchEngines {
		if p.pattern.MatchString(uaLower) {
			return Verdict{
				Kind:    KindSearchEngine,
				Name:    p.name,
				Blocked: !e.allowSearchEngines,
				Score:   0,
				Reason:  "verified search engine crawler",
			}
		}
	}

	// 2. 恶意扫描器
	for _, p := range e.scanners {
		if p.pattern.MatchString(uaLower) {
			return Verdict{
				Kind:    KindScanner,
				Name:    p.name,
				Blocked: e.blockScanners,
				Score:   90,
				Reason:  "known malicious scanner",
			}
		}
	}

	// 3. 数据采集器
	for _, p := range e.scrapers {
		if p.pattern.MatchString(uaLower) {
			return Verdict{
				Kind:    KindScraper,
				Name:    p.name,
				Blocked: e.blockScrapers,
				Score:   50,
				Reason:  "automated scraper / library",
			}
		}
	}

	// 4. 启发式：异常 UA 长度或特殊字符
	if len(ua) > 500 {
		return Verdict{
			Kind:    KindUnknown,
			Name:    "oversized-ua",
			Blocked: false,
			Score:   30,
			Reason:  "abnormally long User-Agent",
		}
	}

	return Verdict{Kind: KindHuman, Name: "human", Score: 0, Reason: "normal user-agent"}
}

// IsSearchEngine 判断是否为搜索引擎爬虫（供外部使用）。
func (e *Engine) IsSearchEngine(ua string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	uaLower := strings.ToLower(ua)
	for _, p := range e.searchEngines {
		if p.pattern.MatchString(uaLower) {
			return true
		}
	}
	return false
}

// ---- 签名表 ----

func compilePatterns(signatures map[string]string) []*botPattern {
	out := make([]*botPattern, 0, len(signatures))
	for name, pat := range signatures {
		re, err := regexp.Compile(pat)
		if err != nil {
			continue
		}
		out = append(out, &botPattern{name: name, pattern: re})
	}
	return out
}

// searchEngineSignatures 合法搜索引擎爬虫 UA 关键字。
var searchEngineSignatures = map[string]string{
	"googlebot":   `googlebot`,
	"bingbot":     `bingbot`,
	"baiduspider": `baiduspider`,
	"yandexbot":   `yandexbot`,
	"sogou":       `sogou web spider|sogou inst spider|sogou spider`,
	"360spider":   `360spider`,
	"bytespider":  `bytespider`,
	"applebot":    `applebot`,
	"duckduckbot": `duckduckbot`,
	"slurp":       `slurp`, // Yahoo
	"facebookbot": `facebookexternalhit`,
	"twitterbot":  `twitterbot`,
	"linkedinbot": `linkedinbot`,
	"telegrambot": `telegrambot`,
	"discordbot":  `discordbot`,
}

// scannerSignatures 恶意扫描器 UA 关键字。
var scannerSignatures = map[string]string{
	"sqlmap":      `sqlmap`,
	"nikto":       `nikto`,
	"nmap":        `nmap`,
	"masscan":     `masscan`,
	"acunetix":    `acunetix`,
	"nessus":      `nessus`,
	"w3af":        `w3af`,
	"dirbuster":   `dirbuster`,
	"gobuster":    `gobuster`,
	"hydra":       `hydra`,
	"metasploit":  `metasploit`,
	"zgrab":       `zgrab`,
	"nuclei":      `nuclei`,
	"wfuzz":       `wfuzz`,
	"ffuf":        `ffuf`,
	"arachni":     `arachni`,
	"skipfish":    `skipfish`,
	"openvas":     `openvas`,
	"whatweb":     `whatweb`,
}

// scraperSignatures 数据采集器 / HTTP 库 UA 关键字。
var scraperSignatures = map[string]string{
	"curl":         `^curl/`,
	"wget":         `^wget/`,
	"python-requests": `python-requests`,
	"python-urllib":   `python-urllib`,
	"scrapy":       `scrapy`,
	"java":         `^java/`,
	"go-http-client": `go-http-client`,
	"okhttp":       `okhttp`,
	"php-curl":     `php/`,
	"ruby":         `^ruby`,
	"perl":         `^perl`,
	"node-fetch":   `node-fetch`,
	"axios":        `^axios`,
	"aiohttp":      `aiohttp`,
	"httpclient":   `^httpclient`,
}
