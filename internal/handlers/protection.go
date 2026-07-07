package handlers

import (
	"encoding/csv"
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

// ============================================================
// 防护管理 Handler
// ============================================================
//
// 路由清单:
//   GET    /api/v1/protection-templates
//   POST   /api/v1/protection-templates
//   PUT    /api/v1/protection-templates/:id
//   DELETE /api/v1/protection-templates/:id
//   POST   /api/v1/protection-templates/:id/apply
//   GET    /api/v1/protection-templates/system
//   POST   /api/v1/protection-templates/system/:id/apply
//   GET    /api/v1/blacklists
//   POST   /api/v1/blacklists
//   DELETE /api/v1/blacklists/:id
//   POST   /api/v1/blacklists/import
//   GET    /api/v1/blacklists/export
//
// 统一响应格式: gin.H{"code": 0, "message": "success", "data": ...}

// ------------------------------------------------------------
// 辅助函数
// ------------------------------------------------------------

func ptSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data})
}

func ptFail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": code, "message": msg, "data": nil})
}

func ptPagination(c *gin.Context) (page, pageSize, offset int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 20
	}
	offset = (page - 1) * pageSize
	return
}

// getCurrentUserID 从上下文获取当前用户 ID(若为 admin 接口可缺省)
func getCurrentUserID(c *gin.Context) uint {
	if v, ok := c.Get("user_id"); ok {
		if uid, ok := v.(uint); ok {
			return uid
		}
	}
	return 0
}

// ------------------------------------------------------------
// ProtectionTemplateList 防护策略模板列表(我的模板 + 系统模板)
// GET /api/v1/protection-templates
// ------------------------------------------------------------
func ProtectionTemplateList(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := getCurrentUserID(c)
	page, pageSize, offset := ptPagination(c)

	q := db.Model(&models.ProtectionTemplate{}).
		Where("user_id = ? OR is_system = ?", userID, true)

	if name := c.Query("name"); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}

	var total int64
	q.Count(&total)

	var list []models.ProtectionTemplate
	if err := q.Order("is_system DESC, id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("查询模板失败: %v", err))
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
// ProtectionTemplateCreate 创建防护策略模板
// POST /api/v1/protection-templates
// Body 示例:
// {
//   "name": "标准防护",
//   "description": "...",
//   "is_default": false,
//   "config": {
//     "cc": {...},
//     "waf": {...},
//     "bot": {...},
//     "region": {...},
//     "access_control": {...}
//   }
// }
// ------------------------------------------------------------
func ProtectionTemplateCreate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := getCurrentUserID(c)

	var tpl models.ProtectionTemplate
	if err := c.ShouldBindJSON(&tpl); err != nil {
		ptFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	if tpl.Name == "" {
		ptFail(c, 400, "模板名称不能为空")
		return
	}

	// 序列化 config 为 JSON 字符串(若传入的是对象)
	if c.Request.ContentLength > 0 {
		var raw map[string]interface{}
		_ = c.ShouldBindJSON(&raw)
		if cfgVal, ok := raw["config"]; ok {
			if b, err := json.Marshal(cfgVal); err == nil {
				tpl.Config = string(b)
			}
		}
	}

	// 再次绑定基础字段
	_ = c.ShouldBindJSON(&tpl)

	tpl.UserID = userID
	tpl.IsSystem = false

	if err := db.Create(&tpl).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("创建模板失败: %v", err))
		return
	}

	ptSuccess(c, tpl)
}

// ------------------------------------------------------------
// ProtectionTemplateUpdate 编辑防护策略模板
// PUT /api/v1/protection-templates/:id
// ------------------------------------------------------------
func ProtectionTemplateUpdate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := getCurrentUserID(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ptFail(c, 400, "无效的 ID")
		return
	}

	var tpl models.ProtectionTemplate
	if err := db.First(&tpl, id).Error; err != nil {
		ptFail(c, 404, "模板不存在")
		return
	}

	// 系统模板不允许非管理员修改
	if tpl.IsSystem && c.GetString("role") != "admin" {
		ptFail(c, 403, "系统模板仅管理员可修改")
		return
	}

	// 非系统模板仅本人可修改
	if !tpl.IsSystem && tpl.UserID != userID {
		ptFail(c, 403, "无权修改他人模板")
		return
	}

	var updates models.ProtectionTemplate
	if err := c.ShouldBindJSON(&updates); err != nil {
		ptFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	updates.ID = tpl.ID
	updates.UserID = tpl.UserID
	updates.IsSystem = tpl.IsSystem

	if err := db.Model(&tpl).Updates(&updates).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("更新模板失败: %v", err))
		return
	}

	db.First(&tpl, id)
	ptSuccess(c, tpl)
}

// ------------------------------------------------------------
// ProtectionTemplateDelete 删除防护策略模板
// DELETE /api/v1/protection-templates/:id
// ------------------------------------------------------------
func ProtectionTemplateDelete(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := getCurrentUserID(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ptFail(c, 400, "无效的 ID")
		return
	}

	var tpl models.ProtectionTemplate
	if err := db.First(&tpl, id).Error; err != nil {
		ptFail(c, 404, "模板不存在")
		return
	}

	if tpl.IsSystem && c.GetString("role") != "admin" {
		ptFail(c, 403, "系统模板仅管理员可删除")
		return
	}
	if !tpl.IsSystem && tpl.UserID != userID {
		ptFail(c, 403, "无权删除他人模板")
		return
	}

	if err := db.Delete(&tpl).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("删除模板失败: %v", err))
		return
	}

	ptSuccess(c, gin.H{"id": id})
}

// ------------------------------------------------------------
// ProtectionTemplateApply 应用模板到域名
// POST /api/v1/protection-templates/:id/apply
// Body: { "domain_id": 123 }
// ------------------------------------------------------------
func ProtectionTemplateApply(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ptFail(c, 400, "无效的 ID")
		return
	}

	var tpl models.ProtectionTemplate
	if err := db.First(&tpl, id).Error; err != nil {
		ptFail(c, 404, "模板不存在")
		return
	}

	var req struct {
		DomainID uint `json:"domain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.DomainID == 0 {
		ptFail(c, 400, "domain_id 不能为空")
		return
	}

	var domain models.Domain
	if err := db.First(&domain, req.DomainID).Error; err != nil {
		ptFail(c, 404, "域名不存在")
		return
	}

	// 将模板配置写入域名的 protection_config 字段
	domain.ProtectionConfig = tpl.Config
	if err := db.Model(&domain).Update("protection_config", tpl.Config).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("应用模板失败: %v", err))
		return
	}

	ptSuccess(c, gin.H{
		"template_id": id,
		"domain_id":   req.DomainID,
	})
}

// ------------------------------------------------------------
// ProtectionTemplateSystemList 系统模板列表
// GET /api/v1/protection-templates/system
// ------------------------------------------------------------
func ProtectionTemplateSystemList(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	page, pageSize, offset := ptPagination(c)

	q := db.Model(&models.ProtectionTemplate{}).Where("is_system = ?", true)
	if name := c.Query("name"); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}

	var total int64
	q.Count(&total)

	var list []models.ProtectionTemplate
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("查询系统模板失败: %v", err))
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
// ProtectionTemplateSystemApply 系统模板应用到域名
// POST /api/v1/protection-templates/system/:id/apply
// Body: { "domain_id": 123 }
// ------------------------------------------------------------
func ProtectionTemplateSystemApply(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ptFail(c, 400, "无效的 ID")
		return
	}

	var tpl models.ProtectionTemplate
	if err := db.Where("id = ? AND is_system = ?", id, true).First(&tpl).Error; err != nil {
		ptFail(c, 404, "系统模板不存在")
		return
	}

	var req struct {
		DomainID uint `json:"domain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.DomainID == 0 {
		ptFail(c, 400, "domain_id 不能为空")
		return
	}

	var domain models.Domain
	if err := db.First(&domain, req.DomainID).Error; err != nil {
		ptFail(c, 404, "域名不存在")
		return
	}

	domain.ProtectionConfig = tpl.Config
	if err := db.Model(&domain).Update("protection_config", tpl.Config).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("应用系统模板失败: %v", err))
		return
	}

	ptSuccess(c, gin.H{
		"template_id": id,
		"domain_id":   req.DomainID,
	})
}

// ------------------------------------------------------------
// BlacklistList 黑白名单列表
// GET /api/v1/blacklists
// ------------------------------------------------------------
func BlacklistList(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := getCurrentUserID(c)
	page, pageSize, offset := ptPagination(c)

	q := db.Model(&models.BlacklistEntry{}).Where("user_id = ?", userID)
	if listType := c.Query("list_type"); listType != "" {
		q = q.Where("list_type = ?", listType)
	}
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
	q.Count(&total)

	var list []models.BlacklistEntry
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("查询黑白名单失败: %v", err))
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
// BlacklistCreate 添加黑白名单
// POST /api/v1/blacklists
// Body:
// {
//   "type": "ip",          // ip / url / domain
//   "list_type": "black",  // black / white
//   "value": "1.2.3.4",
//   "match_mode": "exact", // exact / prefix / regex / cidr
//   "domain_id": 123,
//   "http_method": ""
// }
// ------------------------------------------------------------
func BlacklistCreate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := getCurrentUserID(c)

	var entry models.BlacklistEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		ptFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	if entry.Value == "" {
		ptFail(c, 400, "value 不能为空")
		return
	}
	if entry.Type == "" {
		ptFail(c, 400, "type 不能为空")
		return
	}
	if entry.ListType == "" {
		ptFail(c, 400, "list_type 不能为空")
		return
	}
	if entry.MatchMode == "" {
		entry.MatchMode = "exact"
	}

	entry.UserID = userID

	if err := db.Create(&entry).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("添加黑白名单失败: %v", err))
		return
	}

	ptSuccess(c, entry)
}

// ------------------------------------------------------------
// BlacklistDelete 删除黑白名单
// DELETE /api/v1/blacklists/:id
// ------------------------------------------------------------
func BlacklistDelete(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := getCurrentUserID(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		ptFail(c, 400, "无效的 ID")
		return
	}

	var entry models.BlacklistEntry
	if err := db.First(&entry, id).Error; err != nil {
		ptFail(c, 404, "条目不存在")
		return
	}

	if entry.UserID != userID && c.GetString("role") != "admin" {
		ptFail(c, 403, "无权删除他人条目")
		return
	}

	if err := db.Delete(&entry).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("删除条目失败: %v", err))
		return
	}

	ptSuccess(c, gin.H{"id": id})
}

// ------------------------------------------------------------
// BlacklistImport CSV 批量导入
// POST /api/v1/blacklists/import
// Content-Type: multipart/form-data
// 表单字段: file (CSV), list_type, type, domain_id(可选)
// CSV 格式: value,match_mode,http_method
// ------------------------------------------------------------
func BlacklistImport(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := getCurrentUserID(c)

	listType := c.PostForm("list_type")
	entryType := c.PostForm("type")
	domainIDStr := c.PostForm("domain_id")
	if listType == "" {
		listType = "black"
	}
	if entryType == "" {
		entryType = "ip"
	}
	var domainID *uint
	if domainIDStr != "" {
		if id, err := strconv.ParseUint(domainIDStr, 10, 64); err == nil && id > 0 {
			did := uint(id)
			domainID = &did
		}
	}

	file, err := c.FormFile("file")
	if err != nil {
		ptFail(c, 400, "请上传 CSV 文件")
		return
	}

	f, err := file.Open()
	if err != nil {
		ptFail(c, 500, fmt.Sprintf("打开文件失败: %v", err))
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		ptFail(c, 400, fmt.Sprintf("解析 CSV 失败: %v", err))
		return
	}

	successCount := 0
	failCount := 0
	var errors []string

	for i, row := range records {
		// 跳过表头
		if i == 0 {
			if len(row) > 0 && (strings.EqualFold(row[0], "value") || strings.EqualFold(row[0], "ip")) {
				continue
			}
		}
		if len(row) < 1 || strings.TrimSpace(row[0]) == "" {
			failCount++
			errors = append(errors, fmt.Sprintf("第 %d 行: value 为空", i+1))
			continue
		}

		entry := models.BlacklistEntry{
			UserID:   userID,
			DomainID: domainID,
			Type:     entryType,
			ListType: listType,
			Value:    strings.TrimSpace(row[0]),
		}
		if len(row) > 1 && row[1] != "" {
			entry.MatchMode = strings.TrimSpace(row[1])
		} else {
			entry.MatchMode = "exact"
		}
		if len(row) > 2 {
			entry.HTTPMethod = strings.TrimSpace(row[2])
		}

		if err := db.Create(&entry).Error; err != nil {
			failCount++
			errors = append(errors, fmt.Sprintf("第 %d 行: %v", i+1, err))
			continue
		}
		successCount++
	}

	ptSuccess(c, gin.H{
		"success_count": successCount,
		"fail_count":    failCount,
		"errors":        errors,
		"total":         len(records),
	})
}

// ------------------------------------------------------------
// BlacklistExport 导出黑白名单 (CSV)
// GET /api/v1/blacklists/export
// ------------------------------------------------------------
func BlacklistExport(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := getCurrentUserID(c)

	q := db.Model(&models.BlacklistEntry{}).Where("user_id = ?", userID)
	if listType := c.Query("list_type"); listType != "" {
		q = q.Where("list_type = ?", listType)
	}
	if entryType := c.Query("type"); entryType != "" {
		q = q.Where("type = ?", entryType)
	}

	var list []models.BlacklistEntry
	if err := q.Order("id DESC").Limit(100000).Find(&list).Error; err != nil {
		ptFail(c, 500, fmt.Sprintf("查询失败: %v", err))
		return
	}

	// 生成 CSV
	var sb strings.Builder
	sb.WriteString("id,type,list_type,value,match_mode,http_method,domain_id,created_at\n")
	for _, e := range list {
		domainID := ""
		if e.DomainID != nil {
			domainID = strconv.FormatUint(uint64(*e.DomainID), 10)
		}
		sb.WriteString(fmt.Sprintf("%d,%s,%s,%s,%s,%s,%s,%s\n",
			e.ID, e.Type, e.ListType, e.Value, e.MatchMode, e.HTTPMethod, domainID, e.CreatedAt.Format(time.RFC3339)))
	}

	filename := fmt.Sprintf("blacklists_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.String(http.StatusOK, sb.String())
}
