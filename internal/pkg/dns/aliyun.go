package dns

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// AliyunProvider 实现阿里云 DNS (Alidns) 的 DNS Provider
// 文档: https://help.aliyun.com/document_detail/29776.html
//
// 认证方式: AccessKey 签名 (RPC 风格，HMAC-SHA1)
//
// 阿里云 DNS 中 domain 为根域名，RR（Resource Record）为相对前缀：
//   - 根域名: @
//   - www.example.com -> domain=example.com, RR=www
type AliyunProvider struct {
	accessKeyID     string
	accessKeySecret string
	apiBase         string
	httpc           *http.Client
}

// NewAliyunProvider 创建阿里云 DNS Provider
func NewAliyunProvider(accessKeyID, accessKeySecret string) *AliyunProvider {
	return &AliyunProvider{
		accessKeyID:     strings.TrimSpace(accessKeyID),
		accessKeySecret: strings.TrimSpace(accessKeySecret),
		apiBase:         "https://alidns.aliyuncs.com",
		httpc: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ProviderName 返回服务商标识
func (p *AliyunProvider) ProviderName() string { return "aliyun" }

// aliyunCommonResponse 是阿里云 RPC 通用响应包装
type aliyunCommonResponse struct {
	// 公共错误字段
	Code      string `json:"Code"`
	Message   string `json:"Message"`
	RequestId string `json:"RequestId"`
	// 成功时的业务字段（动态，由具体调用方解析）
	TotalCount int               `json:"TotalCount"`
	DomainRecords struct {
		Record []aliyunRecord `json:"Record"`
	} `json:"DomainRecords"`
	Domains struct {
		Domain []aliyunDomain `json:"Domain"`
	} `json:"Domains"`
	RecordID string `json:"RecordId"`
}

type aliyunRecord struct {
	RecordID string `json:"RecordId"`
	RR       string `json:"RR"`
	Type     string `json:"Type"`
	Value    string `json:"Value"`
	TTL      json.Number `json:"TTL"`
	Status   string `json:"Status"` // Enable / Disable
	Priority int    `json:"Priority"`
}

type aliyunDomain struct {
	DomainName string `json:"DomainName"`
}

// percentEncode 按 RFC 3986 对字符串进行 percent-encode（阿里云规范）
func percentEncode(s string) string {
	encoded := url.QueryEscape(s)
	// 阿里云要求 + -> %20, * -> %2A, %7E -> ~
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

// sign 计算阿里云 RPC 签名
func (p *AliyunProvider) sign(params map[string]string) string {
	// 1. 排序参数
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 构造 canonicalized query string
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(percentEncode(k))
		sb.WriteString("=")
		sb.WriteString(percentEncode(params[k]))
	}
	canonicalized := sb.String()

	// 3. 构造待签名字符串
	stringToSign := http.MethodGet + "&" + percentEncode("/") + "&" + percentEncode(canonicalized)

	// 4. HMAC-SHA1
	mac := hmac.New(sha1.New, []byte(p.accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return signature
}

// call 执行一个阿里云 RPC API 调用
func (p *AliyunProvider) call(action string, bizParams map[string]string) (*aliyunCommonResponse, error) {
	// 公共参数
	params := map[string]string{
		"Format":           "JSON",
		"Version":          "2015-01-09",
		"AccessKeyId":      p.accessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce":   fmt.Sprintf("%d%d", time.Now().UnixNano(), time.Now().Nanosecond()),
		"Action":           action,
	}
	// 合并业务参数
	for k, v := range bizParams {
		params[k] = v
	}

	// 计算签名
	params["Signature"] = p.sign(params)

	// 构造 query string
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	endpoint := p.apiBase + "/?" + q.Encode()

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
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

	var ar aliyunCommonResponse
	if err := json.Unmarshal(data, &ar); err != nil {
		return nil, fmt.Errorf("unmarshal response (status=%d): %w; body=%s",
			resp.StatusCode, err, truncate(string(data), 500))
	}

	// 阿里云错误码为空字符串表示成功；非空且非 "True"/"Success" 视为错误
	if ar.Code != "" {
		return &ar, fmt.Errorf("aliyun api error action=%s code=%s message=%s (status=%d)",
			action, ar.Code, ar.Message, resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return &ar, fmt.Errorf("aliyun api http error action=%s status=%d body=%s",
			action, resp.StatusCode, truncate(string(data), 500))
	}

	return &ar, nil
}

// splitRR 将完整记录名拆分为 (domain, RR)
// 例如 www.example.com -> ("example.com", "www")
//      example.com -> ("example.com", "@")
func splitRR(full string) (domain, rr string, err error) {
	full = strings.TrimSuffix(full, ".")
	parts := strings.Split(full, ".")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid record name %q: need at least domain.tld", full)
	}
	if len(parts) == 2 {
		return full, "@", nil
	}
	// 默认取最后两段为域名，其余为 RR
	domain = strings.Join(parts[len(parts)-2:], ".")
	rr = strings.Join(parts[:len(parts)-2], ".")
	return domain, rr, nil
}

// ensureAliyunTTL 阿里云 TTL 最小 600
func ensureAliyunTTL(ttl int) string {
	if ttl <= 0 || ttl < 600 {
		return "600"
	}
	return fmt.Sprintf("%d", ttl)
}

// CreateRecord 创建一条 DNS 记录
func (p *AliyunProvider) CreateRecord(domain, recordType, name, value string, ttl int) error {
	rr := "@"
	// name 可能是完整域名或相对前缀
	if name == domain || name == "@" {
		rr = "@"
	} else if strings.HasSuffix(name, "."+domain) {
		rr = strings.TrimSuffix(name, "."+domain)
		if rr == "" {
			rr = "@"
		}
	} else {
		// 当 name 不是 domain 子域时，按完整域名拆分
		d, r, err := splitRR(name)
		if err == nil && d == domain {
			rr = r
		} else {
			rr = name
		}
	}

	// 幂等检查：若已存在同 RR+Type 记录则更新
	existing, err := p.findRecord(domain, rr, recordType)
	if err != nil {
		return fmt.Errorf("check existing record: %w", err)
	}
	if existing != nil {
		if existing.Value == value {
			return nil
		}
		_, err = p.call("UpdateDomainRecord", map[string]string{
			"RecordId": existing.RecordID,
			"RR":       rr,
			"Type":     strings.ToUpper(recordType),
			"Value":    value,
			"TTL":      ensureAliyunTTL(ttl),
		})
		if err != nil {
			return fmt.Errorf("update existing record: %w", err)
		}
		return nil
	}

	_, err = p.call("AddDomainRecord", map[string]string{
		"DomainName": domain,
		"RR":         rr,
		"Type":       strings.ToUpper(recordType),
		"Value":      value,
		"TTL":        ensureAliyunTTL(ttl),
	})
	if err != nil {
		return fmt.Errorf("create record: %w", err)
	}
	return nil
}

// findRecord 查找指定 RR+Type 的记录
func (p *AliyunProvider) findRecord(domain, rr, recordType string) (*aliyunRecord, error) {
	ar, err := p.call("DescribeDomainRecords", map[string]string{
		"DomainName": domain,
		"RRKeyWord":  rr,
		"TypeKeyWord": strings.ToUpper(recordType),
	})
	if err != nil {
		return nil, err
	}
	for i := range ar.DomainRecords.Record {
		rec := &ar.DomainRecords.Record[i]
		if rec.RR == rr && strings.EqualFold(rec.Type, recordType) {
			return rec, nil
		}
	}
	return nil, nil
}

// UpdateRecord 更新一条 DNS 记录
func (p *AliyunProvider) UpdateRecord(domain, recordType, name, value string, ttl int) error {
	rr := "@"
	if name == domain || name == "@" {
		rr = "@"
	} else if strings.HasSuffix(name, "."+domain) {
		rr = strings.TrimSuffix(name, "."+domain)
		if rr == "" {
			rr = "@"
		}
	} else {
		rr = name
	}

	existing, err := p.findRecord(domain, rr, recordType)
	if err != nil {
		return fmt.Errorf("find record to update: %w", err)
	}
	if existing == nil {
		return p.CreateRecord(domain, recordType, name, value, ttl)
	}
	_, err = p.call("UpdateDomainRecord", map[string]string{
		"RecordId": existing.RecordID,
		"RR":       rr,
		"Type":     strings.ToUpper(recordType),
		"Value":    value,
		"TTL":      ensureAliyunTTL(ttl),
	})
	if err != nil {
		return fmt.Errorf("update record: %w", err)
	}
	return nil
}

// DeleteRecord 删除一条 DNS 记录
func (p *AliyunProvider) DeleteRecord(domain, recordType, name string) error {
	rr := "@"
	if name == domain || name == "@" {
		rr = "@"
	} else if strings.HasSuffix(name, "."+domain) {
		rr = strings.TrimSuffix(name, "."+domain)
		if rr == "" {
			rr = "@"
		}
	} else {
		rr = name
	}

	existing, err := p.findRecord(domain, rr, recordType)
	if err != nil {
		return fmt.Errorf("find record to delete: %w", err)
	}
	if existing == nil {
		return nil // 幂等
	}
	_, err = p.call("DeleteDomainRecord", map[string]string{
		"RecordId": existing.RecordID,
	})
	if err != nil {
		return fmt.Errorf("delete record: %w", err)
	}
	return nil
}

// GetRecords 查询指定域名下的全部 DNS 记录
func (p *AliyunProvider) GetRecords(domain string) ([]DNSRecord, error) {
	pageSize := "500"
	pageNumber := "1"
	var out []DNSRecord
	for {
		ar, err := p.call("DescribeDomainRecords", map[string]string{
			"DomainName": domain,
			"PageSize":   pageSize,
			"PageNumber": pageNumber,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range ar.DomainRecords.Record {
			ttlStr := r.TTL.String()
			var ttlVal int
			fmt.Sscanf(ttlStr, "%d", &ttlVal)
			// 构造完整记录名
			fullName := r.RR
			if r.RR == "@" {
				fullName = domain
			} else {
				fullName = r.RR + "." + domain
			}
			out = append(out, DNSRecord{
				ID:       r.RecordID,
				Type:     r.Type,
				Name:     fullName,
				Value:    r.Value,
				TTL:      ttlVal,
				Priority: r.Priority,
				Disabled: r.Status == "Disable",
			})
		}
		if len(ar.DomainRecords.Record) == 0 || len(out) >= ar.TotalCount {
			break
		}
		// 下一页
		var pn int
		fmt.Sscanf(pageNumber, "%d", &pn)
		pageNumber = fmt.Sprintf("%d", pn+1)
	}
	return out, nil
}

// ListDomains 列出账户下的全部域名
func (p *AliyunProvider) ListDomains() ([]string, error) {
	pageSize := "50"
	pageNumber := "1"
	var out []string
	for {
		ar, err := p.call("DescribeDomains", map[string]string{
			"PageSize":   pageSize,
			"PageNumber": pageNumber,
		})
		if err != nil {
			return nil, err
		}
		for _, d := range ar.Domains.Domain {
			out = append(out, d.DomainName)
		}
		if len(ar.Domains.Domain) == 0 || len(out) >= ar.TotalCount {
			break
		}
		var pn int
		fmt.Sscanf(pageNumber, "%d", &pn)
		pageNumber = fmt.Sprintf("%d", pn+1)
	}
	return out, nil
}


