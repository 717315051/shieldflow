package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================
// WAF 管理 Handler (admin)
// ============================================================
//
// 路由清单:
//   GET    /api/v1/admin/waf/dashboard
//   GET    /api/v1/admin/waf/config
//   PUT    /api/v1/admin/waf/config
//   GET    /api/v1/admin/waf/logs
//   GET    /api/v1/admin/waf/analysis
//
// WAF 拦截日志存储在 ClickHouse attack_logs 表中（protection_type='waf'）。
// 全局 WAF 配置存储在 PostgreSQL system_settings 表中（key 前缀 waf.*）。

// wafGetCH 获取 ClickHouse 连接，不可用时返回 false 并已响应错误。
func wafGetCH(c *gin.Context) (driver.Conn, bool) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "日志服务不可用"})
		return nil, false
	}
	return ch, true
}

// ------------------------------------------------------------

// WAFDashboard WAF 防护效果统计
// GET /api/v1/admin/waf/dashboard
// 返回: 今日拦截数、本周拦截数、各威胁类型占比、Top攻击IP
func WAFDashboard(c *gin.Context) {
	ch, ok := wafGetCH(c)
	if !ok {
		return
	}

	now := time.Now()
	todayStart := now.Format("2006-01-02 00:00:00")
	weekStart := now.Add(-7 * 24 * time.Hour).Format("2006-01-02 00:00:00")

	// 今日拦截数。
	var todayCount uint64
	if err := ch.QueryRow(context.Background(),
		"SELECT count() FROM attack_logs WHERE protection_type = 'waf' AND timestamp >= ?", todayStart,
	).Scan(&todayCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}

	// 本周拦截数。
	var weekCount uint64
	if err := ch.QueryRow(context.Background(),
		"SELECT count() FROM attack_logs WHERE protection_type = 'waf' AND timestamp >= ?", weekStart,
	).Scan(&weekCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}

	// 各威胁类型占比（今日）。
	rows, err := ch.Query(context.Background(),
		"SELECT attack_type, count() as cnt FROM attack_logs WHERE protection_type = 'waf' AND timestamp >= ? GROUP BY attack_type ORDER BY cnt DESC", todayStart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	type threatTypeStat struct {
		AttackType string `json:"attack_type"`
		Count      uint64 `json:"count"`
	}
	var threatTypes []threatTypeStat
	for rows.Next() {
		var at string
		var cnt uint64
		if err := rows.Scan(&at, &cnt); err != nil {
			continue
		}
		threatTypes = append(threatTypes, threatTypeStat{AttackType: at, Count: cnt})
	}

	// Top 攻击 IP（今日）。
	rows2, err := ch.Query(context.Background(),
		"SELECT client_ip, count() as cnt FROM attack_logs WHERE protection_type = 'waf' AND timestamp >= ? GROUP BY client_ip ORDER BY cnt DESC LIMIT 20", todayStart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows2.Close()

	type topIPStat struct {
		IP    string `json:"ip"`
		Count uint64 `json:"count"`
	}
	var topIPs []topIPStat
	for rows2.Next() {
		var ip string
		var cnt uint64
		if err := rows2.Scan(&ip, &cnt); err != nil {
			continue
		}
		topIPs = append(topIPs, topIPStat{IP: ip, Count: cnt})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"today_count":  todayCount,
			"week_count":   weekCount,
			"threat_types": threatTypes,
			"top_ips":      topIPs,
		},
	})
}

// ------------------------------------------------------------

// WAFConfigGet 获取全局 WAF 配置
// GET /api/v1/admin/waf/config
func WAFConfigGet(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	settings := getSettingsByPrefix(db, "waf.")
	sysSuccess(c, settings)
}

// ------------------------------------------------------------

// WAFConfigUpdate 更新全局 WAF 配置
// PUT /api/v1/admin/waf/config
// Body: { "waf.enabled": "true", "waf.mode": "block", ... }
func WAFConfigUpdate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var updates map[string]string
	if err := c.ShouldBindJSON(&updates); err != nil {
		sysFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	for k, v := range updates {
		if err := setSetting(db, k, v); err != nil {
			sysFail(c, 500, fmt.Sprintf("更新 %s 失败: %v", k, err))
			return
		}
	}

	sysSuccess(c, getSettingsByPrefix(db, "waf."))
}

// ------------------------------------------------------------

// WAFLogs WAF 拦截日志查询
// GET /api/v1/admin/waf/logs
// 支持按域名/IP/威胁类型/时间筛选，分页查询。
func WAFLogs(c *gin.Context) {
	ch, ok := wafGetCH(c)
	if !ok {
		return
	}

	domain := c.Query("domain")
	ip := c.Query("ip")
	attackType := c.Query("attack_type")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	page, pageSize := parsePagination(c)

	where := "WHERE protection_type = 'waf'"
	args := []interface{}{}
	if domain != "" {
		where += " AND domain = ?"
		args = append(args, domain)
	}
	if ip != "" {
		where += " AND client_ip = ?"
		args = append(args, ip)
	}
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

	// 查询总数。
	var total uint64
	if err := ch.QueryRow(context.Background(),
		fmt.Sprintf("SELECT count() FROM attack_logs %s", where), args...,
	).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}

	// 查询数据。
	querySQL := fmt.Sprintf(
		"SELECT domain, attack_type, method, client_ip, url, action, detail, timestamp FROM attack_logs %s ORDER BY timestamp DESC LIMIT %d OFFSET %d",
		where, pageSize, (page-1)*pageSize)
	rows, err := ch.Query(context.Background(), querySQL, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var d, at, method, ipAddr, urlStr, action, detail, ts string
		if err := rows.Scan(&d, &at, &method, &ipAddr, &urlStr, &action, &detail, &ts); err != nil {
			continue
		}
		list = append(list, gin.H{
			"domain":      d,
			"attack_type": at,
			"method":      method,
			"ip":          ipAddr,
			"url":         urlStr,
			"action":      action,
			"detail":      detail,
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

// ------------------------------------------------------------

// WAFAttackAnalysis 攻击模式分析
// GET /api/v1/admin/waf/analysis
// 按威胁类型分组统计攻击趋势（近7天）。
func WAFAttackAnalysis(c *gin.Context) {
	ch, ok := wafGetCH(c)
	if !ok {
		return
	}

	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	if startTime == "" {
		startTime = time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02 00:00:00")
	}
	if endTime == "" {
		endTime = time.Now().Format("2006-01-02 15:04:05")
	}

	// 按威胁类型 + 日期分组统计。
	querySQL := "SELECT attack_type, toDate(timestamp) as day, count() as cnt FROM attack_logs WHERE protection_type = 'waf' AND timestamp >= ? AND timestamp <= ? GROUP BY attack_type, day ORDER BY day, attack_type"
	rows, err := ch.Query(context.Background(), querySQL, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	type dayStat struct {
		AttackType string `json:"attack_type"`
		Day        string `json:"day"`
		Count      uint64 `json:"count"`
	}
	var trends []dayStat
	for rows.Next() {
		var at, day string
		var cnt uint64
		if err := rows.Scan(&at, &day, &cnt); err != nil {
			continue
		}
		trends = append(trends, dayStat{AttackType: at, Day: day, Count: cnt})
	}

	// 按威胁类型汇总（总计）。
	summarySQL := "SELECT attack_type, count() as cnt FROM attack_logs WHERE protection_type = 'waf' AND timestamp >= ? AND timestamp <= ? GROUP BY attack_type ORDER BY cnt DESC"
	rows2, err := ch.Query(context.Background(), summarySQL, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows2.Close()

	type typeSummary struct {
		AttackType string `json:"attack_type"`
		Count      uint64 `json:"count"`
	}
	var summary []typeSummary
	for rows2.Next() {
		var at string
		var cnt uint64
		if err := rows2.Scan(&at, &cnt); err != nil {
			continue
		}
		summary = append(summary, typeSummary{AttackType: at, Count: cnt})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"trends":  trends,
			"summary": summary,
		},
	})
}
