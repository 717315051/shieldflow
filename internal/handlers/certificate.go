package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/models"
	"gorm.io/gorm"
)

// ============ 证书管理 ============

// ListCertificates 证书列表
// GET /api/v1/certificates
func ListCertificates(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	status := c.Query("status")
	domainID := c.Query("domain_id")

	query := db.Model(&models.Certificate{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if domainID != "" {
		query = query.Where("domain_id = ?", domainID)
	}

	var total int64
	query.Count(&total)

	page, pageSize := parsePagination(c)
	var list []models.Certificate
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

// UploadCertificate 上传证书 PEM+KEY
// POST /api/v1/certificates/upload
func UploadCertificate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var req struct {
		DomainID  *uint  `json:"domain_id"`
		CertPEM   string `json:"cert_pem" binding:"required"`
		KeyPEM    string `json:"key_pem" binding:"required"`
		CommonName string `json:"common_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}

	cert := models.Certificate{
		UserID:     userID,
		DomainID:   req.DomainID,
		CertPEM:    req.CertPEM,
		KeyPEM:     req.KeyPEM,
		CommonName: req.CommonName,
		Issuer:     "manual",
		Status:     "active",
	}
	if err := db.Create(&cert).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "上传失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": cert})
}

// GetCertificate 证书详情
// GET /api/v1/certificates/:id
func GetCertificate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var cert models.Certificate
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&cert).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "证书不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": cert})
}

// DeleteCertificate 删除证书
// DELETE /api/v1/certificates/:id
func DeleteCertificate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	if err := db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Certificate{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// DownloadCertificate 下载证书
// GET /api/v1/certificates/:id/download
func DownloadCertificate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var cert models.Certificate
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&cert).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "证书不存在"})
		return
	}

	format := c.DefaultQuery("format", "pem")
	switch format {
	case "key":
		c.Header("Content-Disposition", `attachment; filename="private.key"`)
		c.String(http.StatusOK, "%s", cert.KeyPEM)
	case "pem", "":
		c.Header("Content-Disposition", `attachment; filename="cert.pem"`)
		c.String(http.StatusOK, "%s", cert.CertPEM)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "不支持的格式"})
	}
}

// ============ 证书申请记录 ============

// ListCertificateRequests 证书申请记录
// GET /api/v1/certificates/requests
func ListCertificateRequests(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	status := c.Query("status")
	query := db.Model(&models.CertificateRequest{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	page, pageSize := parsePagination(c)
	var list []models.CertificateRequest
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

// ApplyCertificate 通过ACME申请证书
// POST /api/v1/certificates/apply
func ApplyCertificate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var req struct {
		Domain       string `json:"domain" binding:"required"`
		VerifyType   string `json:"verify_type" binding:"required"` // http / dns
		AcmeAccountID *uint `json:"acme_account_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if req.VerifyType != "http" && req.VerifyType != "dns" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "验证方式必须为 http 或 dns"})
		return
	}

	cr := models.CertificateRequest{
		UserID:        userID,
		Domain:        req.Domain,
		VerifyType:    req.VerifyType,
		AcmeAccountID: req.AcmeAccountID,
		Status:        "pending",
	}
	if err := db.Create(&cr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "创建申请记录失败: " + err.Error()})
		return
	}

	// TODO: 调用 ACME 服务异步执行证书申请流程
	// 目前仅创建记录, 实际申请由后台 worker 处理

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": cr})
}

// GetCertificateRequest 申请详情
// GET /api/v1/certificates/requests/:id
func GetCertificateRequest(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var cr models.CertificateRequest
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&cr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "申请记录不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": cr})
}

// GetCertificateRequestLog 申请日志
// GET /api/v1/certificates/requests/:id/log
func GetCertificateRequestLog(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	var cr models.CertificateRequest
	if err := db.Select("error_log, status, updated_at").Where("id = ? AND user_id = ?", id, userID).First(&cr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "申请记录不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"status":    cr.Status,
			"error_log": cr.ErrorLog,
			"updated_at": cr.UpdatedAt,
		},
	})
}

// DeleteCertificateRequest 删除申请记录
// DELETE /api/v1/certificates/requests/:id
func DeleteCertificateRequest(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的ID"})
		return
	}

	if err := db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CertificateRequest{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

// ============ ACME 账户 ============

// ListAcmeAccounts ACME账户列表
// GET /api/v1/certificates/acme-accounts
func ListAcmeAccounts(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var list []models.AcmeAccount
	db.Where("user_id = ?", userID).Order("id DESC").Find(&list)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// CreateAcmeAccount 添加ACME账户
// POST /api/v1/certificates/acme-accounts
func CreateAcmeAccount(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var req struct {
		Email string `json:"email" binding:"required,email"`
		CAURL string `json:"ca_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}
	if req.CAURL == "" {
		req.CAURL = "https://acme-v02.api.letsencrypt.org/directory"
	}

	// TODO: 实际注册 ACME 账户并生成私钥
	account := models.AcmeAccount{
		UserID: userID,
		Email:  req.Email,
		CAURL:  req.CAURL,
		Key:    "", // 由 ACME 注册流程填充
	}
	if err := db.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": account})
}

// ============ DNS 账户 ============

// ListDNSAccounts DNS账户列表
// GET /api/v1/certificates/dns-accounts
func ListDNSAccounts(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var list []models.DNSAccount
	db.Where("user_id = ?", userID).Order("id DESC").Find(&list)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

// CreateDNSAccount 添加DNS账户
// POST /api/v1/certificates/dns-accounts
func CreateDNSAccount(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var req struct {
		Provider  string `json:"provider" binding:"required"`
		APIKey    string `json:"api_key" binding:"required"`
		APISecret string `json:"api_secret" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "参数错误: " + err.Error()})
		return
	}

	validProviders := map[string]bool{
		"cloudflare": true,
		"aliyun":     true,
		"tencent":    true,
		"dnspod":     true,
	}
	if !validProviders[req.Provider] {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "不支持的 DNS 服务商"})
		return
	}

	account := models.DNSAccount{
		UserID:    userID,
		Provider:  req.Provider,
		APIKey:    req.APIKey,
		APISecret: req.APISecret,
	}
	if err := db.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "创建失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": account})
}

// 避免未使用导入
var _ = time.Now
