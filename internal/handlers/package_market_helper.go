package handlers

// ============================================================
// 套餐市场 handler（按 user-manual 章节 10.1）
// 一次返回三类: 套餐 / 流量包 / 域名包 + 用户余额
// ============================================================

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/shieldflow/shieldflow/internal/models"
)

// PackageMarketHandler 套餐市场一次性聚合接口
type PackageMarketHandler struct {
	db *gorm.DB
}

// NewPackageMarketHandler 构造
func NewPackageMarketHandler(db *gorm.DB) *PackageMarketHandler {
	return &PackageMarketHandler{db: db}
}

// ------------------------------------------------------------
// Market 套餐市场聚合数据
// GET /api/v1/packages/market
// 一次性返回三种商品 + 用户余额
// {
//   "code": 0,
//   "data": {
//     "packages":      [...],   // 七层/四层套餐
//     "traffic":       [...],   // 流量包
//     "domain":        [...],   // 域名包
//     "balance":       0.0,     // 当前余额
//     "frozen_balance":0.0
//   }
// }
// ------------------------------------------------------------
func (h *PackageMarketHandler) Market(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	uid, _ := c.Get("user_id")

	var packages []models.Package
	db.Where("status = ?", "active").Order("price ASC").Find(&packages)

	var trafficPacks []models.TrafficPackage
	db.Where("status = ?", "active").Order("price ASC").Find(&trafficPacks)

	var domainPacks []models.DomainPackage
	db.Where("status = ?", "active").Order("price ASC").Find(&domainPacks)

	balance, frozenBalance := 0.0, 0.0
	if uid != nil {
		userID, _ := uid.(uint)
		var userRow models.User
		if err := db.Select("balance, frozen_balance").Where("id = ?", userID).First(&userRow).Error; err == nil {
			balance = userRow.Balance
			frozenBalance = userRow.FrozenBalance
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"packages":       packages,
			"traffic":        trafficPacks,
			"domain":         domainPacks,
			"balance":        balance,
			"frozen_balance": frozenBalance,
			"currency":       "CNY",
		},
	})
}
