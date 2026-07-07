package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/models"
	"gorm.io/gorm"
)

// ============================================================
// DDoS 防护管理 Handler (admin)
// ============================================================
//
// 路由清单:
//   GET    /api/v1/admin/ddos/dashboard
//   GET    /api/v1/admin/ddos/rules
//   POST   /api/v1/admin/ddos/rules
//   PUT    /api/v1/admin/ddos/rules/:id
//   DELETE /api/v1/admin/ddos/rules/:id
//   GET    /api/v1/admin/ddos/blacklist
//   POST   /api/v1/admin/ddos/blacklist
//   DELETE /api/v1/admin/ddos/blacklist/:id
//   GET    /api/v1/admin/ddos/whitelist
//   POST   /api/v1/admin/ddos/whitelist
//   DELETE /api/v1/admin/ddos/whitelist/:id
//   GET    /api/v1/admin/ddos/logs
//   GET    /api/v1/admin/ddos/intercept-logs
//
// 统一响应格式: gin.H{"code": 0, "message": "success", "data": ...}
// 数据库: c.MustGet("db").(*gorm.DB)
// ClickHouse 可选: c.MustGet("clickhouse").(driver.Conn)

// ------------------------------------------------------------
// 辅助函数
// ------------------------------------------------------------

func ddosSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data})
}

func ddosFail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": code, "message": msg, "data": nil})
}

func ddosPagination(c *gin.Context) (page, pageSize int, offset int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 20
	}
	offset = (page - 1) * pageSize
	return
}

// getClickHouse 从上下文获取 ClickHouse 连接(可能不存在)
func getClickHouse(c *gin.Context) driver.Conn {
	if v, ok := c.Get("clickhouse"); ok {
		if ch, ok := v.(driver.Conn); ok {
			return ch
		}
	}
	return nil
}

// ------------------------------------------------------------
// DDoSDashboard DDoS 仪表盘
// GET /api/v1/admin/ddos/dashboard
// ------------------------------------------------------------
func DDoSDashboard(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	// 当前活跃规则数
	var activeRules int64
	db.Model(&models.DDoSRule{}).Where("status = ?", "active").Count(&activeRules)

	// 当前黑名单 IP 数
	var blacklisted int64
	db.Model(&models.DDoSBlacklistEntry{}).
		Where("type = ? AND (expires_at IS NULL OR expires_at > ?)", "black", time.Now()).
		Count(&blacklisted)

	// 当前白名单 IP 数
	var whitelisted int64
	db.Model(&models.DDoSBlacklistEntry{}).
		Where("type = ?", "white").
		Count(&whitelisted)

	// 近 24h 自动封禁数
	var autoBanned24h int64
	db.Model(&models.DDoSBlacklistEntry{}).
		Where("type = ? AND auto_ban = ? AND created_at > ?", "black", true, time.Now().Add(-24*time.Hour)).
		Count(&autoBanned24h)

	// 当前攻击(从 ClickHouse 拉取近 10 分钟的拦截事件统计)
	currentAttacks := []map[string]interface{}{}
	if ch := getClickHouse(c); ch != nil {
		ctx := c.Request.Context()
		query := `
			SELECT
				toStartOfMinute(event_time) AS minute,
				count() AS hits,
				uniqExact(src_ip) AS uniq_src
			FROM ddos_intercept_logs
			WHERE event_time >= now() - INTERVAL 10 MINUTE
			GROUP BY minute
			ORDER BY minute DESC
			LIMIT 60
		`
		rows, err := ch.Query(ctx, query)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var minute time.Time
				var hits uint64
				var uniqSrc uint64
				if err := rows.Scan(&minute, &hits, &uniqSrc); err == nil {
					currentAttacks = append(currentAttacks, gin.H{
						"minute":   minute,
						"hits":     hits,
						"uniq_src": uniqSrc,
					})
				}
			}
		}
	}

	// 历史统计(从 ClickHouse)
	historyStats := gin.H{}
	if ch := getClickHouse(c); ch != nil {
		ctx := c.Request.Context()
		// 近 7 天每日拦截量
		query := `
			SELECT
				toStartOfDay(event_time) AS day,
				count() AS total,
				uniqExact(src_ip) AS uniq_src
			FROM ddos_intercept_logs
			WHERE event_time >= now() - INTERVAL 7 DAY
			GROUP BY day
			ORDER BY day ASC
		`
		rows, err := ch.Query(ctx, query)
		if err == nil {
			defer rows.Close()
			type dailyStat struct {
				Day    time.Time `json:"day"`
				Total  uint64    `json:"total"`
				UniqIP uint64    `json:"uniq_ip"`
			}
			stats := []dailyStat{}
			for rows.Next() {
				var ds dailyStat
				if err := rows.Scan(&ds.Day, &ds.Total, &ds.UniqIP); err == nil {
					stats = append(stats, ds)
				}
			}
			historyStats["daily_intercept"] = stats
		}
	}

	ddosSuccess(c, gin.H{
		"summary": gin.H{
			"active_rules":     activeRules,
			"blacklisted_ips":  blacklisted,
			"whitelisted_ips":  whitelisted,
			"auto_banned_24h":  autoBanned24h,
		},
		"current_attacks":  currentAttacks,
		"history_stats":    historyStats,
	})
}

// ------------------------------------------------------------
// DDoSRuleList DDoS 规则列表
// GET /api/v1/admin/ddos/rules
// ------------------------------------------------------------
func DDoSRuleList(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	page, pageSize, offset := ddosPagination(c)

	var total int64
	db.Model(&models.DDoSRule{}).Count(&total)

	var rules []models.DDoSRule
	if err := db.Order("id DESC").Offset(offset).Limit(pageSize).Find(&rules).Error; err != nil {
		ddosFail(c, 500, fmt.Sprintf("查询规则失败: %v", err))
		return
	}

	ddosSuccess(c, gin.H{
		"list":       rules,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// ------------------------------------------------------------
// DDoSRuleCreate 创建 DDoS 规则
// POST /api/v1/admin/ddos/rules
// ------------------------------------------------------------
func DDoSRuleCreate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var rule models.DDoSRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		ddosFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	if rule.Name == "" {
		ddosFail(c, 400, "规则名称不能为空")
		return
	}

	// 默认值
	if rule.Scope == "" {
		rule.Scope = "global"
	}
	if rule.Status == "" {
		rule.Status = "active"
	}
	if rule.BanDurationSeconds == 0 {
		rule.BanDurationSeconds = 3600
	}

	if err := db.Create(&rule).Error; err != nil {
		ddosFail(c, 500, fmt.Sprintf("创建规则失败: %v", err))
		return
	}

	ddosSuccess(c, rule)
}

// ------------------------------------------------------------
// DDoSRuleUpdate 更新 DDoS 规则
// PUT /api/v1/admin/ddos/rules/:id
// ------------------------------------------------------------
func DDoSRuleUpdate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ddosFail(c, 400, "无效的 ID")
		return
	}

	var rule models.DDoSRule
	if err := db.First(&rule, id).Error; err != nil {
		ddosFail(c, 404, "规则不存在")
		return
	}

	var updates models.DDoSRule
	if err := c.ShouldBindJSON(&updates); err != nil {
		ddosFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	// 字段更新
	updates.ID = rule.ID // 保护 ID
	if err := db.Model(&rule).Updates(&updates).Error; err != nil {
		ddosFail(c, 500, fmt.Sprintf("更新规则失败: %v", err))
		return
	}

	// 重新查询返回
	db.First(&rule, id)
	ddosSuccess(c, rule)
}

// ------------------------------------------------------------
// DDoSRuleDelete 删除 DDoS 规则
// DELETE /api/v1/admin/ddos/rules/:id
// ------------------------------------------------------------
func DDoSRuleDelete(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ddosFail(c, 400, "无效的 ID")
		return
	}

	if err := db.Delete(&models.DDoSRule{}, id).Error; err != nil {
		ddosFail(c, 500, fmt.Sprintf("删除规则失败: %v", err))
		return
	}

	ddosSuccess(c, gin.H{"id": id})
}

// ------------------------------------------------------------
// DDoSBlacklistList DDoS 黑名单列表
// GET /api/v1/admin/ddos/blacklist
// ------------------------------------------------------------
func DDoSBlacklistList(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	page, pageSize, offset := ddosPagination(c)

	q := db.Model(&models.DDoSBlacklistEntry{}).Where("type = ?", "black")
	if ip := c.Query("ip"); ip != "" {
		q = q.Where("ip LIKE ?", "%"+ip+"%")
	}
	if autoBan := c.Query("auto_ban"); autoBan != "" {
		if autoBan == "1" || autoBan == "true" {
			q = q.Where("auto_ban = ?", true)
		} else if autoBan == "0" || autoBan == "false" {
			q = q.Where("auto_ban = ?", false)
		}
	}
	if expired := c.Query("expired"); expired == "1" {
		q = q.Where("expires_at IS NOT NULL AND expires_at <= ?", time.Now())
	} else if expired == "0" {
		q = q.Where("expires_at IS NULL OR expires_at > ?", time.Now())
	}

	var total int64
	q.Count(&total)

	var list []models.DDoSBlacklistEntry
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		ddosFail(c, 500, fmt.Sprintf("查询黑名单失败: %v", err))
		return
	}

	ddosSuccess(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ------------------------------------------------------------
// DDoSBlacklistCreate 添加 DDoS 黑名单
// POST /api/v1/admin/ddos/blacklist
// ------------------------------------------------------------
func DDoSBlacklistCreate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var entry models.DDoSBlacklistEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		ddosFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	if entry.IP == "" && entry.CIDR == "" {
		ddosFail(c, 400, "IP 或 CIDR 不能同时为空")
		return
	}

	entry.Type = "black"
	entry.AutoBan = false // 手动添加的标记为非自动

	if err := db.Create(&entry).Error; err != nil {
		ddosFail(c, 500, fmt.Sprintf("添加黑名单失败: %v", err))
		return
	}

	ddosSuccess(c, entry)
}

// ------------------------------------------------------------
// DDoSBlacklistDelete 删除 DDoS 黑名单
// DELETE /api/v1/admin/ddos/blacklist/:id
// ------------------------------------------------------------
func DDoSBlacklistDelete(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ddosFail(c, 400, "无效的 ID")
		return
	}

	if err := db.Delete(&models.DDoSBlacklistEntry{}, id).Error; err != nil {
		ddosFail(c, 500, fmt.Sprintf("删除黑名单失败: %v", err))
		return
	}

	ddosSuccess(c, gin.H{"id": id})
}

// ------------------------------------------------------------
// DDoSWhitelistList DDoS 白名单列表
// GET /api/v1/admin/ddos/whitelist
// ------------------------------------------------------------
func DDoSWhitelistList(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	page, pageSize, offset := ddosPagination(c)

	q := db.Model(&models.DDoSBlacklistEntry{}).Where("type = ?", "white")
	if ip := c.Query("ip"); ip != "" {
		q = q.Where("ip LIKE ?", "%"+ip+"%")
	}

	var total int64
	q.Count(&total)

	var list []models.DDoSBlacklistEntry
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		ddosFail(c, 500, fmt.Sprintf("查询白名单失败: %v", err))
		return
	}

	ddosSuccess(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ------------------------------------------------------------
// DDoSWhitelistCreate 添加 DDoS 白名单
// POST /api/v1/admin/ddos/whitelist
// ------------------------------------------------------------
func DDoSWhitelistCreate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var entry models.DDoSBlacklistEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		ddosFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	if entry.IP == "" && entry.CIDR == "" {
		ddosFail(c, 400, "IP 或 CIDR 不能同时为空")
		return
	}

	entry.Type = "white"

	if err := db.Create(&entry).Error; err != nil {
		ddosFail(c, 500, fmt.Sprintf("添加白名单失败: %v", err))
		return
	}

	ddosSuccess(c, entry)
}

// ------------------------------------------------------------
// DDoSWhitelistDelete 删除 DDoS 白名单
// DELETE /api/v1/admin/ddos/whitelist/:id
// ------------------------------------------------------------
func DDoSWhitelistDelete(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ddosFail(c, 400, "无效的 ID")
		return
	}

	if err := db.Delete(&models.DDoSBlacklistEntry{}, id).Error; err != nil {
		ddosFail(c, 500, fmt.Sprintf("删除白名单失败: %v", err))
		return
	}

	ddosSuccess(c, gin.H{"id": id})
}

// ------------------------------------------------------------
// DDoSConnectionLogs 四层连接日志 (ClickHouse)
// GET /api/v1/admin/ddos/logs
// ------------------------------------------------------------
func DDoSConnectionLogs(c *gin.Context) {
	page, pageSize, offset := ddosPagination(c)

	ch := getClickHouse(c)
	if ch == nil {
		// ClickHouse 未配置时返回空列表
		ddosSuccess(c, gin.H{
			"list":      []interface{}{},
			"total":     0,
			"page":      page,
			"page_size": pageSize,
			"note":      "clickhouse not configured",
		})
		return
	}

	ctx := c.Request.Context()
	srcIP := c.Query("src_ip")
	dstIP := c.Query("dst_ip")
	startTime := c.DefaultQuery("start_time", time.Now().Add(-1*time.Hour).Format(time.RFC3339))
	endTime := c.DefaultQuery("end_time", time.Now().Format(time.RFC3339))

	where := "WHERE event_time >= ? AND event_time <= ?"
	args := []interface{}{startTime, endTime}
	if srcIP != "" {
		where += " AND src_ip = ?"
		args = append(args, srcIP)
	}
	if dstIP != "" {
		where += " AND dst_ip = ?"
		args = append(args, dstIP)
	}

	// 总数
	var total uint64
	countQuery := fmt.Sprintf("SELECT count() FROM l4_connection_logs %s", where)
	row := ch.QueryRow(ctx, countQuery, args...)
	if err := row.Err(); err == nil {
		_ = row.Scan(&total)
	}

	// 列表
	listQuery := fmt.Sprintf(`
		SELECT event_time, src_ip, src_port, dst_ip, dst_port, protocol, bytes_in, bytes_out, action
		FROM l4_connection_logs
		%s
		ORDER BY event_time DESC
		LIMIT %d OFFSET %d
	`, where, pageSize, offset)

	rows, err := ch.Query(ctx, listQuery, args...)
	if err != nil {
		ddosFail(c, 500, fmt.Sprintf("查询连接日志失败: %v", err))
		return
	}
	defer rows.Close()

	type connLog struct {
		EventTime time.Time `json:"event_time"`
		SrcIP     string    `json:"src_ip"`
		SrcPort   uint16    `json:"src_port"`
		DstIP     string    `json:"dst_ip"`
		DstPort   uint16    `json:"dst_port"`
		Protocol  string    `json:"protocol"`
		BytesIn   uint64    `json:"bytes_in"`
		BytesOut  uint64    `json:"bytes_out"`
		Action    string    `json:"action"`
	}
	list := []connLog{}
	for rows.Next() {
		var l connLog
		if err := rows.Scan(&l.EventTime, &l.SrcIP, &l.SrcPort, &l.DstIP, &l.DstPort, &l.Protocol, &l.BytesIn, &l.BytesOut, &l.Action); err == nil {
			list = append(list, l)
		}
	}

	ddosSuccess(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ------------------------------------------------------------
// DDoSInterceptLogs 四层拦截日志 (ClickHouse)
// GET /api/v1/admin/ddos/intercept-logs
// ------------------------------------------------------------
func DDoSInterceptLogs(c *gin.Context) {
	page, pageSize, offset := ddosPagination(c)

	ch := getClickHouse(c)
	if ch == nil {
		ddosSuccess(c, gin.H{
			"list":      []interface{}{},
			"total":     0,
			"page":      page,
			"page_size": pageSize,
			"note":      "clickhouse not configured",
		})
		return
	}

	ctx := c.Request.Context()
	srcIP := c.Query("src_ip")
	ruleID := c.Query("rule_id")
	startTime := c.DefaultQuery("start_time", time.Now().Add(-1*time.Hour).Format(time.RFC3339))
	endTime := c.DefaultQuery("end_time", time.Now().Format(time.RFC3339))

	where := "WHERE event_time >= ? AND event_time <= ?"
	args := []interface{}{startTime, endTime}
	if srcIP != "" {
		where += " AND src_ip = ?"
		args = append(args, srcIP)
	}
	if ruleID != "" {
		where += " AND rule_id = ?"
		args = append(args, ruleID)
	}

	var total uint64
	countQuery := fmt.Sprintf("SELECT count() FROM ddos_intercept_logs %s", where)
	row := ch.QueryRow(ctx, countQuery, args...)
	if err := row.Err(); err == nil {
		_ = row.Scan(&total)
	}

	listQuery := fmt.Sprintf(`
		SELECT event_time, src_ip, dst_ip, dst_port, protocol, rule_id, action, reason
		FROM ddos_intercept_logs
		%s
		ORDER BY event_time DESC
		LIMIT %d OFFSET %d
	`, where, pageSize, offset)

	rows, err := ch.Query(ctx, listQuery, args...)
	if err != nil {
		ddosFail(c, 500, fmt.Sprintf("查询拦截日志失败: %v", err))
		return
	}
	defer rows.Close()

	type interceptLog struct {
		EventTime time.Time `json:"event_time"`
		SrcIP     string    `json:"src_ip"`
		DstIP     string    `json:"dst_ip"`
		DstPort   uint16    `json:"dst_port"`
		Protocol  string    `json:"protocol"`
		RuleID    uint32    `json:"rule_id"`
		Action    string    `json:"action"`
		Reason    string    `json:"reason"`
	}
	list := []interceptLog{}
	for rows.Next() {
		var l interceptLog
		if err := rows.Scan(&l.EventTime, &l.SrcIP, &l.DstIP, &l.DstPort, &l.Protocol, &l.RuleID, &l.Action, &l.Reason); err == nil {
			list = append(list, l)
		}
	}

	ddosSuccess(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
