package dns

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// TencentProvider 实现腾讯云 DNSPod (TCB DNS) 的 DNS Provider
// 文档: https://cloud.tencent.com/document/product/1427
//       https://cloud.tencent.com/document/api/1427
//
// 认证方式: 腾讯云 API v3 签名 (HMAC-SHA256, TC3-HMAC-SHA256)
//
// 腾讯云 DNSPod 中 Domain 为根域名，SubDomain 为相对前缀：
//   - 根域名: @
//   - www.example.com -> Domain=example.com, SubDomain=www
type TencentProvider struct {
	secretID  string
	secretKey string
	apiBase   string
	region    string // DNSPod 通常为空字符串
	httpc     *http.Client
}

// NewTencentProvider 创建腾讯云 DNSPod DNS Provider
func NewTencentProvider(secretID, secretKey string) *TencentProvider {
	return &TencentProvider{
		secretID:  strings.TrimSpace(secretID),
		secretKey: strings.TrimSpace(secretKey),
		apiBase:   "dnspod.tencentcloudapi.com",
		region:    "",
		httpc: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProviderName 返回服务商标识
func (p *TencentProvider) ProviderName() string { return "tencent" }

// tencentResponse 是腾讯云 API v3 通用响应包装
type tencentResponse struct {
	Response struct {
		RequestID string          `json:"RequestId"`
		Error     *tencentError   `json:"Error,omitempty"`
		Result    json.RawMessage `json:"-"` // 占位；实际字段在 RawResult 中
	} `json:"Response"`
}

type tencentError struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

// tencentRecordList 对应 DescribeRecordList 响应
type tencentRecordList struct {
	Response struct {
		RequestID string `json:"RequestId"`
		RecordCount int `json:"RecordCount"`
		Records []tencentRecord `json:"Records"`
	} `json:"Response"`
}

type tencentRecord struct {
	RecordID  json.Number `json:"RecordId"`
	Name      string      `json:"Name"`      // SubDomain
	Type      string      `json:"Type"`
	Value     string      `json:"Value"`
	TTL       json.Number `json:"TTL"`
	Status    string      `json:"Status"` // Enable / Disable
	MX        json.Number `json:"MX"`
}

// tencentDomainList 对应 DescribeDomainList 响应
type tencentDomainList struct {
	Response struct {
		RequestID string `json:"RequestId"`
		DomainCount int `json:"DomainCount"`
		DomainList []struct {
			Name string `json:"Name"`
		} `json:"DomainList"`
	} `json:"Response"`
}

// tencentActionResponse 用于返回 RecordId 的操作（CreateRecord/ModifyRecord）
type tencentActionResponse struct {
	Response struct {
		RequestID string      `json:"RequestId"`
		RecordID  json.Number `json:"RecordId"`
	} `json:"Response"`
}

// sha256Hex 计算 SHA256 的十六进制
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// hmacSHA256 计算 HMAC-SHA256
func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// signV3 计算腾讯云 API v3 签名并返回完整 HTTP 请求头
func (p *TencentProvider) signV3(service, action, payload string) (map[string]string, error) {
	algorithm := "TC3-HMAC-SHA256"
	timestamp := time.Now().Unix()
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")

	// 1. 拼接规范请求串
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := "content-type:application/json; charset=utf-8\n" +
		"host:" + p.apiBase + "\n" +
		"x-tc-action:" + strings.ToLower(action) + "\n"
	signedHeaders := "content-type;host;x-tc-action"
	hashedRequestPayload := sha256Hex([]byte(payload))
	canonicalRequest := canonicalURI + "\n" +
		canonicalQueryString + "\n" +
		canonicalHeaders + "\n" +
		signedHeaders + "\n" +
		hashedRequestPayload

	// 2. 拼接待签名串
	credentialScope := date + "/" + service + "/tc3_request"
	hashedCanonicalRequest := sha256Hex([]byte(canonicalRequest))
	stringToSign := algorithm + "\n" +
		fmt.Sprintf("%d", timestamp) + "\n" +
		credentialScope + "\n" +
		hashedCanonicalRequest

	// 3. 计算签名
	secretDate := hmacSHA256([]byte("TC3"+p.secretKey), []byte(date))
	secretService := hmacSHA256(secretDate, []byte(service))
	secretSigning := hmacSHA256(secretService, []byte("tc3_request"))
	signature := hex.EncodeToString(hmacSHA256(secretSigning, []byte(stringToSign)))

	// 4. 拼接 Authorization
	authorization := algorithm + " " +
		"Credential=" + p.secretID + "/" + credentialScope + ", " +
		"SignedHeaders=" + signedHeaders + ", " +
		"Signature=" + signature

	headers := map[string]string{
		"Authorization":  authorization,
		"Content-Type":   "application/json; charset=utf-8",
		"Host":           p.apiBase,
		"X-TC-Action":    action,
		"X-TC-Timestamp": fmt.Sprintf("%d", timestamp),
		"X-TC-Version":   "2021-03-23",
	}
	if p.region != "" {
		headers["X-TC-Region"] = p.region
	}
	return headers, nil
}

// call 执行一个腾讯云 API v3 调用
func (p *TencentProvider) call(action string, bizParams map[string]interface{}) ([]byte, error) {
	payload, err := json.Marshal(bizParams)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	headers, err := p.signV3("dnspod", action, string(payload))
	if err != nil {
		return nil, fmt.Errorf("sign request: %w", err)
	}

	endpoint := "https://" + p.apiBase + "/"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := p.httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// 解析通用响应检查错误
	var tr tencentResponse
	if err := json.Unmarshal(data, &tr); err != nil {
		return nil, fmt.Errorf("unmarshal response (status=%d): %w; body=%s",
			resp.StatusCode, err, truncate(string(data), 500))
	}
	if tr.Response.Error != nil {
		return data, fmt.Errorf("tencent api error action=%s code=%s message=%s",
			action, tr.Response.Error.Code, tr.Response.Error.Message)
	}
	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("tencent api http error action=%s status=%d body=%s",
			action, resp.StatusCode, truncate(string(data), 500))
	}
	return data, nil
}

// subDomain 将完整记录名转换为 SubDomain 相对前缀
func subDomain(name, domain string) string {
	if name == domain || name == "@" {
		return "@"
	}
	if strings.HasSuffix(name, "."+domain) {
		sd := strings.TrimSuffix(name, "."+domain)
		if sd == "" {
			return "@"
		}
		return sd
	}
	return name
}

// ensureTencentTTL 腾讯云 DNSPod TTL 最小 600
func ensureTencentTTL(ttl int) int {
	if ttl <= 0 || ttl < 600 {
		return 600
	}
	return ttl
}

// findRecord 查找指定 SubDomain+Type 的记录
func (p *TencentProvider) findRecord(domain, subDomainName, recordType string) (*tencentRecord, error) {
	data, err := p.call("DescribeRecordList", map[string]interface{}{
		"Domain":   domain,
		"Subdomain": subDomainName,
		"Type":     strings.ToUpper(recordType),
		"Limit":    500,
	})
	if err != nil {
		return nil, err
	}
	var rl tencentRecordList
	if err := json.Unmarshal(data, &rl); err != nil {
		return nil, fmt.Errorf("unmarshal record list: %w", err)
	}
	for i := range rl.Response.Records {
		r := &rl.Response.Records[i]
		if r.Name == subDomainName && strings.EqualFold(r.Type, recordType) {
			return r, nil
		}
	}
	return nil, nil
}

// CreateRecord 创建一条 DNS 记录
func (p *TencentProvider) CreateRecord(domain, recordType, name, value string, ttl int) error {
	sd := subDomain(name, domain)

	// 幂等检查
	existing, err := p.findRecord(domain, sd, recordType)
	if err != nil {
		return fmt.Errorf("check existing record: %w", err)
	}
	if existing != nil {
		if existing.Value == value {
			return nil
		}
		// 已存在但值不同则更新
		var rid int64
		fmt.Sscanf(existing.RecordID.String(), "%d", &rid)
		_, err := p.call("ModifyRecord", map[string]interface{}{
			"Domain":    domain,
			"RecordId":  rid,
			"SubDomain": sd,
			"RecordType": strings.ToUpper(recordType),
			"RecordLine": "默认",
			"Value":     value,
			"TTL":       ensureTencentTTL(ttl),
		})
		if err != nil {
			return fmt.Errorf("modify existing record: %w", err)
		}
		return nil
	}

	_, err = p.call("CreateRecord", map[string]interface{}{
		"Domain":     domain,
		"SubDomain":  sd,
		"RecordType": strings.ToUpper(recordType),
		"RecordLine": "默认",
		"Value":      value,
		"TTL":        ensureTencentTTL(ttl),
	})
	if err != nil {
		return fmt.Errorf("create record: %w", err)
	}
	return nil
}

// UpdateRecord 更新一条 DNS 记录
func (p *TencentProvider) UpdateRecord(domain, recordType, name, value string, ttl int) error {
	sd := subDomain(name, domain)
	existing, err := p.findRecord(domain, sd, recordType)
	if err != nil {
		return fmt.Errorf("find record to update: %w", err)
	}
	if existing == nil {
		return p.CreateRecord(domain, recordType, name, value, ttl)
	}
	var rid int64
	fmt.Sscanf(existing.RecordID.String(), "%d", &rid)
	_, err = p.call("ModifyRecord", map[string]interface{}{
		"Domain":     domain,
		"RecordId":   rid,
		"SubDomain":  sd,
		"RecordType": strings.ToUpper(recordType),
		"RecordLine": "默认",
		"Value":      value,
		"TTL":        ensureTencentTTL(ttl),
	})
	if err != nil {
		return fmt.Errorf("modify record: %w", err)
	}
	return nil
}

// DeleteRecord 删除一条 DNS 记录
func (p *TencentProvider) DeleteRecord(domain, recordType, name string) error {
	sd := subDomain(name, domain)
	existing, err := p.findRecord(domain, sd, recordType)
	if err != nil {
		return fmt.Errorf("find record to delete: %w", err)
	}
	if existing == nil {
		return nil // 幂等
	}
	var rid int64
	fmt.Sscanf(existing.RecordID.String(), "%d", &rid)
	_, err = p.call("DeleteRecord", map[string]interface{}{
		"Domain":   domain,
		"RecordId": rid,
	})
	if err != nil {
		return fmt.Errorf("delete record: %w", err)
	}
	return nil
}

// GetRecords 查询指定域名下的全部 DNS 记录
func (p *TencentProvider) GetRecords(domain string) ([]DNSRecord, error) {
	var out []DNSRecord
	offset := 0
	limit := 3000
	for {
		data, err := p.call("DescribeRecordList", map[string]interface{}{
			"Domain": domain,
			"Limit":  limit,
			"Offset": offset,
		})
		if err != nil {
			return nil, err
		}
		var rl tencentRecordList
		if err := json.Unmarshal(data, &rl); err != nil {
			return nil, fmt.Errorf("unmarshal record list: %w", err)
		}
		for _, r := range rl.Response.Records {
			ttlVal := 600
			if n, err := r.TTL.Int64(); err == nil {
				ttlVal = int(n)
			}
			priority := 0
			if n, err := r.MX.Int64(); err == nil {
				priority = int(n)
			}
			fullName := r.Name
			if r.Name == "@" {
				fullName = domain
			} else {
				fullName = r.Name + "." + domain
			}
			out = append(out, DNSRecord{
				ID:       r.RecordID.String(),
				Type:     r.Type,
				Name:     fullName,
				Value:    r.Value,
				TTL:      ttlVal,
				Priority: priority,
				Disabled: r.Status == "Disable",
			})
		}
		if len(rl.Response.Records) == 0 || len(out) >= rl.Response.RecordCount {
			break
		}
		offset += limit
	}
	return out, nil
}

// ListDomains 列出账户下的全部域名
func (p *TencentProvider) ListDomains() ([]string, error) {
	var out []string
	offset := 0
	limit := 3000
	for {
		data, err := p.call("DescribeDomainList", map[string]interface{}{
			"Limit":  limit,
			"Offset": offset,
		})
		if err != nil {
			return nil, err
		}
		var dl tencentDomainList
		if err := json.Unmarshal(data, &dl); err != nil {
			return nil, fmt.Errorf("unmarshal domain list: %w", err)
		}
		for _, d := range dl.Response.DomainList {
			out = append(out, d.Name)
		}
		if len(dl.Response.DomainList) == 0 || len(out) >= dl.Response.DomainCount {
			break
		}
		offset += limit
	}
	// 排序保证稳定
	sort.Strings(out)
	return out, nil
}
