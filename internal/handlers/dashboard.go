package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/models"
	"gorm.io/gorm"
)

// DashboardHandler 仪表盘处理器
type DashboardHandler struct {
	db *gorm.DB
	ch interface{}
}

// NewDashboardHandler 创建仪表盘处理器
func NewDashboardHandler(db *gorm.DB, ch interface{}) *DashboardHandler {
	return &DashboardHandler{db: db, ch: ch}
}

// Analysis 仪表盘数据
func (h *DashboardHandler) Analysis(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	var domainCount int64
	var todayRequests int64
	var todayBlocked int64
	var todayTraffic int64

	// 域名数量
	query := h.db.Model(&models.Domain{})
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}
	query.Count(&domainCount)

	// 今日请求数/拦截数/流量 (从 ClickHouse 查询)
	// TODO: 查 ClickHouse
	_ = todayRequests
	_ = todayBlocked
	_ = todayTraffic

	// 访问统计/攻击统计 (时序数据)
	// TODO: 查 ClickHouse

	// 地理分布
	// TODO: 查 ClickHouse

	c.JSON(200, gin.H{
		"code": 0,
		"message": "success",
		"data": gin.H{
			"domain_count":      domainCount,
			"today_requests":    todayRequests,
			"today_blocked":     todayBlocked,
			"today_traffic":     todayTraffic,
			"access_stats":      []interface{}{},
			"attack_stats":      []interface{}{},
			"geo_distribution":  gin.H{"china": []interface{}{}, "world": []interface{}{}},
		},
	})
}
