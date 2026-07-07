package dns

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// DNSStatus 表示域名 DNS 同步状态
type DNSStatus string

const (
	DNSStatusActive   DNSStatus = "active"    // CNAME 已正确指向
	DNSStatusPending  DNSStatus = "pending"   // CNAME 未指向或指向错误
	DNSStatusError    DNSStatus = "error"     // 同步异常
	DNSStatusNotFound DNSStatus = "not_found" // 域名不在任何 Provider 中
)

// SyncConfig 同步配置
type SyncConfig struct {
	Interval int // 同步间隔（秒）
	Retry    int // 错误重试次数
}

// DomainEntry 是待同步的域名条目
type DomainEntry struct {
	Domain     string // 根域名，如 example.com
	CNAMEName  string // CNAME 记录名，如 www.example.com 或 www
	CNAMEValue string // CDN 分配的 CNAME 目标，如 www.example.com.cdn.shieldflow.cn
	Provider   string // 指定 provider 名称（可选，空则使用默认 provider）
}

// SyncResult 单个域名的同步结果
type SyncResult struct {
	Domain     string    `json:"domain"`
	CNAMEName  string    `json:"cname_name"`
	CNAMEValue string    `json:"cname_value"`
	Provider   string    `json:"provider"`
	Status     DNSStatus `json:"status"`
	Message    string    `json:"message,omitempty"`
	CheckedAt  time.Time `json:"checked_at"`
}

// SyncReport 是一次同步任务的汇总报告
type SyncReport struct {
	StartedAt  time.Time    `json:"started_at"`
	FinishedAt time.Time    `json:"finished_at"`
	Total      int          `json:"total"`
	Active     int          `json:"active"`
	Pending    int          `json:"pending"`
	Error      int          `json:"error"`
	NotFound   int          `json:"not_found"`
	Results    []SyncResult `json:"results"`
}

// Manager 管理 DNS 同步：选择 Provider、批量同步、状态检查、错误重试
type Manager struct {
	providers      map[string]DNSProvider // 按名称索引
	defaultProvider string
	syncCfg         SyncConfig
	logger          *zap.Logger
	mu              sync.Mutex
	lastReport      *SyncReport
}

// NewManager 创建一个 DNS 管理器
//
// providers 是按名称 -> Provider 的映射；defaultProvider 指定默认使用的 provider
// 当 providers 为空或没有任何 enabled 的 provider 时，Manager 仍可创建但同步将跳过。
func NewManager(providers map[string]DNSProvider, defaultProvider string, cfg SyncConfig, logger *zap.Logger) *Manager {
	if logger == nil {
		logger = zap.NewNop()
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 300
	}
	if cfg.Retry <= 0 {
		cfg.Retry = 3
	}
	m := &Manager{
		providers:       providers,
		defaultProvider: defaultProvider,
		syncCfg:         cfg,
		logger:          logger,
	}
	return m
}

// selectProvider 根据条目配置选择 Provider，回退到默认
func (m *Manager) selectProvider(entry DomainEntry) DNSProvider {
	name := entry.Provider
	if name == "" {
		name = m.defaultProvider
	}
	if name == "" {
		// 取第一个可用 provider
		for n, p := range m.providers {
			m.defaultProvider = n
			return p
		}
	}
	return m.providers[name]
}

// syncOnce 同步单个域名（带重试）
func (m *Manager) syncOnce(entry DomainEntry) SyncResult {
	res := SyncResult{
		Domain:     entry.Domain,
		CNAMEName:  entry.CNAMEName,
		CNAMEValue: entry.CNAMEValue,
		CheckedAt:  time.Now(),
	}

	provider := m.selectProvider(entry)
	if provider == nil {
		res.Status = DNSStatusNotFound
		res.Message = "no DNS provider available"
		return res
	}
	res.Provider = provider.ProviderName()

	// 确保记录名是完整域名形式（部分 Provider 接受相对前缀也接受完整域名）
	cnameName := entry.CNAMEName
	if cnameName == "" || cnameName == "@" {
		cnameName = entry.Domain
	}
	if !strings.Contains(cnameName, ".") {
		cnameName = cnameName + "." + entry.Domain
	}

	// 重试创建/更新 CNAME 记录
	var lastErr error
	for attempt := 1; attempt <= m.syncCfg.Retry; attempt++ {
		err := provider.UpdateRecord(entry.Domain, "CNAME", cnameName, entry.CNAMEValue, 600)
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = err
		m.logger.Warn("DNS record update failed, retrying",
			zap.String("domain", entry.Domain),
			zap.String("provider", provider.ProviderName()),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", m.syncCfg.Retry),
			zap.Error(err))
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	if lastErr != nil {
		res.Status = DNSStatusError
		res.Message = fmt.Sprintf("update record failed after %d retries: %v", m.syncCfg.Retry, lastErr)
		return res
	}

	// 状态检查：CNAME 是否正确指向
	correct, err := m.checkCNAME(provider, entry.Domain, cnameName, entry.CNAMEValue)
	if err != nil {
		res.Status = DNSStatusError
		res.Message = fmt.Sprintf("status check failed: %v", err)
		return res
	}
	if correct {
		res.Status = DNSStatusActive
	} else {
		res.Status = DNSStatusPending
		res.Message = "CNAME not pointing to expected CDN target"
	}
	return res
}

// checkCNAME 检查 DNS 记录是否与预期匹配
func (m *Manager) checkCNAME(provider DNSProvider, domain, name, expectedValue string) (bool, error) {
	records, err := provider.GetRecords(domain)
	if err != nil {
		return false, fmt.Errorf("get records: %w", err)
	}
	for _, r := range records {
		if !strings.EqualFold(r.Type, "CNAME") {
			continue
		}
		// 名称匹配：完整域名或相对前缀
		if strings.EqualFold(r.Name, name) ||
			strings.EqualFold(r.Name, strings.TrimSuffix(name, "."+domain)) {
			if strings.EqualFold(strings.TrimSuffix(r.Value, "."), strings.TrimSuffix(expectedValue, ".")) {
				return true, nil
			}
		}
	}
	return false, nil
}

// Sync 批量同步多个域名条目
func (m *Manager) Sync(entries []DomainEntry) *SyncReport {
	report := &SyncReport{
		StartedAt: time.Now(),
	}
	m.logger.Info("DNS sync started",
		zap.Int("entries", len(entries)),
		zap.Int("interval_sec", m.syncCfg.Interval))

	for _, entry := range entries {
		res := m.syncOnce(entry)
		report.Results = append(report.Results, res)
		report.Total++
		switch res.Status {
		case DNSStatusActive:
			report.Active++
		case DNSStatusPending:
			report.Pending++
		case DNSStatusError:
			report.Error++
		case DNSStatusNotFound:
			report.NotFound++
		}
		m.logger.Info("DNS sync entry",
			zap.String("domain", res.Domain),
			zap.String("provider", res.Provider),
			zap.String("status", string(res.Status)),
			zap.String("message", res.Message))
	}

	report.FinishedAt = time.Now()
	m.mu.Lock()
	m.lastReport = report
	m.mu.Unlock()

	m.logger.Info("DNS sync finished",
		zap.Int("total", report.Total),
		zap.Int("active", report.Active),
		zap.Int("pending", report.Pending),
		zap.Int("error", report.Error),
		zap.Int("not_found", report.NotFound),
		zap.Duration("duration", report.FinishedAt.Sub(report.StartedAt)))
	return report
}

// LastReport 返回最近一次同步报告（线程安全）
func (m *Manager) LastReport() *SyncReport {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastReport
}

// CreateCNAME 为单个域名创建/更新 CNAME 记录
func (m *Manager) CreateCNAME(entry DomainEntry) error {
	provider := m.selectProvider(entry)
	if provider == nil {
		return fmt.Errorf("no DNS provider available for domain %s", entry.Domain)
	}
	cnameName := entry.CNAMEName
	if cnameName == "" || cnameName == "@" {
		cnameName = entry.Domain
	}
	if !strings.Contains(cnameName, ".") {
		cnameName = cnameName + "." + entry.Domain
	}
	return provider.UpdateRecord(entry.Domain, "CNAME", cnameName, entry.CNAMEValue, 600)
}

// DeleteCNAME 删除单个域名的 CNAME 记录（域名下线/删除时调用）
func (m *Manager) DeleteCNAME(entry DomainEntry) error {
	provider := m.selectProvider(entry)
	if provider == nil {
		return fmt.Errorf("no DNS provider available for domain %s", entry.Domain)
	}
	cnameName := entry.CNAMEName
	if cnameName == "" || cnameName == "@" {
		cnameName = entry.Domain
	}
	if !strings.Contains(cnameName, ".") {
		cnameName = cnameName + "." + entry.Domain
	}
	return provider.DeleteRecord(entry.Domain, "CNAME", cnameName)
}

// CheckStatus 检查指定域名 CNAME 是否正确指向
func (m *Manager) CheckStatus(entry DomainEntry) (DNSStatus, error) {
	provider := m.selectProvider(entry)
	if provider == nil {
		return DNSStatusNotFound, fmt.Errorf("no DNS provider available for domain %s", entry.Domain)
	}
	cnameName := entry.CNAMEName
	if cnameName == "" || cnameName == "@" {
		cnameName = entry.Domain
	}
	if !strings.Contains(cnameName, ".") {
		cnameName = cnameName + "." + entry.Domain
	}
	correct, err := m.checkCNAME(provider, entry.Domain, cnameName, entry.CNAMEValue)
	if err != nil {
		return DNSStatusError, err
	}
	if correct {
		return DNSStatusActive, nil
	}
	return DNSStatusPending, nil
}

// Providers 返回已注册的 provider 名称列表
func (m *Manager) Providers() []string {
	names := make([]string, 0, len(m.providers))
	for n := range m.providers {
		names = append(names, n)
	}
	return names
}
