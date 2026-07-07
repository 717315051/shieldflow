package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/models"
	"gorm.io/gorm"
)

// ============ 四层转发管理 ============

// ListLayer4 四层转发列表
// GET /api/v1/layer4
func ListLayer4(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	status := c.Query("status")
	protocol := c.Query("protocol")

	query := db.Model(&models.Layer4Forward{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if protocol != "" {
		query = query.Where("protocol = ?", protocol)
	}

	var total int64
	query.Count(&total)

	page, pageSize := parsePagination(c)
	var list []models.Layer4Forward
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

// CreateLayer4 添加转发规则
// POST /api/v1/layer4
func CreateLayer4(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var fwd models.Layer4Forward
	if err := c.ShouldBindJSON(&fwd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}

	if fwd.Protocol != "tcp" && fwd.Protocol != "udp" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "协议必须为 tcp 或 udp"})
		return
	}
	if fwd.ListenPort < 1 || fwd.ListenPort > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "监听端口范围 1-65535"})
		return
	}
	if fwd.Origins == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "源站列表不能为空"})
		return
	}
	if fwd.LBStrategy == "" {
		fwd.LBStrategy = "round_robin"
	}
	if fwd.Status == "" {
		fwd.Status = "active"
	}

	// 校验端口冲突 (同一用户下)
	var count int64
	db.Model(&models.Layer4Forward{}).Where("user_id = ? AND listen_port = ? AND protocol = ?", userID, fwd.ListenPort, fwd.Protocol).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "该协议下监听端口已被占用"})
		return
	}

	fwd.UserID = userID
	if err := db.Create(&fwd).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "创建失败: " + err.Error()})
		return
	}

	// TODO: 通过 gRPC 下发配置到边缘节点

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": fwd})
}

// UpdateLayer4 编辑转发规则
// PUT /api/v1/layer4/:id
func UpdateLayer4(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var fwd models.Layer4Forward
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&fwd).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "转发规则不存在"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}

	// 不允许修改归属字段
	delete(req, "id")
	delete(req, "user_id")
	delete(req, "created_at")

	// 校验协议和端口
	if proto, ok := req["protocol"].(string); ok && proto != "" {
		if proto != "tcp" && proto != "udp" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "协议必须为 tcp 或 udp"})
			return
		}
	}
	if port, ok := req["listen_port"].(float64); ok {
		if port < 1 || port > 65535 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "监听端口范围 1-65535"})
			return
		}
		// 端口冲突校验 (排除自身)
		var count int64
		db.Model(&models.Layer4Forward{}).Where("user_id = ? AND listen_port = ? AND id != ?", userID, int(port), id).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "监听端口已被占用"})
			return
		}
	}

	if err := db.Model(&fwd).Updates(req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "更新失败: " + err.Error()})
		return
	}

	// TODO: 通过 gRPC 下发更新到边缘节点

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": fwd})
}

// DeleteLayer4 删除转发规则
// DELETE /api/v1/layer4/:id
func DeleteLayer4(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var fwd models.Layer4Forward
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&fwd).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "转发规则不存在"})
		return
	}

	if err := db.Delete(&fwd).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "删除失败: " + err.Error()})
		return
	}

	// TODO: 通过 gRPC 通知边缘节点移除配置

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// UpdateLayer4Status 启用/禁用
// PUT /api/v1/layer4/:id/status
func UpdateLayer4Status(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if req.Status != "active" && req.Status != "disabled" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "状态必须为 active 或 disabled"})
		return
	}

	var fwd models.Layer4Forward
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&fwd).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "转发规则不存在"})
		return
	}

	if err := db.Model(&fwd).Update("status", req.Status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "更新状态失败: " + err.Error()})
		return
	}

	// TODO: 通过 gRPC 通知边缘节点更新状态

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": fwd})
}
