package storage

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ===================== ClickHouse 日志数据结构 =====================
//
// 这些结构对应独立的日志服务器 (log-server) 使用的扩展 ClickHouse 表结构，
// 与 sql/002_init_clickhouse.sql 中的基础表相比，列更丰富（地理位置、风险评分等）。
// 建议通过 log-server 自带的 schema 初始化脚本创建对应的扩展表。
//
// 所有结构均实现 LogEntry 接口（LogType + AppendToBatch），可被 LogWriter 批量写入。

// AccessLog 访问日志（七层 HTTP/HTTPS）
type AccessLog struct {
	Time        time.Time `json:"time"`
	Domain      string    `json:"domain"`
	ClientIP    string    `json:"client_ip"`
	Method      string    `json:"method"`
	URL         string    `json:"url"`
	Status      uint16    `json:"status"`
	Bytes       uint64    `json:"bytes"`
	UA          string    `json:"ua"`
	Referer     string    `json:"referer"`
	Runtime     float32   `json:"runtime"`      // 响应时间（毫秒）
	CacheStatus string    `json:"cache_status"` // HIT/MISS/BYPASS/...
	EdgeNode    string    `json:"edge_node"`
	Country     string    `json:"country"`
	Province    string    `json:"province"`
	City        string    `json:"city"`
	ISP         string    `json:"isp"`
	AttackType  string    `json:"attack_type"`
	RiskScore   float32   `json:"risk_score"`
}

// LogType 实现 LogEntry
func (l AccessLog) LogType() string { return LogTypeAccess }

// AppendToBatch 将单条访问日志追加到 ClickHouse batch
// 列顺序需与 access_logs 扩展表一致：
//
//	time, domain, client_ip, method, url, status, bytes, ua, referer,
//	runtime, cache_status, edge_node, country, province, city, isp,
//	attack_type, risk_score
func (l AccessLog) AppendToBatch(ctx context.Context, batch driver.Batch) error {
	return batch.Append(
		l.Time,
		ensureStr(l.Domain),
		ipv4ToUInt32(l.ClientIP),
		ensureStr(l.Method),
		ensureStr(l.URL),
		l.Status,
		l.Bytes,
		ensureStr(l.UA),
		ensureStr(l.Referer),
		l.Runtime,
		ensureStr(l.CacheStatus),
		ensureStr(l.EdgeNode),
		ensureStr(l.Country),
		ensureStr(l.Province),
		ensureStr(l.City),
		ensureStr(l.ISP),
		ensureStr(l.AttackType),
		l.RiskScore,
	)
}

// AttackLog 攻击日志（WAF/CC/Bot）
type AttackLog struct {
	Time         time.Time `json:"time"`
	Domain       string    `json:"domain"`
	ClientIP     string    `json:"client_ip"`
	AttackType   string    `json:"attack_type"`
	RuleID       string    `json:"rule_id"`
	MatchContent string    `json:"match_content"`
	Action       string    `json:"action"`
	URL          string    `json:"url"`
	Method       string    `json:"method"`
}

// LogType 实现 LogEntry
func (l AttackLog) LogType() string { return LogTypeAttack }

// AppendToBatch 列顺序：
//
//	time, domain, client_ip, attack_type, rule_id, match_content, action, url, method
func (l AttackLog) AppendToBatch(ctx context.Context, batch driver.Batch) error {
	return batch.Append(
		l.Time,
		ensureStr(l.Domain),
		ipv4ToUInt32(l.ClientIP),
		ensureStr(l.AttackType),
		ensureStr(l.RuleID),
		ensureStr(l.MatchContent),
		ensureStr(l.Action),
		ensureStr(l.URL),
		ensureStr(l.Method),
	)
}

// DDoSLog DDoS 防护日志
type DDoSLog struct {
	Time        time.Time `json:"time"`
	ClientIP    string    `json:"client_ip"`
	TargetIP    string    `json:"target_ip"`
	TargetPort  uint16    `json:"target_port"`
	Protocol    string    `json:"protocol"`
	PacketCount uint64    `json:"packet_count"`
	ByteCount   uint64    `json:"byte_count"`
	Action      string    `json:"action"`
}

// LogType 实现 LogEntry
func (l DDoSLog) LogType() string { return LogTypeDDoS }

// AppendToBatch 列顺序：
//
//	time, client_ip, target_ip, target_port, protocol, packet_count, byte_count, action
func (l DDoSLog) AppendToBatch(ctx context.Context, batch driver.Batch) error {
	return batch.Append(
		l.Time,
		ipv4ToUInt32(l.ClientIP),
		ipv4ToUInt32(l.TargetIP),
		l.TargetPort,
		ensureStr(l.Protocol),
		l.PacketCount,
		l.ByteCount,
		ensureStr(l.Action),
	)
}

// Layer4Log 四层转发日志
type Layer4Log struct {
	Time         time.Time `json:"time"`
	ClientIP     string    `json:"client_ip"`
	TargetIP     string    `json:"target_ip"`
	ListenPort   uint16    `json:"listen_port"`
	Protocol     string    `json:"protocol"`
	BytesIn      uint64    `json:"bytes_in"`
	BytesOut     uint64    `json:"bytes_out"`
	ConnDuration float32   `json:"conn_duration"` // 连接时长（秒）
}

// LogType 实现 LogEntry
func (l Layer4Log) LogType() string { return LogTypeLayer4 }

// AppendToBatch 列顺序：
//
//	time, client_ip, target_ip, listen_port, protocol, bytes_in, bytes_out, conn_duration
func (l Layer4Log) AppendToBatch(ctx context.Context, batch driver.Batch) error {
	return batch.Append(
		l.Time,
		ipv4ToUInt32(l.ClientIP),
		ipv4ToUInt32(l.TargetIP),
		l.ListenPort,
		ensureStr(l.Protocol),
		l.BytesIn,
		l.BytesOut,
		l.ConnDuration,
	)
}

// Layer4InterceptLog 四层拦截日志
type Layer4InterceptLog struct {
	Time       time.Time `json:"time"`
	ClientIP   string    `json:"client_ip"`
	TargetIP   string    `json:"target_ip"`
	ListenPort uint16    `json:"listen_port"`
	Protocol   string    `json:"protocol"`
	Reason     string    `json:"reason"`
	Action     string    `json:"action"`
}

// LogType 实现 LogEntry
func (l Layer4InterceptLog) LogType() string { return LogTypeLayer4Intercept }

// AppendToBatch 列顺序：
//
//	time, client_ip, target_ip, listen_port, protocol, reason, action
func (l Layer4InterceptLog) AppendToBatch(ctx context.Context, batch driver.Batch) error {
	return batch.Append(
		l.Time,
		ipv4ToUInt32(l.ClientIP),
		ipv4ToUInt32(l.TargetIP),
		l.ListenPort,
		ensureStr(l.Protocol),
		ensureStr(l.Reason),
		ensureStr(l.Action),
	)
}

// AILog AI 调用日志
type AILog struct {
	Time       time.Time `json:"time"`
	Domain     string    `json:"domain"`
	ClientIP   string    `json:"client_ip"`
	AIType     string    `json:"ai_type"`
	InputText  string    `json:"input_text"`
	OutputText string    `json:"output_text"`
	RiskScore  float32   `json:"risk_score"`
	Model      string    `json:"model"`
	Tokens     uint32    `json:"tokens"`
}

// LogType 实现 LogEntry
func (l AILog) LogType() string { return LogTypeAI }

// AppendToBatch 列顺序：
//
//	time, domain, client_ip, ai_type, input_text, output_text, risk_score, model, tokens
func (l AILog) AppendToBatch(ctx context.Context, batch driver.Batch) error {
	return batch.Append(
		l.Time,
		ensureStr(l.Domain),
		ipv4ToUInt32(l.ClientIP),
		ensureStr(l.AIType),
		ensureStr(l.InputText),
		ensureStr(l.OutputText),
		l.RiskScore,
		ensureStr(l.Model),
		l.Tokens,
	)
}

// ===================== 通用查询请求/响应结构 =====================

// AccessLogQuery 访问日志查询参数
type AccessLogQuery struct {
	Domain    string
	ClientIP  string
	URL       string
	Method    string
	Status    string // 状态码（支持 "200" 或 "2xx"）
	StartTime string // "2006-01-02 15:04:05"
	EndTime   string
	EdgeNode  string
	Page      int
	PageSize  int
}

// AttackLogQuery 攻击日志查询参数
type AttackLogQuery struct {
	Domain      string
	ClientIP    string
	AttackType  string
	Action      string
	StartTime   string
	EndTime     string
	Page        int
	PageSize    int
}

// Layer4LogQuery 四层转发日志查询参数
type Layer4LogQuery struct {
	ClientIP   string
	TargetIP   string
	ListenPort string
	Protocol   string
	StartTime  string
	EndTime    string
	Page       int
	PageSize   int
}

// Layer4InterceptLogQuery 四层拦截日志查询参数
type Layer4InterceptLogQuery struct {
	ClientIP   string
	TargetIP   string
	ListenPort string
	Protocol   string
	Reason     string
	StartTime  string
	EndTime    string
	Page       int
	PageSize   int
}

// AILogQuery AI 日志查询参数
type AILogQuery struct {
	Domain    string
	ClientIP  string
	AIType    string
	Model     string
	StartTime string
	EndTime   string
	Page      int
	PageSize  int
}

// LogListResult 日志列表查询结果
type LogListResult struct {
	List     []map[string]interface{} `json:"list"`
	Total    uint64                   `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}

// TrafficStatsQuery 流量统计查询参数
type TrafficStatsQuery struct {
	Domain    string
	StartTime string
	EndTime   string
}

// TrafficStatsResult 流量统计结果
type TrafficStatsResult struct {
	TotalRequests    uint64  `json:"total_requests"`
	TotalBytes       uint64  `json:"total_bytes"`
	TotalBandwidthBps uint64 `json:"total_bandwidth_bps"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
}

// TopNQuery Top N 排行查询参数
type TopNQuery struct {
	Domain    string
	Field     string // ip / url / ua / status
	StartTime string
	EndTime   string
	Limit     int
}

// TopNResult Top N 排行结果
type TopNResult struct {
	Field string            `json:"field"`
	Items []TopNItem        `json:"items"`
}

// TopNItem 单条排行项
type TopNItem struct {
	Value string `json:"value"`
	Count uint64 `json:"count"`
}

// GeoMapResult 日志地图（地理位置聚合）结果
type GeoMapResult struct {
	Items []GeoMapItem `json:"items"`
}

// GeoMapItem 地理位置聚合项
type GeoMapItem struct {
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	Count    uint64 `json:"count"`
	Bytes    uint64 `json:"bytes"`
}

// LogExportResult 日志导出结果
type LogExportResult struct {
	Format string `json:"format"` // csv / json
	Data   []byte `json:"data"`
}
