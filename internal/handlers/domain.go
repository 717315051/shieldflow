package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/models"
	"gorm.io/gorm"
)

// DomainHandler 域名管理 handler 集合
type DomainHandler struct{}

// NewDomainHandler 构造 DomainHandler
func NewDomainHandler() *DomainHandler {
	return &DomainHandler{}
}

// ListDomains GET /api/v1/domains
func (h *DomainHandler) ListDomains(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	keyword := strings.TrimSpace(c.Query("keyword"))
	status := strings.TrimSpace(c.Query("status"))

	query := db.Model(&models.Domain{})
	if role != "admin" {
		query = query.Where("user_id = ?", userID)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("domain_name LIKE ?", like)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var domains []models.Domain
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&domains).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "查询失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      domains,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// CreateDomain POST /api/v1/domains
func (h *DomainHandler) CreateDomain(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var req struct {
		DomainName  string `json:"domain_name"`
		OriginConfig string `json:"origin_config"` // JSON 字符串或结构体
		PackageID   *uint  `json:"package_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	req.DomainName = strings.TrimSpace(req.DomainName)
	if req.DomainName == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "域名不能为空", "data": nil})
		return
	}

	// 校验域名是否已存在
	var cnt int64
	db.Model(&models.Domain{}).Where("domain_name = ?", req.DomainName).Count(&cnt)
	if cnt > 0 {
		c.JSON(http.StatusOK, gin.H{"code": 409, "message": "域名已存在", "data": nil})
		return
	}

	// 检查套餐域名数量限制
	if req.PackageID != nil {
		var pkg models.Package
		if err := db.First(&pkg, *req.PackageID).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 404, "message": "套餐不存在", "data": nil})
			return
		}
		var domainCount int64
		db.Model(&models.Domain{}).Where("user_id = ? AND package_id = ?", userID, *req.PackageID).Count(&domainCount)
		if pkg.DomainLimit > 0 && int(domainCount) >= pkg.DomainLimit {
			c.JSON(http.StatusOK, gin.H{"code": 403, "message": "套餐域名数量已达上限", "data": nil})
			return
		}
	}

	originJSON := normalizeJSONString(req.OriginConfig)
	domain := models.Domain{
		UserID:       userID,
		DomainName:   req.DomainName,
		PackageID:    req.PackageID,
		CNAME:        fmt.Sprintf("%s.shieldflow.net", req.DomainName),
		Status:       "pending",
		OriginConfig: originJSON,
	}
	if err := db.Create(&domain).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "创建域名失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": domain})
}

// BatchCreateDomains POST /api/v1/domains/batch
func (h *DomainHandler) BatchCreateDomains(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var req struct {
		Data      string `json:"data"`       // 每行: 域名|源站
		PackageID *uint  `json:"package_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}

	lines := strings.Split(req.Data, "\n")
	successCount := 0
	var errs []string
	for idx, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		domainName := strings.TrimSpace(parts[0])
		var originConfig string
		if len(parts) > 1 {
			origin := strings.TrimSpace(parts[1])
			originConfig = fmt.Sprintf(`{"origin":"%s"}`, origin)
		}
		if domainName == "" {
			errs = append(errs, fmt.Sprintf("第 %d 行: 域名为空", idx+1))
			continue
		}
		var cnt int64
		db.Model(&models.Domain{}).Where("domain_name = ?", domainName).Count(&cnt)
		if cnt > 0 {
			errs = append(errs, fmt.Sprintf("第 %d 行: 域名 %s 已存在", idx+1, domainName))
			continue
		}
		domain := models.Domain{
			UserID:       userID,
			DomainName:   domainName,
			PackageID:    req.PackageID,
			CNAME:        fmt.Sprintf("%s.shieldflow.net", domainName),
			Status:       "pending",
			OriginConfig: originConfig,
		}
		if err := db.Create(&domain).Error; err != nil {
			errs = append(errs, fmt.Sprintf("第 %d 行: %s 创建失败: %v", idx+1, domainName, err))
			continue
		}
		successCount++
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"success_count": successCount,
			"errors":        errs,
		},
	})
}

// GetDomain GET /api/v1/domains/:id
func (h *DomainHandler) GetDomain(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的域名 ID", "data": nil})
		return
	}

	var domain models.Domain
	if err := db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "域名不存在", "data": nil})
		return
	}
	if role != "admin" && domain.UserID != userID {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "无权访问", "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": domain})
}

// UpdateDomain PUT /api/v1/domains/:id
func (h *DomainHandler) UpdateDomain(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的域名 ID", "data": nil})
		return
	}

	var domain models.Domain
	if err := db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "域名不存在", "data": nil})
		return
	}
	if role != "admin" && domain.UserID != userID {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "无权访问", "data": nil})
		return
	}

	var req struct {
		DomainName string `json:"domain_name"`
		PackageID  *uint  `json:"package_id"`
		Status     string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	updates := map[string]interface{}{}
	if req.DomainName != "" {
		updates["domain_name"] = req.DomainName
	}
	if req.PackageID != nil {
		updates["package_id"] = req.PackageID
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if err := db.Model(&domain).Updates(updates).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "更新失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// DeleteDomain DELETE /api/v1/domains/:id
func (h *DomainHandler) DeleteDomain(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的域名 ID", "data": nil})
		return
	}
	var domain models.Domain
	if err := db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "域名不存在", "data": nil})
		return
	}
	if role != "admin" && domain.UserID != userID {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "无权访问", "data": nil})
		return
	}
	if err := db.Delete(&domain).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "删除失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// UpdateDomainStatus PUT /api/v1/domains/:id/status
func (h *DomainHandler) UpdateDomainStatus(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的域名 ID", "data": nil})
		return
	}
	var domain models.Domain
	if err := db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "域名不存在", "data": nil})
		return
	}
	if role != "admin" && domain.UserID != userID {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "无权访问", "data": nil})
		return
	}
	var req struct {
		Status string `json:"status"` // active / disabled / pending
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	if err := db.Model(&domain).Update("status", req.Status).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "更新失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// ChangeDomainPackage PUT /api/v1/domains/:id/package
func (h *DomainHandler) ChangeDomainPackage(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的域名 ID", "data": nil})
		return
	}
	var domain models.Domain
	if err := db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "域名不存在", "data": nil})
		return
	}
	if role != "admin" && domain.UserID != userID {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "无权访问", "data": nil})
		return
	}
	var req struct {
		PackageID uint `json:"package_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	var pkg models.Package
	if err := db.First(&pkg, req.PackageID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "套餐不存在", "data": nil})
		return
	}
	if err := db.Model(&domain).Update("package_id", req.PackageID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "更新失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// GetDomainConfig GET /api/v1/domains/:id/config
func (h *DomainHandler) GetDomainConfig(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的域名 ID", "data": nil})
		return
	}
	var domain models.Domain
	if err := db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "域名不存在", "data": nil})
		return
	}
	if role != "admin" && domain.UserID != userID {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "无权访问", "data": nil})
		return
	}

	config := gin.H{
		"origin_config":     parseJSONField(domain.OriginConfig),
		"https_config":      parseJSONField(domain.HTTPSConfig),
		"cache_config":      parseJSONField(domain.CacheConfig),
		"advanced_config":   parseJSONField(domain.AdvancedConfig),
		"custom_headers":    parseJSONField(domain.CustomHeaders),
		"custom_pages":      parseJSONField(domain.CustomPages),
		"protection_config": parseJSONField(domain.ProtectionConfig),
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": config})
}

// SaveDomainBasic PUT /api/v1/domains/:id/basic
func (h *DomainHandler) SaveDomainBasic(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的域名 ID", "data": nil})
		return
	}
	var domain models.Domain
	if err := db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "域名不存在", "data": nil})
		return
	}
	if role != "admin" && domain.UserID != userID {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "无权访问", "data": nil})
		return
	}

	var req map[string]json.RawMessage
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}

	updates := map[string]interface{}{}
	if v, ok := req["origin_config"]; ok {
		updates["origin_config"] = string(v)
	}
	if v, ok := req["https_config"]; ok {
		updates["https_config"] = string(v)
	}
	if v, ok := req["cache_config"]; ok {
		updates["cache_config"] = string(v)
	}
	if v, ok := req["advanced_config"]; ok {
		updates["advanced_config"] = string(v)
	}
	if v, ok := req["custom_headers"]; ok {
		updates["custom_headers"] = string(v)
	}
	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
		return
	}
	if err := db.Model(&domain).Updates(updates).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "保存失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// SaveDomainProtection PUT /api/v1/domains/:id/protection
func (h *DomainHandler) SaveDomainProtection(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的域名 ID", "data": nil})
		return
	}
	var domain models.Domain
	if err := db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "域名不存在", "data": nil})
		return
	}
	if role != "admin" && domain.UserID != userID {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "无权访问", "data": nil})
		return
	}

	var req map[string]json.RawMessage
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}

	protectionConfig := "{}"
	if v, ok := req["protection_config"]; ok {
		protectionConfig = string(v)
	} else {
		// 整个 body 视为 protection_config
		bodyBytes, _ := c.GetRawData()
		if len(bodyBytes) > 0 {
			protectionConfig = string(bodyBytes)
		}
	}
	if err := db.Model(&domain).Update("protection_config", protectionConfig).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "保存失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// SaveDomainCustomPages PUT /api/v1/domains/:id/custom-pages
func (h *DomainHandler) SaveDomainCustomPages(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的域名 ID", "data": nil})
		return
	}
	var domain models.Domain
	if err := db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "域名不存在", "data": nil})
		return
	}
	if role != "admin" && domain.UserID != userID {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "无权访问", "data": nil})
		return
	}

	bodyBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "读取请求体失败: " + err.Error(), "data": nil})
		return
	}
	customPages := string(bodyBytes)
	if customPages == "" {
		customPages = "{}"
	}
	if err := db.Model(&domain).Update("custom_pages", customPages).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "保存失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// ApplyCertificate POST /api/v1/domains/:id/certificate
func (h *DomainHandler) ApplyCertificate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "无效的域名 ID", "data": nil})
		return
	}
	var domain models.Domain
	if err := db.First(&domain, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "域名不存在", "data": nil})
		return
	}
	if role != "admin" && domain.UserID != userID {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "无权访问", "data": nil})
		return
	}

	// TODO: 接入 ACME 客户端实际申请证书
	certReq := models.CertificateRequest{
		UserID:     userID,
		Domain:     domain.DomainName,
		VerifyType: "http",
		Status:     "pending",
	}
	if err := db.Create(&certReq).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "创建证书申请记录失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    gin.H{"request_id": certReq.ID, "status": "pending"},
	})
}

// BatchApplyCertificate POST /api/v1/domains/batch-certificate
func (h *DomainHandler) BatchApplyCertificate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)
	role := c.MustGet("role").(string)

	var req struct {
		DomainIDs []uint `json:"domain_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}

	successCount := 0
	var errs []string
	for _, did := range req.DomainIDs {
		var domain models.Domain
		if err := db.First(&domain, did).Error; err != nil {
			errs = append(errs, fmt.Sprintf("域名 ID %d 不存在", did))
			continue
		}
		if role != "admin" && domain.UserID != userID {
			errs = append(errs, fmt.Sprintf("域名 %s 无权操作", domain.DomainName))
			continue
		}
		certReq := models.CertificateRequest{
			UserID:     userID,
			Domain:     domain.DomainName,
			VerifyType: "http",
			Status:     "pending",
		}
		if err := db.Create(&certReq).Error; err != nil {
			errs = append(errs, fmt.Sprintf("域名 %s 申请记录创建失败: %v", domain.DomainName, err))
			continue
		}
		successCount++
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"success_count": successCount,
			"errors":        errs,
		},
	})
}

// --- 辅助函数 ---

// normalizeJSONString 规范化 JSON 字符串
func normalizeJSONString(s string) string {
	if s == "" {
		return "{}"
	}
	var dst interface{}
	if err := json.Unmarshal([]byte(s), &dst); err != nil {
		// 不是合法 JSON，包裹一下
		return fmt.Sprintf(`{"raw":%s}`, strconvQuote(s))
	}
	b, _ := json.Marshal(dst)
	return string(b)
}

// parseJSONField 解析 JSON 字符串字段为 interface{}
func parseJSONField(s string) interface{} {
	if s == "" {
		return nil
	}
	var dst interface{}
	if err := json.Unmarshal([]byte(s), &dst); err != nil {
		return s
	}
	return dst
}

// strconvQuote 使用 strconv.Quote
func strconvQuote(s string) string {
	// 直接用 json.Marshal 得到带引号的字符串
	b, _ := json.Marshal(s)
	return string(b)
}

// 避免 time 未使用告警
var _ = time.Now
