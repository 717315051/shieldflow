package dns

// DNSRecord 表示一条 DNS 解析记录
type DNSRecord struct {
	ID       string `json:"id"`       // 服务商返回的记录 ID
	Type     string `json:"type"`     // A / CNAME / TXT ...
	Name     string `json:"name"`     // 记录名称（相对域名或完整域名）
	Value    string `json:"value"`    // 记录值
	TTL      int    `json:"ttl"`      // 生存时间（秒），0 表示使用默认
	Priority int    `json:"priority"` // MX 优先级等
	Disabled bool   `json:"disabled"` // 是否暂停
}

// DNSProvider 抽象 DNS 服务商操作接口
//
// 参数说明：
//   - domain: 根域名（zone），如 example.com
//   - recordType: 记录类型，如 CNAME / A / AAAA / TXT
//   - name: 记录名称。对于 Cloudflare 为完整域名；对于阿里云/腾讯云为相对前缀（@ 或 www）
//   - value: 记录值
//   - ttl: 生存时间秒数，0 表示使用服务商默认值
type DNSProvider interface {
	// ProviderName 返回服务商标识（cloudflare / aliyun / tencent）
	ProviderName() string

	// CreateRecord 创建一条 DNS 记录
	CreateRecord(domain, recordType, name, value string, ttl int) error

	// UpdateRecord 更新一条 DNS 记录（按 name+type 匹配）
	UpdateRecord(domain, recordType, name, value string, ttl int) error

	// DeleteRecord 删除一条 DNS 记录（按 name+type 匹配）
	DeleteRecord(domain, recordType, name string) error

	// GetRecords 查询指定域名下的全部（或按类型过滤的）DNS 记录
	GetRecords(domain string) ([]DNSRecord, error)

	// ListDomains 列出该账户下的全部域名（zone）
	ListDomains() ([]string, error)
}

// ProviderConfig 是单个服务商的配置（供工厂函数使用）
type ProviderConfig struct {
	Enabled        bool
	APIToken       string // Cloudflare
	AccessKeyID    string // Aliyun
	AccessKeySecret string // Aliyun
	SecretID       string // Tencent
	SecretKey      string // Tencent
}
