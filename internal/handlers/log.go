package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/models"
	"gorm.io/gorm"
)

// ============ 日志查询 (ClickHouse) ============
//
// 约定的 ClickHouse 表结构 (供查询参考):
//   access_logs:       domain, ip, url, method, status, bytes, user_agent, referer, country, city, timestamp
//   attack_logs:       domain, ip, url, attack_type, rule_id, action, timestamp
//   layer4_logs:       listen_port, protocol, src_ip, dst_ip, bytes_in, bytes_out, timestamp
//   layer4_intercept_logs: listen_port, src_ip, attack_type, action, timestamp
//   ai_logs:           domain, ip, feature, model, score, action, timestamp

// queryAccessLogs 查询访问日志的公共逻辑
func queryAccessLogs(c *gin.Context, ch driver.Conn, table string) {
	userID := c.MustGet("user_id").(uint)
	db := c.MustGet("db").(*gorm.DB)

	// 获取用户域名列表用于权限过滤
	var domains []string
	db.Model(&models.Domain{}).Where("user_id = ?", userID).Pluck("domain_name", &domains)
	if len(domains) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	// 查询参数
	domain := c.Query("domain")
	ip := c.Query("ip")
	url := c.Query("url")
	statusCode := c.Query("status")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	page, pageSize := parsePagination(c)

	// 构造 WHERE
	where := "WHERE domain IN (?)"
	args := []interface{}{domains}
	if domain != "" {
		where += " AND domain = ?"
		args = append(args, domain)
	}
	if ip != "" {
		where += " AND ip = ?"
		args = append(args, ip)
	}
	if url != "" {
		where += " AND url LIKE ?"
		args = append(args, "%"+url+"%")
	}
	if statusCode != "" {
		where += " AND status = ?"
		args = append(args, statusCode)
	}
	if startTime != "" {
		where += " AND timestamp >= ?"
		args = append(args, startTime)
	}
	if endTime != "" {
		where += " AND timestamp <= ?"
		args = append(args, endTime)
	}

	// 默认时间范围: 最近 24 小时
	if startTime == "" && endTime == "" {
		where += " AND timestamp >= ?"
		args = append(args, time.Now().Add(-24*time.Hour).Format("2006-01-02 15:04:05"))
	}

	// 查询总数
	countSQL := fmt.Sprintf("SELECT count() FROM %s %s", table, where)
	var total uint64
	if err := ch.QueryRow(context.Background(), countSQL, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}

	// 查询数据
	querySQL := fmt.Sprintf("SELECT domain, ip, url, method, status, bytes, user_agent, referer, country, city, timestamp FROM %s %s ORDER BY timestamp DESC LIMIT %d OFFSET %d", table, where, pageSize, (page-1)*pageSize)
	rows, err := ch.Query(context.Background(), querySQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var d, ipAddr, urlStr, method, ua, ref, country, city, ts string
		var status, bytesVal int
		if err := rows.Scan(&d, &ipAddr, &urlStr, &method, &status, &bytesVal, &ua, &ref, &country, &city, &ts); err != nil {
			continue
		}
		list = append(list, gin.H{
			"domain":     d,
			"ip":         ipAddr,
			"url":        urlStr,
			"method":     method,
			"status":     status,
			"bytes":      bytesVal,
			"user_agent": ua,
			"referer":    ref,
			"country":    country,
			"city":       city,
			"timestamp":  ts,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetAccessLogs 访问日志
// GET /api/v1/logs/access
func GetAccessLogs(c *gin.Context) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "日志服务不可用"})
		return
	}
	queryAccessLogs(c, ch, "access_logs")
}

// GetAttackLogs 拦截日志
// GET /api/v1/logs/attack
func GetAttackLogs(c *gin.Context) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "日志服务不可用"})
		return
	}

	userID := c.MustGet("user_id").(uint)
	db := c.MustGet("db").(*gorm.DB)

	var domains []string
	db.Model(&models.Domain{}).Where("user_id = ?", userID).Pluck("domain_name", &domains)
	if len(domains) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	attackType := c.Query("attack_type")
	domain := c.Query("domain")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	page, pageSize := parsePagination(c)

	where := "WHERE domain IN (?)"
	args := []interface{}{domains}
	if attackType != "" {
		where += " AND attack_type = ?"
		args = append(args, attackType)
	}
	if domain != "" {
		where += " AND domain = ?"
		args = append(args, domain)
	}
	if startTime != "" {
		where += " AND timestamp >= ?"
		args = append(args, startTime)
	}
	if endTime != "" {
		where += " AND timestamp <= ?"
		args = append(args, endTime)
	}
	if startTime == "" && endTime == "" {
		where += " AND timestamp >= ?"
		args = append(args, time.Now().Add(-24*time.Hour).Format("2006-01-02 15:04:05"))
	}

	var total uint64
	if err := ch.QueryRow(context.Background(), fmt.Sprintf("SELECT count() FROM attack_logs %s", where), args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}

	querySQL := fmt.Sprintf("SELECT domain, ip, url, attack_type, rule_id, action, timestamp FROM attack_logs %s ORDER BY timestamp DESC LIMIT %d OFFSET %d", where, pageSize, (page-1)*pageSize)
	rows, err := ch.Query(context.Background(), querySQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var d, ipAddr, urlStr, at, action, ts string
		var ruleID int64
		if err := rows.Scan(&d, &ipAddr, &urlStr, &at, &ruleID, &action, &ts); err != nil {
			continue
		}
		list = append(list, gin.H{
			"domain":      d,
			"ip":          ipAddr,
			"url":         urlStr,
			"attack_type": at,
			"rule_id":     ruleID,
			"action":      action,
			"timestamp":   ts,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetLayer4Logs 四层转发日志
// GET /api/v1/logs/layer4
func GetLayer4Logs(c *gin.Context) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "日志服务不可用"})
		return
	}

	userID := c.MustGet("user_id").(uint)
	db := c.MustGet("db").(*gorm.DB)

	// 获取用户的四层转发监听端口
	var forwards []models.Layer4Forward
	db.Where("user_id = ?", userID).Find(&forwards)
	if len(forwards) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	ports := make([]int, len(forwards))
	for i, f := range forwards {
		ports[i] = f.ListenPort
	}

	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	page, pageSize := parsePagination(c)

	where := "WHERE listen_port IN (?)"
	args := []interface{}{ports}
	if startTime != "" {
		where += " AND timestamp >= ?"
		args = append(args, startTime)
	}
	if endTime != "" {
		where += " AND timestamp <= ?"
		args = append(args, endTime)
	}
	if startTime == "" && endTime == "" {
		where += " AND timestamp >= ?"
		args = append(args, time.Now().Add(-24*time.Hour).Format("2006-01-02 15:04:05"))
	}

	var total uint64
	if err := ch.QueryRow(context.Background(), fmt.Sprintf("SELECT count() FROM layer4_logs %s", where), args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}

	querySQL := fmt.Sprintf("SELECT listen_port, protocol, src_ip, dst_ip, bytes_in, bytes_out, timestamp FROM layer4_logs %s ORDER BY timestamp DESC LIMIT %d OFFSET %d", where, pageSize, (page-1)*pageSize)
	rows, err := ch.Query(context.Background(), querySQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var proto, srcIP, dstIP, ts string
		var port, bytesIn, bytesOut int64
		if err := rows.Scan(&port, &proto, &srcIP, &dstIP, &bytesIn, &bytesOut, &ts); err != nil {
			continue
		}
		list = append(list, gin.H{
			"listen_port": port,
			"protocol":    proto,
			"src_ip":      srcIP,
			"dst_ip":      dstIP,
			"bytes_in":    bytesIn,
			"bytes_out":   bytesOut,
			"timestamp":   ts,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetLayer4InterceptLogs 四层拦截日志
// GET /api/v1/logs/layer4-intercept
func GetLayer4InterceptLogs(c *gin.Context) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "日志服务不可用"})
		return
	}

	userID := c.MustGet("user_id").(uint)
	db := c.MustGet("db").(*gorm.DB)

	var forwards []models.Layer4Forward
	db.Where("user_id = ?", userID).Find(&forwards)
	if len(forwards) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	ports := make([]int, len(forwards))
	for i, f := range forwards {
		ports[i] = f.ListenPort
	}

	attackType := c.Query("attack_type")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	page, pageSize := parsePagination(c)

	where := "WHERE listen_port IN (?)"
	args := []interface{}{ports}
	if attackType != "" {
		where += " AND attack_type = ?"
		args = append(args, attackType)
	}
	if startTime != "" {
		where += " AND timestamp >= ?"
		args = append(args, startTime)
	}
	if endTime != "" {
		where += " AND timestamp <= ?"
		args = append(args, endTime)
	}
	if startTime == "" && endTime == "" {
		where += " AND timestamp >= ?"
		args = append(args, time.Now().Add(-24*time.Hour).Format("2006-01-02 15:04:05"))
	}

	var total uint64
	if err := ch.QueryRow(context.Background(), fmt.Sprintf("SELECT count() FROM layer4_intercept_logs %s", where), args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}

	querySQL := fmt.Sprintf("SELECT listen_port, src_ip, attack_type, action, timestamp FROM layer4_intercept_logs %s ORDER BY timestamp DESC LIMIT %d OFFSET %d", where, pageSize, (page-1)*pageSize)
	rows, err := ch.Query(context.Background(), querySQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var srcIP, at, action, ts string
		var port int64
		if err := rows.Scan(&port, &srcIP, &at, &action, &ts); err != nil {
			continue
		}
		list = append(list, gin.H{
			"listen_port": port,
			"src_ip":      srcIP,
			"attack_type": at,
			"action":      action,
			"timestamp":   ts,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetAILogs AI调用日志
// GET /api/v1/logs/ai
func GetAILogs(c *gin.Context) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "日志服务不可用"})
		return
	}

	userID := c.MustGet("user_id").(uint)
	db := c.MustGet("db").(*gorm.DB)

	var domains []string
	db.Model(&models.Domain{}).Where("user_id = ?", userID).Pluck("domain_name", &domains)
	if len(domains) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	domain := c.Query("domain")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	page, pageSize := parsePagination(c)

	where := "WHERE domain IN (?)"
	args := []interface{}{domains}
	if domain != "" {
		where += " AND domain = ?"
		args = append(args, domain)
	}
	if startTime != "" {
		where += " AND timestamp >= ?"
		args = append(args, startTime)
	}
	if endTime != "" {
		where += " AND timestamp <= ?"
		args = append(args, endTime)
	}
	if startTime == "" && endTime == "" {
		where += " AND timestamp >= ?"
		args = append(args, time.Now().Add(-24*time.Hour).Format("2006-01-02 15:04:05"))
	}

	var total uint64
	if err := ch.QueryRow(context.Background(), fmt.Sprintf("SELECT count() FROM ai_logs %s", where), args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}

	querySQL := fmt.Sprintf("SELECT domain, ip, feature, model, score, action, timestamp FROM ai_logs %s ORDER BY timestamp DESC LIMIT %d OFFSET %d", where, pageSize, (page-1)*pageSize)
	rows, err := ch.Query(context.Background(), querySQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var d, ipAddr, feature, model, action, ts string
		var score float64
		if err := rows.Scan(&d, &ipAddr, &feature, &model, &score, &action, &ts); err != nil {
			continue
		}
		list = append(list, gin.H{
			"domain":    d,
			"ip":        ipAddr,
			"feature":   feature,
			"model":     model,
			"score":     score,
			"action":    action,
			"timestamp": ts,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// ExportLogs 日志导出 CSV/JSON
// POST /api/v1/logs/export
func ExportLogs(c *gin.Context) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "日志服务不可用"})
		return
	}

	var req struct {
		LogType   string `json:"log_type" binding:"required"` // access / attack / layer4 / layer4_intercept / ai
		Format    string `json:"format"`                       // csv / json, 默认 csv
		Domain    string `json:"domain"`
		IP        string `json:"ip"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		Limit     int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if req.Format == "" {
		req.Format = "csv"
	}
	if req.Limit <= 0 || req.Limit > 10000 {
		req.Limit = 1000
	}

	tableMap := map[string]string{
		"access":          "access_logs",
		"attack":          "attack_logs",
		"layer4":          "layer4_logs",
		"layer4_intercept": "layer4_intercept_logs",
		"ai":              "ai_logs",
	}
	table, ok := tableMap[req.LogType]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "不支持的日志类型"})
		return
	}

	where := "WHERE 1=1"
	args := []interface{}{}
	if req.Domain != "" {
		where += " AND domain = ?"
		args = append(args, req.Domain)
	}
	if req.IP != "" {
		where += " AND ip = ?"
		args = append(args, req.IP)
	}
	if req.StartTime != "" {
		where += " AND timestamp >= ?"
		args = append(args, req.StartTime)
	}
	if req.EndTime != "" {
		where += " AND timestamp <= ?"
		args = append(args, req.EndTime)
	}
	if req.StartTime == "" && req.EndTime == "" {
		where += " AND timestamp >= ?"
		args = append(args, time.Now().Add(-24*time.Hour).Format("2006-01-02 15:04:05"))
	}

	querySQL := fmt.Sprintf("SELECT * FROM %s %s ORDER BY timestamp DESC LIMIT %d", table, where, req.Limit)
	rows, err := ch.Query(context.Background(), querySQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	columns := rows.Columns()
	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	switch req.Format {
	case "json":
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s_export.json", req.LogType))
		c.Header("Content-Type", "application/json")
		json.NewEncoder(c.Writer).Encode(results)
	case "csv":
		fallthrough
	default:
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s_export.csv", req.LogType))
		c.Header("Content-Type", "text/csv")
		w := csv.NewWriter(c.Writer)
		w.Write(columns)
		for _, row := range results {
			record := make([]string, len(columns))
			for i, col := range columns {
				record[i] = fmt.Sprintf("%v", row[col])
			}
			w.Write(record)
		}
		w.Flush()
	}
}

// GetLogMap 日志地图: 地理位置分布
// GET /api/v1/logs/map
func GetLogMap(c *gin.Context) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "日志服务不可用"})
		return
	}

	userID := c.MustGet("user_id").(uint)
	db := c.MustGet("db").(*gorm.DB)

	var domains []string
	db.Model(&models.Domain{}).Where("user_id = ?", userID).Pluck("domain_name", &domains)
	if len(domains) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": []interface{}{}})
		return
	}

	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	where := "WHERE domain IN (?)"
	args := []interface{}{domains}
	if startTime != "" {
		where += " AND timestamp >= ?"
		args = append(args, startTime)
	}
	if endTime != "" {
		where += " AND timestamp <= ?"
		args = append(args, endTime)
	}
	if startTime == "" && endTime == "" {
		where += " AND timestamp >= ?"
		args = append(args, time.Now().Add(-24*time.Hour).Format("2006-01-02 15:04:05"))
	}

	querySQL := fmt.Sprintf("SELECT country, city, count() as cnt, any(ip) as sample_ip FROM access_logs %s GROUP BY country, city ORDER BY cnt DESC LIMIT 200", where)
	rows, err := ch.Query(context.Background(), querySQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var country, city, sampleIP string
		var cnt uint64
		if err := rows.Scan(&country, &city, &cnt, &sampleIP); err != nil {
			continue
		}
		list = append(list, gin.H{
			"country":    country,
			"city":       city,
			"count":      cnt,
			"sample_ip":  sampleIP,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// 避免未使用导入
var _ = strconv.Atoi
