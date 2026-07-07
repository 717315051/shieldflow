package grpc

import (
	"encoding/json"
	"time"
)

// ============================================================================
// 消息类型定义（代替 proto 生成的 .pb.go）
// 所有结构体均使用 JSON tag，便于 HTTP REST 序列化。
// ============================================================================

// --- 通用响应 ---

type Response struct {
	Code    int         `json:"code"`     // 0=成功, 非0=错误
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func NewOKResponse(data interface{}) Response {
	return Response{Code: 0, Message: "ok", Data: data}
}

func NewErrorResponse(code int, msg string) Response {
	return Response{Code: code, Message: msg}
}

// --- Config Service ---

type PushDomainConfigRequest struct {
	NodeID   uint            `json:"node_id"`
	DomainID uint            `json:"domain_id"`
	Config   json.RawMessage `json:"config"` // 域名配置 JSON
}

type PushDDoSConfigRequest struct {
	NodeID uint            `json:"node_id"`
	Config json.RawMessage `json:"config"` // DDoS 配置 JSON
}

type PushGlobalConfigRequest struct {
	NodeIDs []uint          `json:"node_ids"`
	Config  json.RawMessage `json:"config"` // 全局配置 JSON
}

type SyncNodeStatusRequest struct {
	NodeID uint            `json:"node_id"`
	Status json.RawMessage `json:"status"` // 节点状态 JSON
}

type HeartbeatRequest struct {
	NodeID  uint            `json:"node_id"`
	Metrics json.RawMessage `json:"metrics"` // 指标 JSON
}

type HeartbeatResponse struct {
	ACK      bool   `json:"ack"`
	Timestamp int64 `json:"timestamp"`
	ConfigVersion string `json:"config_version"`
}

// --- Log Service ---

type AccessLogEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	NodeID       uint      `json:"node_id"`
	DomainID     uint      `json:"domain_id"`
	ClientIP     string    `json:"client_ip"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	Status       int       `json:"status"`
	BytesSent    int64     `json:"bytes_sent"`
	BytesRecv    int64     `json:"bytes_recv"`
	UserAgent    string    `json:"user_agent"`
	Referer      string    `json:"referer"`
	CacheStatus  string    `json:"cache_status"`
	ResponseTime float64   `json:"response_time"` // 秒
	Upstream     string    `json:"upstream"`
}

type AccessLogBatch struct {
	NodeID uint             `json:"node_id"`
	Logs   []AccessLogEntry `json:"logs"`
}

type AttackLogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	NodeID      uint      `json:"node_id"`
	DomainID    uint      `json:"domain_id"`
	ClientIP    string    `json:"client_ip"`
	AttackType  string    `json:"attack_type"` // sql_injection / xss / cc / scanner ...
	RuleID      string    `json:"rule_id"`
	Action      string    `json:"action"` // block / captcha / log
	Request     string    `json:"request"`
	MatchedData string    `json:"matched_data"`
}

type AttackLogBatch struct {
	NodeID uint             `json:"node_id"`
	Logs   []AttackLogEntry `json:"logs"`
}

type DDoSLogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	NodeID      uint      `json:"node_id"`
	ClientIP    string    `json:"client_ip"`
	Protocol    string    `json:"protocol"`
	SrcPort     int       `json:"src_port"`
	DstPort     int       `json:"dst_port"`
	PacketRate  int64     `json:"packet_rate"`
	ByteRate    int64     `json:"byte_rate"`
	Action      string    `json:"action"`
	RuleID      string    `json:"rule_id"`
}

type DDoSLogBatch struct {
	NodeID uint           `json:"node_id"`
	Logs   []DDoSLogEntry `json:"logs"`
}

type Layer4LogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	NodeID     uint      `json:"node_id"`
	Protocol   string    `json:"protocol"`
	ListenPort int       `json:"listen_port"`
	ClientIP   string    `json:"client_ip"`
	BackendIP  string    `json:"backend_ip"`
	BytesIn    int64     `json:"bytes_in"`
	BytesOut   int64     `json:"bytes_out"`
	ConnID     string    `json:"conn_id"`
	Duration   float64   `json:"duration"`
}

type Layer4LogBatch struct {
	NodeID uint             `json:"node_id"`
	Logs   []Layer4LogEntry `json:"logs"`
}

type AILogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	NodeID     uint      `json:"node_id"`
	DomainID   uint      `json:"domain_id"`
	ClientIP   string    `json:"client_ip"`
	Model      string    `json:"model"`
	Score      float64   `json:"score"`     // 异常分数 0-1
	Verdict    string    `json:"verdict"`   // normal / suspicious / malicious
	Reason     string    `json:"reason"`
	Request    string    `json:"request"`
}

type AILogBatch struct {
	NodeID uint         `json:"node_id"`
	Logs   []AILogEntry `json:"logs"`
}

// --- Node Service ---

type RegisterNodeRequest struct {
	NodeID     uint   `json:"node_id"`
	LicenseKey string `json:"license_key"`
	Info       NodeInfo `json:"info"`
}

type NodeInfo struct {
	Name        string `json:"name"`
	IP          string `json:"ip"`
	Region      string `json:"region"`
	Version     string `json:"version"`
	CPU         int    `json:"cpu"`
	Memory      int    `json:"memory"`
	Disk        int    `json:"disk"`
	Bandwidth   int    `json:"bandwidth"`
	GRPCAddress string `json:"grpc_address"`
}

type RegisterNodeResponse struct {
	NodeID  uint            `json:"node_id"`
	Success bool            `json:"success"`
	Config  json.RawMessage `json:"config,omitempty"` // 节点应加载的配置
}

type UpdateNodeStatusRequest struct {
	NodeID    uint   `json:"node_id"`
	Status    string `json:"status"` // online / offline / maintenance
	ConnCount int    `json:"conn_count"`
	QPS       int    `json:"qps"`
}

type GetNodeConfigResponse struct {
	NodeID      uint            `json:"node_id"`
	Domains     json.RawMessage `json:"domains"`
	DDoSRules   json.RawMessage `json:"ddos_rules"`
	Certificates json.RawMessage `json:"certificates"`
	ConfigVersion string        `json:"config_version"`
}

// --- Cache Service ---

type PurgeCacheRequest struct {
	NodeID   uint     `json:"node_id"`
	DomainID uint     `json:"domain_id"`
	URLs     []string `json:"urls"`
	Recursive bool    `json:"recursive"` // 是否递归刷新目录
}

type PreheatCacheRequest struct {
	NodeID   uint     `json:"node_id"`
	DomainID uint     `json:"domain_id"`
	URLs     []string `json:"urls"`
}

type CacheStatsResponse struct {
	DomainID    uint   `json:"domain_id"`
	CacheSize   int64  `json:"cache_size"`    // bytes
	CacheItems  int64  `json:"cache_items"`
	HitRate     float64 `json:"hit_rate"`
	MissRate    float64 `json:"miss_rate"`
	Evictions   int64  `json:"evictions"`
}

// --- Auth Service ---

type VerifyLicenseRequest struct {
	LicenseKey string `json:"license_key"`
	NodeID     uint   `json:"node_id"`
	NodeIP     string `json:"node_ip"`
}

type VerifyLicenseResponse struct {
	Valid     bool   `json:"valid"`
	ExpiresAt string `json:"expires_at,omitempty"`
	NodeID    uint   `json:"node_id,omitempty"`
	Message   string `json:"message"`
}
