package handlers

// ============================================================
// 白名单专用 handler（按 API.md 文档规范）
// 复用 BlacklistEntry 表,通过 list_type='white' 区分
// ============================================================

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/shieldflow/shieldflow/internal/models"
)

// ------------------------------------------------------------
// WhitelistList 白名单列表
// GET /api/v1/protection/whitelist
// Query: type, value, domain_id, page, page_size
// ------------------------------------------------------------
func WhitelistList(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := getCurrentUserID(c)
	page, pageSize, offset := ptPagination(c)

	q := db.Model(&models.BlacklistEntry{}).
		Where("user_id = ?", userID).
		Where("list_type = ?", "white")

	if entryType := c.Query("type"); entryType != "" {
		q = q.Where("type = ?", entryType)
	}
	if value := c.Query("value"); value != "" {
		q = q.Where("value LIKE ?", "%"+value+"%")
	}
	if domainID := c.Query("domain_id"); domainID != "" {
		q = q.Where("domain_id = ?", domainID)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("查询白名单失败: %v", err))
		return
	}

	var list []models.BlacklistEntry
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("查询白名单失败: %v", err))
		return
	}

	ptSuccess(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ------------------------------------------------------------
// WhitelistCreate 新增白名单
// POST /api/v1/protection/whitelist
// Body:
// {
//   "type":         "ip",          // ip / url / domain
//   "value":        "1.2.3.4",     // 单条规则字符串
//   "match_mode":   "exact",       // exact / prefix / regex / cidr
//   "domain_id":    123,           // 可选,作用域名
//   "http_method":  "GET"          // 可选,空=匹配全部方法
// }
// ------------------------------------------------------------
func WhitelistCreate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := getCurrentUserID(c)

	var req struct {
		Type       string `json:"type"`
		Value      string `json:"value"`
		MatchMode  string `json:"match_mode"`
		DomainID   *uint  `json:"domain_id"`
		HTTPMethod string `json:"http_method"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ptFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	if req.Type == "" {
		req.Type = "ip"
	}
	if req.Value == "" {
		ptFail(c, 400, "value 不能为空")
		return
	}
	if req.MatchMode == "" {
		req.MatchMode = "exact"
	}

	entry := models.BlacklistEntry{
		UserID:     userID,
		DomainID:   req.DomainID,
		Type:       req.Type,
		ListType:   "white",
		Value:      req.Value,
		MatchMode:  req.MatchMode,
		HTTPMethod: req.HTTPMethod,
	}

	if err := db.Create(&entry).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("新增白名单失败: %v", err))
		return
	}

	ptSuccess(c, entry)
}

// ------------------------------------------------------------
// WhitelistDelete 删除白名单
// DELETE /api/v1/protection/whitelist/:id
// ------------------------------------------------------------
func WhitelistDelete(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := getCurrentUserID(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ptFail(c, 400, "无效的 ID")
		return
	}

	var entry models.BlacklistEntry
	if err := db.Where("id = ? AND list_type = ?", id, "white").First(&entry).Error; err != nil {
		ptFail(c, 404, "白名单条目不存在")
		return
	}

	if entry.UserID != userID && c.GetString("role") != "admin" {
		ptFail(c, 403, "无权删除他人条目")
		return
	}

	if err := db.Delete(&entry).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("删除白名单失败: %v", err))
		return
	}

	ptSuccess(c, gin.H{"id": id})
}
