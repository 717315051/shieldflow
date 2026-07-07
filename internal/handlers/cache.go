package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/models"
	"gorm.io/gorm"
)

// ============ 缓存管理 ============

// FileRefresh 文件刷新
// POST /api/v1/cache/file-refresh
func FileRefresh(c *gin.Context) {
	createCacheTask(c, "file_refresh")
}

// DirRefresh 目录刷新
// POST /api/v1/cache/dir-refresh
func DirRefresh(c *gin.Context) {
	createCacheTask(c, "dir_refresh")
}

// FilePreheat 文件预读
// POST /api/v1/cache/file-preheat
func FilePreheat(c *gin.Context) {
	createCacheTask(c, "file_preheat")
}

// createCacheTask 创建缓存任务的公共逻辑
func createCacheTask(c *gin.Context, taskType string) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var req struct {
		DomainID uint     `json:"domain_id" binding:"required"`
		URLs     []string `json:"urls" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if len(req.URLs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "URL列表不能为空"})
		return
	}
	if len(req.URLs) > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "单次任务最多1000个URL"})
		return
	}

	// 校验域名归属
	var domain models.Domain
	if err := db.Where("id = ? AND user_id = ?", req.DomainID, userID).First(&domain).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "域名不存在"})
		return
	}

	urlsJSON, _ := json.Marshal(req.URLs)
	task := models.CacheTask{
		UserID:   userID,
		DomainID: req.DomainID,
		Type:     taskType,
		URLs:     string(urlsJSON),
		Status:   "pending",
	}
	if err := db.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "创建任务失败: " + err.Error()})
		return
	}

	// TODO: 通过 gRPC 下发任务到边缘节点

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": task})
}

// ListCacheTasks 任务列表
// GET /api/v1/cache/tasks
func ListCacheTasks(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	status := c.Query("status")
	taskType := c.Query("type")
	domainID := c.Query("domain_id")

	query := db.Model(&models.CacheTask{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if taskType != "" {
		query = query.Where("type = ?", taskType)
	}
	if domainID != "" {
		query = query.Where("domain_id = ?", domainID)
	}

	var total int64
	query.Count(&total)

	page, pageSize := parsePagination(c)
	var list []models.CacheTask
	query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

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

// GetCacheTask 任务详情
// GET /api/v1/cache/tasks/:id
func GetCacheTask(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var task models.CacheTask
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": task})
}

// CancelCacheTask 取消任务
// POST /api/v1/cache/tasks/:id/cancel
func CancelCacheTask(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var task models.CacheTask
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "任务不存在"})
		return
	}

	if task.Status != "pending" && task.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "任务状态不允许取消"})
		return
	}

	if err := db.Model(&task).Update("status", "cancelled").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "取消失败: " + err.Error()})
		return
	}

	// TODO: 通过 gRPC 通知边缘节点取消任务

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": task})
}
