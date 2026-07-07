package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shieldflow/shieldflow/internal/models"
	"gorm.io/gorm"
)

// PackageHandler 套餐管理 handler 集合
type PackageHandler struct {
	db *gorm.DB
}

// NewPackageHandler 构造 PackageHandler
func NewPackageHandler(db *gorm.DB) *PackageHandler {
	return &PackageHandler{db: db}
}

// ============ 用户端 ============

// List 套餐市场: 七层/四层套餐列表
// GET /api/v1/packages
func (h *PackageHandler) List(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	pkgType := c.Query("type") // l7 / l4, 可选过滤

	query := db.Model(&models.Package{}).Where("status = ?", "active")
	if pkgType != "" {
		query = query.Where("type = ?", pkgType)
	}

	var total int64
	query.Count(&total)

	page, pageSize := parsePagination(c)
	var packages []models.Package
	query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&packages)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      packages,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// TrafficPackages 流量包列表
// GET /api/v1/packages/traffic
func (h *PackageHandler) TrafficPackages(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var total int64
	db.Model(&models.TrafficPackage{}).Where("status = ?", "active").Count(&total)

	page, pageSize := parsePagination(c)
	var packages []models.TrafficPackage
	db.Where("status = ?", "active").Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&packages)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      packages,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// DomainPackages 域名包列表
// GET /api/v1/packages/domain
func (h *PackageHandler) DomainPackages(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var total int64
	db.Model(&models.DomainPackage{}).Where("status = ?", "active").Count(&total)

	page, pageSize := parsePagination(c)
	var packages []models.DomainPackage
	db.Where("status = ?", "active").Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&packages)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      packages,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// Purchase 购买套餐
// POST /api/v1/packages/:id/purchase
func (h *PackageHandler) Purchase(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的套餐ID"})
		return
	}

	var pkg models.Package
	if err := db.First(&pkg, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "套餐不存在"})
		return
	}
	if pkg.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "套餐已下架"})
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "用户不存在"})
		return
	}
	if user.Balance < pkg.Price {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "余额不足"})
		return
	}

	// 事务: 扣余额 + 创建实例 + 创建订单
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("balance", gorm.Expr("balance - ?", pkg.Price)).Error; err != nil {
			return err
		}

		expiresAt := time.Now().AddDate(0, 0, pkg.DurationDays)
		up := models.UserPackage{
			UserID:       userID,
			PackageID:    pkg.ID,
			InstanceNo:   generateOrderNo(),
			TrafficLimit: pkg.TrafficLimit,
			Status:       "active",
			ExpiresAt:    &expiresAt,
		}
		if err := tx.Create(&up).Error; err != nil {
			return err
		}

		order := models.Order{
			UserID:      userID,
			OrderNo:     generateOrderNo(),
			ProductType: "package",
			ProductName: pkg.Name,
			Amount:      pkg.Price,
			Channel:     "balance",
			Status:      "paid",
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "购买失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"status": "paid"}})
}

// PurchaseTraffic 购买流量包
// POST /api/v1/packages/traffic/:id/purchase
func (h *PackageHandler) PurchaseTraffic(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的流量包ID"})
		return
	}

	var tp models.TrafficPackage
	if err := db.First(&tp, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "流量包不存在"})
		return
	}
	if tp.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "流量包已下架"})
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "用户不存在"})
		return
	}
	if user.Balance < tp.Price {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "余额不足"})
		return
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("balance", gorm.Expr("balance - ?", tp.Price)).Error; err != nil {
			return err
		}
		order := models.Order{
			UserID:      userID,
			OrderNo:     generateOrderNo(),
			ProductType: "traffic",
			ProductName: tp.Name,
			Amount:      tp.Price,
			Channel:     "balance",
			Status:      "paid",
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "购买失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"status": "paid"}})
}

// PurchaseDomain 购买域名包
// POST /api/v1/packages/domain/:id/purchase
func (h *PackageHandler) PurchaseDomain(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的域名包ID"})
		return
	}

	var dp models.DomainPackage
	if err := db.First(&dp, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "域名包不存在"})
		return
	}
	if dp.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "域名包已下架"})
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "用户不存在"})
		return
	}
	if user.Balance < dp.Price {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "余额不足"})
		return
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("balance", gorm.Expr("balance - ?", dp.Price)).Error; err != nil {
			return err
		}
		order := models.Order{
			UserID:      userID,
			OrderNo:     generateOrderNo(),
			ProductType: "domain",
			ProductName: dp.Name,
			Amount:      dp.Price,
			Channel:     "balance",
			Status:      "paid",
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "购买失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"status": "paid"}})
}

// MyPackages 我的套餐列表
// GET /api/v1/user-packages
func (h *PackageHandler) MyPackages(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	status := c.Query("status")

	query := db.Model(&models.UserPackage{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	page, pageSize := parsePagination(c)
	var list []models.UserPackage
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

// PackageDetail 套餐用量详情
// GET /api/v1/user-packages/:id
func (h *PackageHandler) PackageDetail(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var up models.UserPackage
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&up).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "套餐实例不存在"})
		return
	}

	var pkg models.Package
	db.First(&pkg, up.PackageID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"package":       up,
			"package_info":  pkg,
			"traffic_used":  up.TrafficUsed,
			"traffic_limit": up.TrafficLimit,
			"usage_percent": trafficPercent(up.TrafficUsed, up.TrafficLimit),
		},
	})
}

// Renew 续费
// POST /api/v1/user-packages/:id/renew
func (h *PackageHandler) Renew(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var up models.UserPackage
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&up).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "套餐实例不存在"})
		return
	}

	var pkg models.Package
	if err := db.First(&pkg, up.PackageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "套餐不存在"})
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "用户不存在"})
		return
	}
	if user.Balance < pkg.Price {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "余额不足"})
		return
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("balance", gorm.Expr("balance - ?", pkg.Price)).Error; err != nil {
			return err
		}
		// 续费: 在原到期时间基础上延长, 若已过期则从当前时间开始
		base := time.Now()
		if up.ExpiresAt != nil && up.ExpiresAt.After(base) {
			base = *up.ExpiresAt
		}
		newExpires := base.AddDate(0, 0, pkg.DurationDays)
		if err := tx.Model(&up).Updates(map[string]interface{}{
			"expires_at": newExpires,
			"status":     "active",
		}).Error; err != nil {
			return err
		}
		order := models.Order{
			UserID:      userID,
			OrderNo:     generateOrderNo(),
			ProductType: "package",
			ProductName: pkg.Name + "(续费)",
			Amount:      pkg.Price,
			Channel:     "balance",
			Status:      "paid",
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "续费失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"status": "paid"}})
}

// Orders 购买记录
// GET /api/v1/orders
func (h *PackageHandler) Orders(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	status := c.Query("status")
	productType := c.Query("product_type")

	query := db.Model(&models.Order{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if productType != "" {
		query = query.Where("product_type = ?", productType)
	}

	var total int64
	query.Count(&total)

	page, pageSize := parsePagination(c)
	var list []models.Order
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

// OrderDetail 订单详情
// GET /api/v1/orders/:id
func (h *PackageHandler) OrderDetail(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var order models.Order
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "订单不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": order})
}

// Balance 余额信息
// GET /api/v1/balance
func (h *PackageHandler) Balance(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var user models.User
	if err := db.Select("balance, frozen_balance").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"balance":        user.Balance,
			"frozen_balance": user.FrozenBalance,
			"available":      user.Balance - user.FrozenBalance,
		},
	})
}

// Recharge 充值
// POST /api/v1/balance/recharge
func (h *PackageHandler) Recharge(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var req struct {
		Amount  float64 `json:"amount" binding:"required,gt=0"`
		Channel string  `json:"channel"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if req.Channel == "" {
		req.Channel = "manual"
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", userID).
			Update("balance", gorm.Expr("balance + ?", req.Amount)).Error; err != nil {
			return err
		}
		order := models.Order{
			UserID:      userID,
			OrderNo:     generateOrderNo(),
			ProductType: "recharge",
			ProductName: "余额充值",
			Amount:      req.Amount,
			Channel:     req.Channel,
			Status:      "paid",
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "充值失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"status": "paid"}})
}

// ============ 管理端 ============

// AdminList 套餐列表
// GET /api/v1/admin/packages
func (h *PackageHandler) AdminList(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	pkgType := c.Query("type")
	query := db.Model(&models.Package{})
	if pkgType != "" {
		query = query.Where("type = ?", pkgType)
	}

	var total int64
	query.Count(&total)

	page, pageSize := parsePagination(c)
	var list []models.Package
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

// AdminCreate 创建套餐
// POST /api/v1/admin/packages
func (h *PackageHandler) AdminCreate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var pkg models.Package
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if pkg.Type != "l7" && pkg.Type != "l4" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "套餐类型必须为 l7 或 l4"})
		return
	}
	if pkg.Status == "" {
		pkg.Status = "active"
	}

	if err := db.Create(&pkg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": pkg})
}

// AdminUpdate 编辑套餐
// PUT /api/v1/admin/packages/:id
func (h *PackageHandler) AdminUpdate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var pkg models.Package
	if err := db.First(&pkg, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "套餐不存在"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	// 不允许修改 ID
	delete(req, "id")

	if err := db.Model(&pkg).Updates(req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "更新失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": pkg})
}

// AdminDelete 删除套餐
// DELETE /api/v1/admin/packages/:id
func (h *PackageHandler) AdminDelete(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	if err := db.Delete(&models.Package{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// AdminCreateTraffic 创建流量包
// POST /api/v1/admin/packages/traffic
func (h *PackageHandler) AdminCreateTraffic(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var tp models.TrafficPackage
	if err := c.ShouldBindJSON(&tp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if tp.Status == "" {
		tp.Status = "active"
	}

	if err := db.Create(&tp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": tp})
}

// AdminCreateDomain 创建域名包
// POST /api/v1/admin/packages/domain
func (h *PackageHandler) AdminCreateDomain(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var dp models.DomainPackage
	if err := c.ShouldBindJSON(&dp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if dp.Status == "" {
		dp.Status = "active"
	}

	if err := db.Create(&dp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": dp})
}

// ============ 工具函数 (包级别共享) ============

// parsePagination 解析分页参数, 返回 page 和 page_size
func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 20
	}
	return page, pageSize
}

// generateOrderNo 生成订单号
func generateOrderNo() string {
	return fmt.Sprintf("ZY%s", uuid.New().String()[:16])
}

// trafficPercent 计算流量使用百分比
func trafficPercent(used, limit int64) float64 {
	if limit <= 0 {
		return 0
	}
	return float64(used) / float64(limit) * 100
}

// atoi 简易字符串转整数 (用于无 error 返回的便捷场景)
func atoi(s string) (int, error) {
	return strconv.Atoi(s)
}
