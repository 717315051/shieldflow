package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserHandler 用户管理(admin) handler 集合
type UserHandler struct{}

// NewUserHandler 构造 UserHandler
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// ListUsers GET /api/v1/admin/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	keyword := strings.TrimSpace(c.Query("keyword"))
	role := strings.TrimSpace(c.Query("role"))
	status := strings.TrimSpace(c.Query("status"))

	query := db.Model(&models.User{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR email LIKE ? OR phone LIKE ?", like, like, like)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var users []models.User
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "查询失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      users,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// CreateUser POST /api/v1/admin/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Status   string `json:"status"`
		Balance  float64 `json:"balance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "用户名和密码不能为空", "data": nil})
		return
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "密码长度至少 6 位", "data": nil})
		return
	}

	var cnt int64
	db.Model(&models.User{}).Where("username = ? OR email = ?", req.Username, req.Email).Count(&cnt)
	if cnt > 0 {
		c.JSON(http.StatusOK, gin.H{"code": 409, "message": "用户名或邮箱已存在", "data": nil})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "密码加密失败: " + err.Error(), "data": nil})
		return
	}

	role := req.Role
	if role == "" {
		role = "user"
	}
	status := req.Status
	if status == "" {
		status = "active"
	}

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: string(hash),
		Role:         role,
		Status:       status,
		Balance:      req.Balance,
	}
	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "创建用户失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": user})
}

// UpdateUser PUT /api/v1/admin/users/:id
func (h *UserHandler) UpdateUser(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的用户 ID", "data": nil})
		return
	}

	var req struct {
		Username string  `json:"username"`
		Email    string  `json:"email"`
		Phone    string  `json:"phone"`
		Role     string  `json:"role"`
		Status   string  `json:"status"`
		Balance  float64 `json:"balance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}

	updates := map[string]interface{}{}
	if req.Username != "" {
		var cnt int64
		db.Model(&models.User{}).Where("username = ? AND id <> ?", req.Username, id).Count(&cnt)
		if cnt > 0 {
			c.JSON(http.StatusOK, gin.H{"code": 409, "message": "用户名已存在", "data": nil})
			return
		}
		updates["username"] = req.Username
	}
	if req.Email != "" {
		var cnt int64
		db.Model(&models.User{}).Where("email = ? AND id <> ?", req.Email, id).Count(&cnt)
		if cnt > 0 {
			c.JSON(http.StatusOK, gin.H{"code": 409, "message": "邮箱已存在", "data": nil})
			return
		}
		updates["email"] = req.Email
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	updates["balance"] = req.Balance

	if err := db.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "更新失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// DeleteUser DELETE /api/v1/admin/users/:id
func (h *UserHandler) DeleteUser(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的用户 ID", "data": nil})
		return
	}
	if id == 1 {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "不允许删除超级管理员", "data": nil})
		return
	}
	if err := db.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "删除失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// UpdateUserStatus PUT /api/v1/admin/users/:id/status
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的用户 ID", "data": nil})
		return
	}

	var req struct {
		Status string `json:"status"` // active / disabled
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	if req.Status != "active" && req.Status != "disabled" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "status 取值无效", "data": nil})
		return
	}
	if err := db.Model(&models.User{}).Where("id = ?", id).Update("status", req.Status).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "更新失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// ListUserPackages GET /api/v1/admin/users/:id/packages
func (h *UserHandler) ListUserPackages(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的用户 ID", "data": nil})
		return
	}

	var userPackages []models.UserPackage
	if err := db.Where("user_id = ?", id).Order("id DESC").Find(&userPackages).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "查询失败: " + err.Error(), "data": nil})
		return
	}

	// 附带套餐信息
	type UserPackageWithPackage struct {
		models.UserPackage
		Package models.Package `json:"package"`
	}
	result := make([]UserPackageWithPackage, 0, len(userPackages))
	for _, up := range userPackages {
		var pkg models.Package
		db.First(&pkg, up.PackageID)
		result = append(result, UserPackageWithPackage{
			UserPackage: up,
			Package:     pkg,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// 避免 time 引用未使用告警
var _ = time.Now
