package storage

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"
)

// LogQuerier ClickHouse 日志查询器
type LogQuerier struct {
	conn   driver.Conn
	logger *zap.Logger
}

// NewLogQuerier 创建查询器
func NewLogQuerier(conn driver.Conn, logger *zap.Logger) *LogQuerier {
	return &LogQuerier{conn: conn, logger: logger}
}

// ===================== 访问日志查询 =====================

// QueryAccessLogs 查询访问日志
// 支持：域名/IP/URL/方法/状态码/时间范围/边缘节点/分页
func (q *LogQuerier) QueryAccessLogs(ctx context.Context, req AccessLogQuery) (*LogListResult, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 1000 {
		req.PageSize = 50
	}

	// 默认时间范围：最近 24 小时
	startTime, endTime := q.ensureTimeRange(req.StartTime, req.EndTime)

	var (
		conds []string
		args  []interface{}
	)
	conds = append(conds, "time >= ?", "time <= ?")
	args = append(args, startTime, endTime)

	if req.Domain != "" {
		conds = append(conds, "domain = ?")
		args = append(args, req.Domain)
	}
	if req.ClientIP != "" {
		ipUint := ipv4ToUInt32(req.ClientIP)
		conds = append(conds, "client_ip = ?")
		args = append(args, ipUint)
	}
	if req.URL != "" {
		conds = append(conds, "url LIKE ?")
		args = append(args, "%"+req.URL+"%")
	}
	if req.Method != "" {
		conds = append(conds, "method = ?")
		args = append(args, strings.ToUpper(req.Method))
	}
	if req.Status != "" {
		if strings.HasSuffix(strings.ToLower(req.Status), "xx") {
			// 例如 "2xx" -> status >= 200 AND status < 300
			prefix := req.Status[:1]
			base, err := strconv.Atoi(prefix)
			if err == nil {
				conds = append(conds, "status >= ? AND status < ?")
				args = append(args, base*100, (base+1)*100)
			}
		} else {
			conds = append(conds, "status = ?")
			args = append(args, req.Status)
		}
	}
	if req.EdgeNode != "" {
		conds = append(conds, "edge_node = ?")
		args = append(args, req.EdgeNode)
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	// 总数
	var total uint64
	if err := q.conn.QueryRow(ctx, fmt.Sprintf("SELECT count() FROM %s %s", LogTypeAccess, where), args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count access_logs failed: %w", err)
	}

	// 数据
	querySQL := fmt.Sprintf(`
SELECT time, domain, IPv4NumToString(client_ip) AS client_ip, method, url, status, bytes, ua, referer,
       runtime, cache_status, edge_node, country, province, city, isp, attack_type, risk_score
FROM %s %s
ORDER BY time DESC
LIMIT %d OFFSET %d`,
		LogTypeAccess, where, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := q.conn.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("query access_logs failed: %w", err)
	}
	defer rows.Close()

	list := make([]map[string]interface{}, 0, req.PageSize)
	for rows.Next() {
		var (
			t                                  time.Time
			domain, clientIP, method, url, ua  string
			referer, cacheStatus, edgeNode     string
			country, province, city, isp       string
			attackType                         string
			status                             uint16
			bytesVal                           uint64
			runtime, riskScore                 float32
		)
		if err := rows.Scan(&t, &domain, &clientIP, &method, &url, &status, &bytesVal, &ua,
			&referer, &runtime, &cacheStatus, &edgeNode, &country, &province, &city, &isp,
			&attackType, &riskScore); err != nil {
			q.logger.Warn("scan access_log row failed", zap.Error(err))
			continue
		}
		list = append(list, map[string]interface{}{
			"time":         t.Format("2006-01-02 15:04:05"),
			"domain":       domain,
			"client_ip":    clientIP,
			"method":       method,
			"url":          url,
			"status":       status,
			"bytes":        bytesVal,
			"ua":           ua,
			"referer":      referer,
			"runtime":      runtime,
			"cache_status": cacheStatus,
			"edge_node":    edgeNode,
			"country":      country,
			"province":     province,
			"city":         city,
			"isp":          isp,
			"attack_type":  attackType,
			"risk_score":   riskScore,
		})
	}

	return &LogListResult{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// ===================== 攻击日志查询 =====================

// QueryAttackLogs 查询攻击日志
func (q *LogQuerier) QueryAttackLogs(ctx context.Context, req AttackLogQuery) (*LogListResult, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 1000 {
		req.PageSize = 50
	}

	startTime, endTime := q.ensureTimeRange(req.StartTime, req.EndTime)

	var (
		conds []string
		args  []interface{}
	)
	conds = append(conds, "time >= ?", "time <= ?")
	args = append(args, startTime, endTime)

	if req.Domain != "" {
		conds = append(conds, "domain = ?")
		args = append(args, req.Domain)
	}
	if req.ClientIP != "" {
		ipUint := ipv4ToUInt32(req.ClientIP)
		conds = append(conds, "client_ip = ?")
		args = append(args, ipUint)
	}
	if req.AttackType != "" {
		conds = append(conds, "attack_type = ?")
		args = append(args, req.AttackType)
	}
	if req.Action != "" {
		conds = append(conds, "action = ?")
		args = append(args, req.Action)
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total uint64
	if err := q.conn.QueryRow(ctx, fmt.Sprintf("SELECT count() FROM %s %s", LogTypeAttack, where), args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count attack_logs failed: %w", err)
	}

	querySQL := fmt.Sprintf(`
SELECT time, domain, IPv4NumToString(client_ip) AS client_ip, attack_type, rule_id, match_content, action, url, method
FROM %s %s
ORDER BY time DESC
LIMIT %d OFFSET %d`,
		LogTypeAttack, where, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := q.conn.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("query attack_logs failed: %w", err)
	}
	defer rows.Close()

	list := make([]map[string]interface{}, 0, req.PageSize)
	for rows.Next() {
		var (
			t                              time.Time
			domain, clientIP, attackType   string
			ruleID, matchContent, action   string
			url, method                    string
		)
		if err := rows.Scan(&t, &domain, &clientIP, &attackType, &ruleID, &matchContent, &action, &url, &method); err != nil {
			q.logger.Warn("scan attack_log row failed", zap.Error(err))
			continue
		}
		list = append(list, map[string]interface{}{
			"time":          t.Format("2006-01-02 15:04:05"),
			"domain":        domain,
			"client_ip":     clientIP,
			"attack_type":   attackType,
			"rule_id":       ruleID,
			"match_content": matchContent,
			"action":        action,
			"url":           url,
			"method":        method,
		})
	}

	return &LogListResult{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// ===================== 四层转发日志查询 =====================

// QueryLayer4Logs 查询四层转发日志
func (q *LogQuerier) QueryLayer4Logs(ctx context.Context, req Layer4LogQuery) (*LogListResult, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 1000 {
		req.PageSize = 50
	}

	startTime, endTime := q.ensureTimeRange(req.StartTime, req.EndTime)

	var (
		conds []string
		args  []interface{}
	)
	conds = append(conds, "time >= ?", "time <= ?")
	args = append(args, startTime, endTime)

	if req.ClientIP != "" {
		conds = append(conds, "client_ip = ?")
		args = append(args, ipv4ToUInt32(req.ClientIP))
	}
	if req.TargetIP != "" {
		conds = append(conds, "target_ip = ?")
		args = append(args, ipv4ToUInt32(req.TargetIP))
	}
	if req.ListenPort != "" {
		conds = append(conds, "listen_port = ?")
		args = append(args, parsePort(req.ListenPort))
	}
	if req.Protocol != "" {
		conds = append(conds, "protocol = ?")
		args = append(args, strings.ToLower(req.Protocol))
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total uint64
	if err := q.conn.QueryRow(ctx, fmt.Sprintf("SELECT count() FROM %s %s", LogTypeLayer4, where), args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count layer4_logs failed: %w", err)
	}

	querySQL := fmt.Sprintf(`
SELECT time, IPv4NumToString(client_ip) AS client_ip, IPv4NumToString(target_ip) AS target_ip,
       listen_port, protocol, bytes_in, bytes_out, conn_duration
FROM %s %s
ORDER BY time DESC
LIMIT %d OFFSET %d`,
		LogTypeLayer4, where, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := q.conn.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("query layer4_logs failed: %w", err)
	}
	defer rows.Close()

	list := make([]map[string]interface{}, 0, req.PageSize)
	for rows.Next() {
		var (
			t                            time.Time
			clientIP, targetIP, protocol string
			listenPort                   uint16
			bytesIn, bytesOut            uint64
			connDuration                 float32
		)
		if err := rows.Scan(&t, &clientIP, &targetIP, &listenPort, &protocol, &bytesIn, &bytesOut, &connDuration); err != nil {
			q.logger.Warn("scan layer4_log row failed", zap.Error(err))
			continue
		}
		list = append(list, map[string]interface{}{
			"time":          t.Format("2006-01-02 15:04:05"),
			"client_ip":     clientIP,
			"target_ip":     targetIP,
			"listen_port":   listenPort,
			"protocol":      protocol,
			"bytes_in":      bytesIn,
			"bytes_out":     bytesOut,
			"conn_duration": connDuration,
		})
	}

	return &LogListResult{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// ===================== 四层拦截日志查询 =====================

// QueryLayer4InterceptLogs 查询四层拦截日志
func (q *LogQuerier) QueryLayer4InterceptLogs(ctx context.Context, req Layer4InterceptLogQuery) (*LogListResult, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 1000 {
		req.PageSize = 50
	}

	startTime, endTime := q.ensureTimeRange(req.StartTime, req.EndTime)

	var (
		conds []string
		args  []interface{}
	)
	conds = append(conds, "time >= ?", "time <= ?")
	args = append(args, startTime, endTime)

	if req.ClientIP != "" {
		conds = append(conds, "client_ip = ?")
		args = append(args, ipv4ToUInt32(req.ClientIP))
	}
	if req.TargetIP != "" {
		conds = append(conds, "target_ip = ?")
		args = append(args, ipv4ToUInt32(req.TargetIP))
	}
	if req.ListenPort != "" {
		conds = append(conds, "listen_port = ?")
		args = append(args, parsePort(req.ListenPort))
	}
	if req.Protocol != "" {
		conds = append(conds, "protocol = ?")
		args = append(args, strings.ToLower(req.Protocol))
	}
	if req.Reason != "" {
		conds = append(conds, "reason LIKE ?")
		args = append(args, "%"+req.Reason+"%")
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total uint64
	if err := q.conn.QueryRow(ctx, fmt.Sprintf("SELECT count() FROM %s %s", LogTypeLayer4Intercept, where), args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count layer4_intercept_logs failed: %w", err)
	}

	querySQL := fmt.Sprintf(`
SELECT time, IPv4NumToString(client_ip) AS client_ip, IPv4NumToString(target_ip) AS target_ip,
       listen_port, protocol, reason, action
FROM %s %s
ORDER BY time DESC
LIMIT %d OFFSET %d`,
		LogTypeLayer4Intercept, where, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := q.conn.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("query layer4_intercept_logs failed: %w", err)
	}
	defer rows.Close()

	list := make([]map[string]interface{}, 0, req.PageSize)
	for rows.Next() {
		var (
			t                        time.Time
			clientIP, targetIP       string
			protocol, reason, action string
			listenPort               uint16
		)
		if err := rows.Scan(&t, &clientIP, &targetIP, &listenPort, &protocol, &reason, &action); err != nil {
			q.logger.Warn("scan layer4_intercept_log row failed", zap.Error(err))
			continue
		}
		list = append(list, map[string]interface{}{
			"time":        t.Format("2006-01-02 15:04:05"),
			"client_ip":   clientIP,
			"target_ip":   targetIP,
			"listen_port": listenPort,
			"protocol":    protocol,
			"reason":      reason,
			"action":      action,
		})
	}

	return &LogListResult{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// ===================== AI 日志查询 =====================

// QueryAILogs 查询 AI 调用日志
func (q *LogQuerier) QueryAILogs(ctx context.Context, req AILogQuery) (*LogListResult, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 1000 {
		req.PageSize = 50
	}

	startTime, endTime := q.ensureTimeRange(req.StartTime, req.EndTime)

	var (
		conds []string
		args  []interface{}
	)
	conds = append(conds, "time >= ?", "time <= ?")
	args = append(args, startTime, endTime)

	if req.Domain != "" {
		conds = append(conds, "domain = ?")
		args = append(args, req.Domain)
	}
	if req.ClientIP != "" {
		conds = append(conds, "client_ip = ?")
		args = append(args, ipv4ToUInt32(req.ClientIP))
	}
	if req.AIType != "" {
		conds = append(conds, "ai_type = ?")
		args = append(args, req.AIType)
	}
	if req.Model != "" {
		conds = append(conds, "model = ?")
		args = append(args, req.Model)
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total uint64
	if err := q.conn.QueryRow(ctx, fmt.Sprintf("SELECT count() FROM %s %s", LogTypeAI, where), args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count ai_logs failed: %w", err)
	}

	querySQL := fmt.Sprintf(`
SELECT time, domain, IPv4NumToString(client_ip) AS client_ip, ai_type, input_text, output_text,
       risk_score, model, tokens
FROM %s %s
ORDER BY time DESC
LIMIT %d OFFSET %d`,
		LogTypeAI, where, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := q.conn.Query(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("query ai_logs failed: %w", err)
	}
	defer rows.Close()

	list := make([]map[string]interface{}, 0, req.PageSize)
	for rows.Next() {
		var (
			t                          time.Time
			domain, clientIP, aiType   string
			inputText, outputText      string
			model                      string
			riskScore                  float32
			tokens                     uint32
		)
		if err := rows.Scan(&t, &domain, &clientIP, &aiType, &inputText, &outputText, &riskScore, &model, &tokens); err != nil {
			q.logger.Warn("scan ai_log row failed", zap.Error(err))
			continue
		}
		list = append(list, map[string]interface{}{
			"time":        t.Format("2006-01-02 15:04:05"),
			"domain":      domain,
			"client_ip":   clientIP,
			"ai_type":     aiType,
			"input_text":  inputText,
			"output_text": outputText,
			"risk_score":  riskScore,
			"model":       model,
			"tokens":      tokens,
		})
	}

	return &LogListResult{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// ===================== 流量统计 =====================

// QueryTrafficStats 查询流量统计
// 返回：总请求数、总流量、带宽(bps)、缓存命中率
func (q *LogQuerier) QueryTrafficStats(ctx context.Context, req TrafficStatsQuery) (*TrafficStatsResult, error) {
	startTime, endTime := q.ensureTimeRange(req.StartTime, req.EndTime)

	var (
		conds []string
		args  []interface{}
	)
	conds = append(conds, "time >= ?", "time <= ?")
	args = append(args, startTime, endTime)

	if req.Domain != "" {
		conds = append(conds, "domain = ?")
		args = append(args, req.Domain)
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	// 计算时间范围秒数（用于估算带宽：总字节 * 8 / 时间范围秒数）
	durSec := endTime.Sub(startTime).Seconds()
	if durSec <= 0 {
		durSec = 1
	}

	var (
		totalReq  uint64
		totalByte uint64
		cacheHit  uint64
	)
	sql := fmt.Sprintf(`
SELECT count(), sum(bytes), countIf(cache_status = 'HIT')
FROM %s %s`, LogTypeAccess, where)
	if err := q.conn.QueryRow(ctx, sql, args...).Scan(&totalReq, &totalByte, &cacheHit); err != nil {
		return nil, fmt.Errorf("query traffic stats failed: %w", err)
	}

	var hitRate float64
	if totalReq > 0 {
		hitRate = float64(cacheHit) / float64(totalReq)
	}

	return &TrafficStatsResult{
		TotalRequests:     totalReq,
		TotalBytes:        totalByte,
		TotalBandwidthBps: uint64(float64(totalByte) * 8 / durSec),
		CacheHitRate:      hitRate,
	}, nil
}

// ===================== Top N 排行 =====================

// QueryTopN 指标排行：Top IP / URL / UA / 状态码
// field 取值: ip / url / ua / status
func (q *LogQuerier) QueryTopN(ctx context.Context, req TopNQuery) (*TopNResult, error) {
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 10
	}

	startTime, endTime := q.ensureTimeRange(req.StartTime, req.EndTime)

	var (
		conds []string
		args  []interface{}
	)
	conds = append(conds, "time >= ?", "time <= ?")
	args = append(args, startTime, endTime)

	if req.Domain != "" {
		conds = append(conds, "domain = ?")
		args = append(args, req.Domain)
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var selExpr string
	switch strings.ToLower(req.Field) {
	case "ip", "client_ip":
		selExpr = "IPv4NumToString(client_ip)"
	case "url":
		selExpr = "url"
	case "ua", "user_agent":
		selExpr = "ua"
	case "status", "status_code":
		selExpr = "toString(status)"
	case "country":
		selExpr = "country"
	case "isp":
		selExpr = "isp"
	default:
		selExpr = "IPv4NumToString(client_ip)"
		req.Field = "ip"
	}

	sql := fmt.Sprintf(`
SELECT %s AS v, count() AS c
FROM %s %s
GROUP BY v
ORDER BY c DESC
LIMIT %d`,
		selExpr, LogTypeAccess, where, req.Limit)

	rows, err := q.conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query top n failed: %w", err)
	}
	defer rows.Close()

	items := make([]TopNItem, 0, req.Limit)
	for rows.Next() {
		var v string
		var c uint64
		if err := rows.Scan(&v, &c); err != nil {
			q.logger.Warn("scan top_n row failed", zap.Error(err))
			continue
		}
		items = append(items, TopNItem{Value: v, Count: c})
	}

	return &TopNResult{
		Field: req.Field,
		Items: items,
	}, nil
}

// ===================== 日志地图（地理位置聚合） =====================

// QueryGeoMap 按地理位置聚合访问日志
func (q *LogQuerier) QueryGeoMap(ctx context.Context, req TrafficStatsQuery) (*GeoMapResult, error) {
	startTime, endTime := q.ensureTimeRange(req.StartTime, req.EndTime)

	var (
		conds []string
		args  []interface{}
	)
	conds = append(conds, "time >= ?", "time <= ?")
	args = append(args, startTime, endTime)

	if req.Domain != "" {
		conds = append(conds, "domain = ?")
		args = append(args, req.Domain)
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	sql := fmt.Sprintf(`
SELECT country, province, city, count(), sum(bytes)
FROM %s %s
GROUP BY country, province, city
ORDER BY count() DESC
LIMIT 1000`,
		LogTypeAccess, where)

	rows, err := q.conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query geo map failed: %w", err)
	}
	defer rows.Close()

	items := make([]GeoMapItem, 0, 64)
	for rows.Next() {
		var (
			country, province, city string
			count                   uint64
			bytes                   uint64
		)
		if err := rows.Scan(&country, &province, &city, &count, &bytes); err != nil {
			q.logger.Warn("scan geo row failed", zap.Error(err))
			continue
		}
		items = append(items, GeoMapItem{
			Country:  country,
			Province: province,
			City:     city,
			Count:    count,
			Bytes:    bytes,
		})
	}

	return &GeoMapResult{Items: items}, nil
}

// ===================== 日志导出 (CSV / JSON) =====================

// ExportAccessLogs 导出访问日志
// format: "csv" 或 "json"
func (q *LogQuerier) ExportAccessLogs(ctx context.Context, req AccessLogQuery, format string) (*LogExportResult, error) {
	// 不分页，导出最多 10000 条
	req.Page = 1
	req.PageSize = 10000

	result, err := q.QueryAccessLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(format) {
	case "csv":
		data, err := exportListAsCSV(result.List)
		if err != nil {
			return nil, err
		}
		return &LogExportResult{Format: "csv", Data: data}, nil
	case "json":
		data, err := json.MarshalIndent(result.List, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal json failed: %w", err)
		}
		return &LogExportResult{Format: "json", Data: data}, nil
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportListAsCSV 将 []map[string]interface{} 导出为 CSV
func exportListAsCSV(list []map[string]interface{}) ([]byte, error) {
	if len(list) == 0 {
		return []byte{}, nil
	}

	// 收集所有 key（保持稳定顺序：使用第一条记录的 key 顺序）
	headers := make([]string, 0, len(list[0]))
	seen := make(map[string]struct{})
	for _, row := range list {
		for k := range row {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				headers = append(headers, k)
			}
		}
	}

	var sb strings.Builder
	w := csv.NewWriter(&sb)

	if err := w.Write(headers); err != nil {
		return nil, fmt.Errorf("write csv header failed: %w", err)
	}

	for _, row := range list {
		rec := make([]string, len(headers))
		for i, h := range headers {
			v, ok := row[h]
			if !ok || v == nil {
				rec[i] = ""
				continue
			}
			rec[i] = fmt.Sprintf("%v", v)
		}
		if err := w.Write(rec); err != nil {
			return nil, fmt.Errorf("write csv row failed: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flush csv failed: %w", err)
	}

	return []byte(sb.String()), nil
}

// ===================== 辅助函数 =====================

// ensureTimeRange 确保查询有起止时间，缺省返回最近 24 小时
func (q *LogQuerier) ensureTimeRange(start, end string) (time.Time, time.Time) {
	layout := "2006-01-02 15:04:05"

	var (
		startT = time.Now().Add(-24 * time.Hour)
		endT   = time.Now()
		err    error
	)

	if start != "" {
		if startT, err = time.ParseInLocation(layout, start, time.Local); err != nil {
			q.logger.Warn("parse start_time failed, using default", zap.String("start", start), zap.Error(err))
			startT = time.Now().Add(-24 * time.Hour)
		}
	}
	if end != "" {
		if endT, err = time.ParseInLocation(layout, end, time.Local); err != nil {
			q.logger.Warn("parse end_time failed, using default", zap.String("end", end), zap.Error(err))
			endT = time.Now()
		}
	}

	// 容错：start > end 时交换
	if startT.After(endT) {
		startT, endT = endT, startT
	}

	return startT, endT
}

// ipToNum 旧别名（保留以防外部引用）—— 等价于 ipv4ToUInt32
func ipToNum(ip string) uint32 {
	return ipv4ToUInt32(ip)
}

// ipFromNum 将 ClickHouse IPv4 (UInt32) 转回字符串（仅用于反向兼容工具）
func ipFromNum(n uint32) string {
	return net.IPv4(
		byte(n>>24),
		byte(n>>16),
		byte(n>>8),
		byte(n),
	).String()
}
