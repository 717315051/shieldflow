package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Email        string         `gorm:"size:100;uniqueIndex" json:"email"`
	Phone        string         `gorm:"size:20" json:"phone"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	Role         string         `gorm:"size:20;default:user" json:"role"` // user / admin
	Status       string         `gorm:"size:20;default:active" json:"status"` // active / disabled
	RealName     string         `gorm:"size:50" json:"real_name"`
	IDCard       string         `gorm:"size:18" json:"id_card"`
	Verified     bool           `gorm:"default:false" json:"verified"` // 实名认证
	Balance      float64        `gorm:"default:0" json:"balance"`
	FrozenBalance float64       `gorm:"default:0" json:"frozen_balance"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Node 边缘节点
type Node struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"size:100;not null" json:"name"`
	IP            string         `gorm:"size:45;not null" json:"ip"`
	Region        string         `gorm:"size:50" json:"region"`
	GroupID       *uint          `gorm:"index" json:"group_id"`
	Status        string         `gorm:"size:20;default:offline" json:"status"` // online / offline
	LicenseKey    string         `gorm:"size:100" json:"license_key"`
	GRPCAddress   string         `gorm:"size:255" json:"grpc_address"`
	Version       string         `gorm:"size:50" json:"version"`
	CPU           int            `gorm:"default:0" json:"cpu"`        // 核数
	Memory        int            `gorm:"default:0" json:"memory"`      // MB
	Disk          int            `gorm:"default:0" json:"disk"`        // GB
	Bandwidth     int            `gorm:"default:0" json:"bandwidth"`   // Mbps
	ConnCount     int            `gorm:"default:0" json:"conn_count"`  // 当前连接数
	QPS           int            `gorm:"default:0" json:"qps"`
	LastHeartbeat *time.Time     `json:"last_heartbeat"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// NodeGroup 节点分组
type NodeGroup struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Domain 域名
type Domain struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            uint           `gorm:"index;not null" json:"user_id"`
	DomainName        string         `gorm:"size:255;uniqueIndex;not null" json:"domain_name"`
	PackageID         *uint          `gorm:"index" json:"package_id"`
	CNAME             string         `gorm:"size:255" json:"cname"`
	Status            string         `gorm:"size:20;default:pending" json:"status"` // pending / active / disabled
	OriginConfig      string         `gorm:"type:text" json:"origin_config"`       // JSON
	HTTPSConfig       string         `gorm:"type:text" json:"https_config"`        // JSON
	CacheConfig       string         `gorm:"type:text" json:"cache_config"`        // JSON
	AdvancedConfig    string         `gorm:"type:text" json:"advanced_config"`     // JSON
	CustomHeaders     string         `gorm:"type:text" json:"custom_headers"`      // JSON
	CustomPages       string         `gorm:"type:text" json:"custom_pages"`        // JSON
	ProtectionConfig  string         `gorm:"type:text" json:"protection_config"`   // JSON
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// Package 套餐
type Package struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	Type           string         `gorm:"size:10;not null" json:"type"` // l7 / l4
	TrafficLimit   int64          `gorm:"default:0" json:"traffic_limit"`     // bytes
	BandwidthLimit int            `gorm:"default:0" json:"bandwidth_limit"`   // Mbps
	DomainLimit    int            `gorm:"default:0" json:"domain_limit"`
	Price          float64        `gorm:"default:0" json:"price"`
	DurationDays   int            `gorm:"default:30" json:"duration_days"`
	Status         string         `gorm:"size:20;default:active" json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// UserPackage 用户套餐实例
type UserPackage struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	UserID        uint           `gorm:"index;not null" json:"user_id"`
	PackageID     uint           `gorm:"not null" json:"package_id"`
	InstanceNo    string         `gorm:"size:50;uniqueIndex" json:"instance_no"`
	TrafficUsed   int64          `gorm:"default:0" json:"traffic_used"`  // bytes
	TrafficLimit  int64          `gorm:"default:0" json:"traffic_limit"`
	DomainCount   int            `gorm:"default:0" json:"domain_count"`
	Status        string         `gorm:"size:20;default:active" json:"status"`
	ExpiresAt     *time.Time     `json:"expires_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// Certificate SSL证书
type Certificate struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"index;not null" json:"user_id"`
	DomainID    *uint          `gorm:"index" json:"domain_id"`
	CertPEM     string         `gorm:"type:text" json:"-"`
	KeyPEM      string         `gorm:"type:text" json:"-"`
	Issuer      string         `gorm:"size:100" json:"issuer"`
	CommonName  string         `gorm:"size:255" json:"common_name"`
	SAN         string         `gorm:"type:text" json:"san"` // Subject Alternative Names
	NotBefore   *time.Time     `json:"not_before"`
	NotAfter    *time.Time     `json:"not_after"`
	Status      string         `gorm:"size:20;default:active" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// AcmeAccount ACME账户
type AcmeAccount struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	Email     string         `gorm:"size:100;not null" json:"email"`
	Key       string         `gorm:"type:text" json:"-"` // private key
	CAURL     string         `gorm:"size:255" json:"ca_url"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// DNSAccount DNS服务商账户
type DNSAccount struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	Provider  string         `gorm:"size:50;not null" json:"provider"` // cloudflare / aliyun / tencent / dnspod
	APIKey    string         `gorm:"size:255" json:"-"`
	APISecret string         `gorm:"size:255" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BlacklistEntry 黑白名单条目
type BlacklistEntry struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"index" json:"user_id"`
	DomainID   *uint          `gorm:"index" json:"domain_id"`
	Type       string         `gorm:"size:20;not null" json:"type"`       // ip / url / domain
	ListType   string         `gorm:"size:20;not null" json:"list_type"`   // black / white
	Value      string         `gorm:"size:255;not null" json:"value"`
	MatchMode  string         `gorm:"size:20;default:exact" json:"match_mode"` // exact / prefix / regex / cidr
	HTTPMethod string         `gorm:"size:10" json:"http_method"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// ProtectionTemplate 防护策略模板
type ProtectionTemplate struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"index" json:"user_id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Description string         `gorm:"size:500" json:"description"`
	IsDefault   bool           `gorm:"default:false" json:"is_default"`
	IsSystem    bool           `gorm:"default:false" json:"is_system"`
	Config      string         `gorm:"type:text" json:"config"` // JSON
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Layer4Forward 四层转发
type Layer4Forward struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       uint           `gorm:"index;not null" json:"user_id"`
	DomainID     *uint          `gorm:"index" json:"domain_id"`
	PackageID    *uint          `gorm:"index" json:"package_id"`
	Protocol     string         `gorm:"size:10;not null" json:"protocol"` // tcp / udp
	ListenPort   int            `gorm:"not null" json:"listen_port"`
	LBStrategy   string         `gorm:"size:20;default:round_robin" json:"lb_strategy"`
	Origins      string         `gorm:"type:text" json:"origins"`     // JSON
	Advanced     string         `gorm:"type:text" json:"advanced"`     // JSON
	Status       string         `gorm:"size:20;default:active" json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// CacheTask 缓存刷新/预热任务
type CacheTask struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"index;not null" json:"user_id"`
	DomainID   uint           `gorm:"index;not null" json:"domain_id"`
	Type       string         `gorm:"size:20;not null" json:"type"` // file_refresh / dir_refresh / file_preheat
	URLs       string         `gorm:"type:text" json:"urls"`       // JSON array
	Status     string         `gorm:"size:20;default:pending" json:"status"` // pending / running / completed / failed / cancelled
	Progress   int            `gorm:"default:0" json:"progress"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// SystemSetting 系统设置
type SystemSetting struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"size:100;uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DDoSRule DDoS防护规则
type DDoSRule struct {
	ID                     uint           `gorm:"primaryKey" json:"id"`
	Name                   string         `gorm:"size:100;not null" json:"name"`
	Scope                  string         `gorm:"size:20;default:global" json:"scope"` // global / node
	NodeIDs                string         `gorm:"type:text" json:"node_ids"`           // JSON array
	MaxConnectionsPerIP    int            `gorm:"default:0" json:"max_connections_per_ip"`
	NewConnectionsPerSec   int            `gorm:"default:0" json:"new_connections_per_sec"`
	MaxPacketsPerSec       int            `gorm:"default:0" json:"max_packets_per_sec"`
	AutoBanEnabled         bool           `gorm:"default:false" json:"auto_ban_enabled"`
	BanThresholdConn       int            `gorm:"default:0" json:"ban_threshold_connections"`
	BanThresholdPackets    int            `gorm:"default:0" json:"ban_threshold_packets"`
	BanDurationSeconds     int            `gorm:"default:3600" json:"ban_duration_seconds"`
	Status                 string         `gorm:"size:20;default:active" json:"status"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`
}

// DDoSBlacklistEntry DDoS黑白名单
type DDoSBlacklistEntry struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	IP        string         `gorm:"size:45;not null" json:"ip"`
	CIDR      string         `gorm:"size:50" json:"cidr"`
	Type      string         `gorm:"size:20;not null" json:"type"` // black / white
	Reason    string         `gorm:"size:255" json:"reason"`
	AutoBan   bool           `gorm:"default:false" json:"auto_ban"`
	ExpiresAt *time.Time     `json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// LogServerConfig 日志服务器配置
type LogServerConfig struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Mode      string         `gorm:"size:20;default:master" json:"mode"` // master / log_server
	Address   string         `gorm:"size:255" json:"address"`
	Token     string         `gorm:"size:255" json:"-"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// AIConfig AI配置
type AIConfigModel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Provider  string         `gorm:"size:50;not null" json:"provider"`
	Model     string         `gorm:"size:100" json:"model"`
	APIKey    string         `gorm:"type:text" json:"-"`
	BaseURL   string         `gorm:"size:255" json:"base_url"`
	Enabled   bool           `gorm:"default:false" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Order 购买记录
type Order struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"index;not null" json:"user_id"`
	OrderNo     string         `gorm:"size:50;uniqueIndex;not null" json:"order_no"`
	ProductType string         `gorm:"size:20;not null" json:"product_type"` // package / traffic / domain
	ProductName string         `gorm:"size:100" json:"product_name"`
	Amount      float64        `gorm:"default:0" json:"amount"`
	Channel     string         `gorm:"size:50" json:"channel"`
	Status      string         `gorm:"size:20;default:pending" json:"status"` // pending / paid / failed / cancelled
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// OperationLog 操作日志
type OperationLog struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index" json:"user_id"`
	Action    string         `gorm:"size:100;not null" json:"action"`
	Target    string         `gorm:"size:255" json:"target"`
	Detail    string         `gorm:"type:text" json:"detail"`
	IP        string         `gorm:"size:45" json:"ip"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TrafficPackage 流量包
type TrafficPackage struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"size:100;not null" json:"name"`
	TrafficLimit  int64          `gorm:"not null" json:"traffic_limit"` // bytes
	Price         float64        `gorm:"default:0" json:"price"`
	Status        string         `gorm:"size:20;default:active" json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// DomainPackage 域名包
type DomainPackage struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	DomainLimit int            `gorm:"not null" json:"domain_limit"`
	Price       float64        `gorm:"default:0" json:"price"`
	Status      string         `gorm:"size:20;default:active" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// CertificateRequest 证书申请记录
type CertificateRequest struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      uint           `gorm:"index;not null" json:"user_id"`
	Domain      string         `gorm:"size:255;not null" json:"domain"`
	VerifyType  string         `gorm:"size:20" json:"verify_type"` // http / dns
	AcmeAccountID *uint        `gorm:"index" json:"acme_account_id"`
	Status      string         `gorm:"size:20;default:pending" json:"status"` // pending / succ / failed
	ErrorLog    string         `gorm:"type:text" json:"error_log"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
