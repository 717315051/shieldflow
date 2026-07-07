package handlers

import (
	"context"
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
// AI 管理 Handler (admin)
// ============================================================
//
// 路由清单:
//   GET    /api/v1/admin/ai/dashboard
//   GET    /api/v1/admin/ai/token-stats
//   GET    /api/v1/admin/ai/cost-analysis
//   GET    /api/v1/admin/ai/models
//   POST   /api/v1/admin/ai/models
//   PUT    /api/v1/admin/ai/models/:id
//   DELETE /api/v1/admin/ai/models/:id
//
// AI 调用日志存储在 ClickHouse ai_logs 表中。
// AI 模型配置存储在 PostgreSQL ai_config 表中。

// aiGetCH 获取 ClickHouse 连接，不可用时返回 false 并已响应错误。
func aiGetCH(c *gin.Context) (driver.Conn, bool) {
	ch, ok := c.MustGet("ch").(driver.Conn)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 1, "message": "日志服务不可用"})
		return nil, false
	}
	return ch, true
}

// ------------------------------------------------------------

// AIDashboard AI 防护全局统计
// GET /api/v1/admin/ai/dashboard
// 返回: 今日调用次数、Token 消耗、各功能调用占比。
func AIDashboard(c *gin.Context) {
	ch, ok := aiGetCH(c)
	if !ok {
		return
	}

	todayStart := time.Now().Format("2006-01-02 00:00:00")

	// 今日调用次数。
	var todayCalls uint64
	if err := ch.QueryRow(context.Background(),
		"SELECT count() FROM ai_logs WHERE timestamp >= ?", todayStart,
	).Scan(&todayCalls); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}

	// 今日 Token 消耗。
	var todayTokens uint64
	if err := ch.QueryRow(context.Background(),
		"SELECT sum(token_count) FROM ai_logs WHERE timestamp >= ?", todayStart,
	).Scan(&todayTokens); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}

	// 各功能调用占比（今日）。
	rows, err := ch.Query(context.Background(),
		"SELECT feature, count() as cnt, sum(token_count) as tokens FROM ai_logs WHERE timestamp >= ? GROUP BY feature ORDER BY cnt DESC", todayStart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	type featureStat struct {
		Feature string `json:"feature"`
		Count   uint64 `json:"count"`
		Tokens  uint64 `json:"tokens"`
	}
	var features []featureStat
	for rows.Next() {
		var f string
		var cnt, tokens uint64
		if err := rows.Scan(&f, &cnt, &tokens); err != nil {
			continue
		}
		features = append(features, featureStat{Feature: f, Count: cnt, Tokens: tokens})
	}

	// 本周 Token 消耗趋势（按天）。
	weekStart := time.Now().Add(-7 * 24 * time.Hour).Format("2006-01-02 00:00:00")
	rows2, err := ch.Query(context.Background(),
		"SELECT toDate(timestamp) as day, sum(token_count) as tokens, count() as cnt FROM ai_logs WHERE timestamp >= ? GROUP BY day ORDER BY day", weekStart)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows2.Close()

	type dayTrend struct {
		Day    string `json:"day"`
		Tokens uint64 `json:"tokens"`
		Count  uint64 `json:"count"`
	}
	var trends []dayTrend
	for rows2.Next() {
		var day string
		var tokens, cnt uint64
		if err := rows2.Scan(&day, &tokens, &cnt); err != nil {
			continue
		}
		trends = append(trends, dayTrend{Day: day, Tokens: tokens, Count: cnt})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"today_calls":  todayCalls,
			"today_tokens": todayTokens,
			"features":     features,
			"trends":       trends,
		},
	})
}

// ------------------------------------------------------------

// AITokenStats Token 使用统计
// GET /api/v1/admin/ai/token-stats
// 按天/模型统计 Token 使用（默认近30天）。
func AITokenStats(c *gin.Context) {
	ch, ok := aiGetCH(c)
	if !ok {
		return
	}

	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	if startTime == "" {
		startTime = time.Now().Add(-30 * 24 * time.Hour).Format("2006-01-02 00:00:00")
	}
	if endTime == "" {
		endTime = time.Now().Format("2006-01-02 15:04:05")
	}

	// ai_logs 表没有 model 列，按 feature 统计。
	rows, err := ch.Query(context.Background(),
		"SELECT feature, toDate(timestamp) as day, sum(token_count) as tokens, count() as cnt FROM ai_logs WHERE timestamp >= ? AND timestamp <= ? GROUP BY feature, day ORDER BY day, feature",
		startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	type tokenStat struct {
		Feature string `json:"feature"`
		Day     string `json:"day"`
		Tokens  uint64 `json:"tokens"`
		Count   uint64 `json:"count"`
	}
	var stats []tokenStat
	for rows.Next() {
		var f, day string
		var tokens, cnt uint64
		if err := rows.Scan(&f, &day, &tokens, &cnt); err != nil {
			continue
		}
		stats = append(stats, tokenStat{Feature: f, Day: day, Tokens: tokens, Count: cnt})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"stats":      stats,
			"start_time": startTime,
			"end_time":   endTime,
		},
	})
}

// ------------------------------------------------------------

// AICostAnalysis 成本分析
// GET /api/v1/admin/ai/cost-analysis
// 按模型/功能计算成本（cost 字段在 ai_logs 表中）。
func AICostAnalysis(c *gin.Context) {
	ch, ok := aiGetCH(c)
	if !ok {
		return
	}

	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	if startTime == "" {
		startTime = time.Now().Add(-30 * 24 * time.Hour).Format("2006-01-02 00:00:00")
	}
	if endTime == "" {
		endTime = time.Now().Format("2006-01-02 15:04:05")
	}

	// 按 feature 分组统计成本与 Token。
	rows, err := ch.Query(context.Background(),
		"SELECT feature, sum(token_count) as tokens, sum(cost) as total_cost, count() as cnt FROM ai_logs WHERE timestamp >= ? AND timestamp <= ? GROUP BY feature ORDER BY total_cost DESC",
		startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败: " + err.Error()})
		return
	}
	defer rows.Close()

	type costStat struct {
		Feature   string  `json:"feature"`
		Tokens    uint64  `json:"tokens"`
		Cost      float64 `json:"cost"`
		Calls     uint64  `json:"calls"`
	}
	var stats []costStat
	var totalCost float64
	for rows.Next() {
		var f string
		var tokens, cnt uint64
		var cost float64
		if err := rows.Scan(&f, &tokens, &cost, &cnt); err != nil {
			continue
		}
		stats = append(stats, costStat{Feature: f, Tokens: tokens, Cost: cost, Calls: cnt})
		totalCost += cost
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"stats":       stats,
			"total_cost":  totalCost,
			"start_time":  startTime,
			"end_time":    endTime,
		},
	})
}

// ------------------------------------------------------------

// AIModelList AI 模型列表
// GET /api/v1/admin/ai/models
func AIModelList(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var models []models.AIConfigModel
	if err := db.Find(&models).Error; err != nil {
		sysFail(c, 500, fmt.Sprintf("查询失败: %v", err))
		return
	}

	// 隐藏 APIKey，仅返回是否有。
	result := make([]gin.H, 0, len(models))
	for _, m := range models {
		result = append(result, gin.H{
			"id":          m.ID,
			"provider":    m.Provider,
			"model":       m.Model,
			"base_url":    m.BaseURL,
			"enabled":     m.Enabled,
			"has_api_key": m.APIKey != "",
			"created_at":  m.CreatedAt,
			"updated_at":  m.UpdatedAt,
		})
	}

	sysSuccess(c, gin.H{"list": result})
}

// ------------------------------------------------------------

// AIModelCreate 添加 AI 模型
// POST /api/v1/admin/ai/models
func AIModelCreate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
		APIKey   string `json:"api_key"`
		BaseURL  string `json:"base_url"`
		Enabled  bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sysFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}
	if req.Provider == "" {
		sysFail(c, 400, "provider 不能为空")
		return
	}

	m := models.AIConfigModel{
		Provider: req.Provider,
		Model:    req.Model,
		APIKey:   req.APIKey,
		BaseURL:  req.BaseURL,
		Enabled:  req.Enabled,
	}
	if err := db.Create(&m).Error; err != nil {
		sysFail(c, 500, fmt.Sprintf("创建失败: %v", err))
		return
	}

	sysSuccess(c, gin.H{
		"id":          m.ID,
		"provider":    m.Provider,
		"model":       m.Model,
		"base_url":    m.BaseURL,
		"enabled":     m.Enabled,
		"has_api_key": m.APIKey != "",
	})
}

// ------------------------------------------------------------

// AIModelUpdate 更新模型配置
// PUT /api/v1/admin/ai/models/:id
func AIModelUpdate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		sysFail(c, 400, "无效的 ID")
		return
	}

	var req struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
		APIKey   string `json:"api_key"`
		BaseURL  string `json:"base_url"`
		Enabled  bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sysFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	var m models.AIConfigModel
	if err := db.First(&m, id).Error; err != nil {
		sysFail(c, 404, "模型不存在")
		return
	}

	updates := map[string]interface{}{
		"provider": req.Provider,
		"model":    req.Model,
		"base_url": req.BaseURL,
		"enabled":  req.Enabled,
	}
	if req.APIKey != "" {
		updates["api_key"] = req.APIKey
	}
	if err := db.Model(&m).Updates(updates).Error; err != nil {
		sysFail(c, 500, fmt.Sprintf("更新失败: %v", err))
		return
	}

	db.First(&m, id)
	sysSuccess(c, gin.H{
		"id":          m.ID,
		"provider":    m.Provider,
		"model":       m.Model,
		"base_url":    m.BaseURL,
		"enabled":     m.Enabled,
		"has_api_key": m.APIKey != "",
	})
}

// ------------------------------------------------------------

// AIModelDelete 删除模型
// DELETE /api/v1/admin/ai/models/:id
func AIModelDelete(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		sysFail(c, 400, "无效的 ID")
		return
	}

	if err := db.Delete(&models.AIConfigModel{}, id).Error; err != nil {
		sysFail(c, 500, fmt.Sprintf("删除失败: %v", err))
		return
	}

	sysSuccess(c, nil)
}
