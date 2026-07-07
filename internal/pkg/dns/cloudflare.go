package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// CloudflareProvider 实现 Cloudflare API v4 的 DNS Provider
// 文档: https://developers.cloudflare.com/api/
//
// 认证方式: API Token (Authorization: Bearer <token>)
//
// Cloudflare 中 zone 表示根域名（如 example.com）。
// 记录 name 使用完整域名形式 (www.example.com)，type 为大写。
type CloudflareProvider struct {
	apiToken  string
	apiBase   string
	httpc     *http.Client
	accountID string // 可选，用于列出 zones 时过滤
}

// NewCloudflareProvider 创建一个 Cloudflare DNS Provider
func NewCloudflareProvider(apiToken string) *CloudflareProvider {
	return &CloudflareProvider{
		apiToken: strings.TrimSpace(apiToken),
		apiBase:  "https://api.cloudflare.com/client/v4",
		httpc: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProviderName 返回服务商标识
func (p *CloudflareProvider) ProviderName() string { return "cloudflare" }

// cfResponse 是 Cloudflare API 通用响应包装
type cfResponse struct {
	Success bool            `json:"success"`
	Errors  []cfMessage     `json:"errors"`
	Messages []cfMessage    `json:"messages"`
	Result  json.RawMessage `json:"result"`
	ResultInfo *cfResultInfo `json:"result_info,omitempty"`
}

type cfMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cfResultInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalCount int `json:"total_count"`
}

// cfRecord 对应 Cloudflare DNS Record 资源
type cfRecord struct {
	ID         string `json:"id"`
	ZoneID     string `json:"zone_id"`
	ZoneName   string `json:"zone_name"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Content    string `json:"content"`
	TTL        int    `json:"ttl"`
	Proxied    bool   `json:"proxied"`
	Disabled   bool   `json:"disabled"`
	Priority   int    `json:"priority,omitempty"`
}

// cfZone 对应 Cloudflare Zone 资源
type cfZone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// do 执行一个 Cloudflare API 请求并解析通用响应
func (p *CloudflareProvider) do(method, path string, body interface{}) (*cfResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	endpoint := p.apiBase + path
	req, err := http.NewRequest(method, endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("new request %s %s: %w", method, endpoint, err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request %s %s: %w", method, endpoint, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var cfr cfResponse
	if err := json.Unmarshal(data, &cfr); err != nil {
		return nil, fmt.Errorf("unmarshal response (status=%d): %w; body=%s", resp.StatusCode, err, truncate(string(data), 500))
	}

	if resp.StatusCode >= 400 || !cfr.Success {
		var msgs []string
		for _, e := range cfr.Errors {
			msgs = append(msgs, fmt.Sprintf("code=%d msg=%s", e.Code, e.Message))
		}
		return &cfr, fmt.Errorf("cloudflare api error %s %s (status=%d): %s",
			method, path, resp.StatusCode, strings.Join(msgs, "; "))
	}

	return &cfr, nil
}

// getZoneID 通过域名查询对应的 Zone ID
func (p *CloudflareProvider) getZoneID(domain string) (string, error) {
	q := url.Values{}
	q.Set("name", domain)
	path := "/zones?" + q.Encode()

	cfr, err := p.do(http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}

	var zones []cfZone
	if err := json.Unmarshal(cfr.Result, &zones); err != nil {
		return "", fmt.Errorf("unmarshal zones: %w", err)
	}
	if len(zones) == 0 {
		return "", fmt.Errorf("zone not found for domain %q", domain)
	}
	return zones[0].ID, nil
}

// ensureTTL 规范化 TTL：0 表示自动（Cloudflare 用 1 表示自动）
func ensureTTL(ttl int) int {
	if ttl <= 0 {
		return 1 // Cloudflare: 1 = Automatic TTL
	}
	return ttl
}

// findRecord 在指定 zone 内查找匹配 type+name 的记录
func (p *CloudflareProvider) findRecord(zoneID, recordType, name string) (*cfRecord, error) {
	q := url.Values{}
	if recordType != "" {
		q.Set("type", recordType)
	}
	if name != "" {
		q.Set("name", name)
	}
	path := "/zones/" + zoneID + "/dns_records?" + q.Encode()

	cfr, err := p.do(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var recs []cfRecord
	if err := json.Unmarshal(cfr.Result, &recs); err != nil {
		return nil, fmt.Errorf("unmarshal dns records: %w", err)
	}
	for i := range recs {
		if strings.EqualFold(recs[i].Type, recordType) &&
			strings.EqualFold(recs[i].Name, name) {
			return &recs[i], nil
		}
	}
	return nil, nil
}

// CreateRecord 创建一条 DNS 记录
func (p *CloudflareProvider) CreateRecord(domain, recordType, name, value string, ttl int) error {
	zoneID, err := p.getZoneID(domain)
	if err != nil {
		return err
	}
	// 若已存在同 type+name 记录则视为成功（幂等）
	exist, err := p.findRecord(zoneID, recordType, name)
	if err != nil {
		return fmt.Errorf("check existing record: %w", err)
	}
	if exist != nil {
		// 已存在且值相同则跳过
		if strings.EqualFold(exist.Content, value) {
			return nil
		}
		// 否则走更新逻辑
		return p.updateRecordByID(zoneID, exist.ID, recordType, name, value, ttl)
	}

	body := map[string]interface{}{
		"type":    strings.ToUpper(recordType),
		"name":    name,
		"content": value,
		"ttl":     ensureTTL(ttl),
		"proxied": false,
	}
	if _, err := p.do(http.MethodPost, "/zones/"+zoneID+"/dns_records", body); err != nil {
		return fmt.Errorf("create record: %w", err)
	}
	return nil
}

// updateRecordByID 通过记录 ID 更新
func (p *CloudflareProvider) updateRecordByID(zoneID, recordID, recordType, name, value string, ttl int) error {
	body := map[string]interface{}{
		"type":    strings.ToUpper(recordType),
		"name":    name,
		"content": value,
		"ttl":     ensureTTL(ttl),
		"proxied": false,
	}
	if _, err := p.do(http.MethodPut, "/zones/"+zoneID+"/dns_records/"+recordID, body); err != nil {
		return fmt.Errorf("update record by id: %w", err)
	}
	return nil
}

// UpdateRecord 按 type+name 更新记录值
func (p *CloudflareProvider) UpdateRecord(domain, recordType, name, value string, ttl int) error {
	zoneID, err := p.getZoneID(domain)
	if err != nil {
		return err
	}
	rec, err := p.findRecord(zoneID, recordType, name)
	if err != nil {
		return fmt.Errorf("find record to update: %w", err)
	}
	if rec == nil {
		// 不存在则创建
		return p.CreateRecord(domain, recordType, name, value, ttl)
	}
	return p.updateRecordByID(zoneID, rec.ID, recordType, name, value, ttl)
}

// DeleteRecord 按 type+name 删除记录
func (p *CloudflareProvider) DeleteRecord(domain, recordType, name string) error {
	zoneID, err := p.getZoneID(domain)
	if err != nil {
		return err
	}
	rec, err := p.findRecord(zoneID, recordType, name)
	if err != nil {
		return fmt.Errorf("find record to delete: %w", err)
	}
	if rec == nil {
		return nil // 幂等：不存在即成功
	}
	if _, err := p.do(http.MethodDelete, "/zones/"+zoneID+"/dns_records/"+rec.ID, nil); err != nil {
		return fmt.Errorf("delete record: %w", err)
	}
	return nil
}

// GetRecords 查询指定域名下的全部 DNS 记录
func (p *CloudflareProvider) GetRecords(domain string) ([]DNSRecord, error) {
	zoneID, err := p.getZoneID(domain)
	if err != nil {
		return nil, err
	}

	var out []DNSRecord
	page := 1
	perPage := 100
	for {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("per_page", strconv.Itoa(perPage))
		path := "/zones/" + zoneID + "/dns_records?" + q.Encode()

		cfr, err := p.do(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		var recs []cfRecord
		if err := json.Unmarshal(cfr.Result, &recs); err != nil {
			return nil, fmt.Errorf("unmarshal dns records: %w", err)
		}
		for _, r := range recs {
			out = append(out, DNSRecord{
				ID:       r.ID,
				Type:     r.Type,
				Name:     r.Name,
				Value:    r.Content,
				TTL:      r.TTL,
				Priority: r.Priority,
				Disabled: r.Disabled,
			})
		}
		if cfr.ResultInfo == nil || page*perPage >= cfr.ResultInfo.TotalCount {
			break
		}
		page++
	}
	return out, nil
}

// ListDomains 列出账户下全部 zones
func (p *CloudflareProvider) ListDomains() ([]string, error) {
	var names []string
	page := 1
	perPage := 50
	for {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("per_page", strconv.Itoa(perPage))
		if p.accountID != "" {
			q.Set("account.id", p.accountID)
		}
		path := "/zones?" + q.Encode()

		cfr, err := p.do(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		var zones []cfZone
		if err := json.Unmarshal(cfr.Result, &zones); err != nil {
			return nil, fmt.Errorf("unmarshal zones: %w", err)
		}
		for _, z := range zones {
			names = append(names, z.Name)
		}
		if cfr.ResultInfo == nil || page*perPage >= cfr.ResultInfo.TotalCount || len(zones) < perPage {
			break
		}
		page++
	}
	return names, nil
}

// truncate 截断字符串用于错误信息
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
