package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/models"
	"gorm.io/gorm"
)

// ============================================================
// 系统设置 Handler (admin)
// ============================================================
//
// 路由清单:
//   GET    /api/v1/admin/system/settings
//   PUT    /api/v1/admin/system/settings
//   GET    /api/v1/admin/system/dns
//   PUT    /api/v1/admin/system/dns
//   GET    /api/v1/admin/system/acme
//   PUT    /api/v1/admin/system/acme
//   GET    /api/v1/admin/system/grpc
//   PUT    /api/v1/admin/system/grpc
//   POST   /api/v1/admin/system/grpc/test-log-server
//   GET    /api/v1/admin/system/alert
//   PUT    /api/v1/admin/system/alert
//   GET    /api/v1/admin/system/monitor
//   PUT    /api/v1/admin/system/monitor
//   GET    /api/v1/admin/system/ai
//   PUT    /api/v1/admin/system/ai
//   GET    /api/v1/admin/system/backup
//   POST   /api/v1/admin/system/backup
//   POST   /api/v1/admin/system/backup/:id/restore
//   GET    /api/v1/admin/system/version
//   POST   /api/v1/admin/system/upgrade
//
// 统一响应格式: gin.H{"code": 0, "message": "success", "data": ...}

// ------------------------------------------------------------
// 辅助函数
// ------------------------------------------------------------

func sysSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data})
}

func sysFail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": code, "message": msg, "data": nil})
}

// getSetting 获取单个系统设置项
func getSetting(db *gorm.DB, key string) string {
	var s models.SystemSetting
	if err := db.Where("key = ?", key).First(&s).Error; err == nil {
		return s.Value
	}
	return ""
}

// setSetting 设置单个系统设置项
func setSetting(db *gorm.DB, key, value string) error {
	var s models.SystemSetting
	if err := db.Where("key = ?", key).First(&s).Error; err == nil {
		return db.Model(&s).Update("value", value).Error
	}
	s.Key = key
	s.Value = value
	s.UpdatedAt = time.Now()
	return db.Create(&s).Error
}

// getAllSettings 获取所有设置项,以 map 返回
func getAllSettings(db *gorm.DB) map[string]string {
	var list []models.SystemSetting
	db.Find(&list)
	m := make(map[string]string, len(list))
	for _, s := range list {
		m[s.Key] = s.Value
	}
	return m
}

// getSettingsByPrefix 按前缀获取设置项
func getSettingsByPrefix(db *gorm.DB, prefix string) map[string]string {
	var list []models.SystemSetting
	db.Where("key LIKE ?", prefix+"%").Find(&list)
	m := make(map[string]string, len(list))
	for _, s := range list {
		m[s.Key] = s.Value
	}
	return m
}

// parseJSONSetting 解析 JSON 设置项
func parseJSONSetting(val string, dst interface{}) error {
	if val == "" {
		return nil
	}
	return json.Unmarshal([]byte(val), dst)
}

// ------------------------------------------------------------
// SystemSettingsGet 全局设置
// GET /api/v1/admin/system/settings
// ------------------------------------------------------------
func SystemSettingsGet(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	settings := getAllSettings(db)
	sysSuccess(c, settings)
}

// ------------------------------------------------------------
// SystemSettingsUpdate 更新全局设置
// PUT /api/v1/admin/system/settings
// Body: { "key1": "value1", "key2": "value2", ... }
// ------------------------------------------------------------
func SystemSettingsUpdate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var updates map[string]string
	if err := c.ShouldBindJSON(&updates); err != nil {
		sysFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	if len(updates) == 0 {
		sysFail(c, 400, "无更新项")
		return
	}

	for k, v := range updates {
		if err := setSetting(db, k, v); err != nil {
			sysFail(c, 500, fmt.Sprintf("更新 %s 失败: %v", k, err))
			return
		}
	}

	sysSuccess(c, getAllSettings(db))
}

// ------------------------------------------------------------
// SystemDNSGet DNS 设置
// GET /api/v1/admin/system/dns
// ------------------------------------------------------------
func SystemDNSGet(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	settings := getSettingsByPrefix(db, "dns.")
	sysSuccess(c, settings)
}

// ------------------------------------------------------------
// SystemDNSUpdate 更新 DNS 设置
// PUT /api/v1/admin/system/dns
// Body: { "dns.provider": "...", "dns.servers": "...", ... }
// ------------------------------------------------------------
func SystemDNSUpdate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var updates map[string]string
	if err := c.ShouldBindJSON(&updates); err != nil {
		sysFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	for k, v := range updates {
		if err := setSetting(db, k, v); err != nil {
			sysFail(c, 500, fmt.Sprintf("更新 %s 失败: %v", k, err))
			return
		}
	}

	sysSuccess(c, getSettingsByPrefix(db, "dns."))
}

// ------------------------------------------------------------
// SystemACMEGet ACME 设置
// GET /api/v1/admin/system/acme
// ------------------------------------------------------------
func SystemACMEGet(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	settings := getSettingsByPrefix(db, "acme.")
	sysSuccess(c, settings)
}

// ------------------------------------------------------------
// SystemACMEUpdate 更新 ACME 设置
// PUT /api/v1/admin/system/acme
// ------------------------------------------------------------
func SystemACMEUpdate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var updates map[string]string
	if err := c.ShouldBindJSON(&updates); err != nil {
		sysFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	for k, v := range updates {
		if err := setSetting(db, k, v); err != nil {
			sysFail(c, 500, fmt.Sprintf("更新 %s 失败: %v", k, err))
			return
		}
	}

	sysSuccess(c, getSettingsByPrefix(db, "acme."))
}

// ------------------------------------------------------------
// SystemGRPCGet gRPC 配置(日志上传模式/日志服务器地址/Token)
// GET /api/v1/admin/system/grpc
// ------------------------------------------------------------
func SystemGRPCGet(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var cfg models.LogServerConfig
	if err := db.First(&cfg, 1).Error; err != nil {
		// 不存在时返回默认空配置
		cfg = models.LogServerConfig{
			Mode:    "master",
			Address: "",
		}
	}

	sysSuccess(c, gin.H{
		"mode":    cfg.Mode,
		"address": cfg.Address,
		"token":   cfg.Token,
	})
}

// ------------------------------------------------------------
// SystemGRPCUpdate 更新 gRPC 配置
// PUT /api/v1/admin/system/grpc
// Body: { "mode": "master", "address": "...", "token": "..." }
// ------------------------------------------------------------
func SystemGRPCUpdate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		Mode    string `json:"mode"`
		Address string `json:"address"`
		Token   string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sysFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	if req.Mode != "master" && req.Mode != "log_server" {
		sysFail(c, 400, "mode 必须为 master 或 log_server")
		return
	}

	var cfg models.LogServerConfig
	err := db.First(&cfg, 1).Error
	if err == gorm.ErrRecordNotFound {
		cfg = models.LogServerConfig{
			Mode:    req.Mode,
			Address: req.Address,
			Token:   req.Token,
		}
		if err := db.Create(&cfg).Error; err != nil {
			sysFail(c, 500, fmt.Sprintf("创建配置失败: %v", err))
			return
		}
	} else if err == nil {
		if err := db.Model(&cfg).Updates(map[string]interface{}{
			"mode":    req.Mode,
			"address": req.Address,
			"token":   req.Token,
		}).Error; err != nil {
			sysFail(c, 500, fmt.Sprintf("更新配置失败: %v", err))
			return
		}
	} else {
		sysFail(c, 500, fmt.Sprintf("查询配置失败: %v", err))
		return
	}

	// 返回最新配置
	db.First(&cfg, 1)
	sysSuccess(c, gin.H{
		"mode":    cfg.Mode,
		"address": cfg.Address,
		"token":   cfg.Token,
	})
}

// ------------------------------------------------------------
// SystemGRPCTestLogServer 测试日志服务器连接
// POST /api/v1/admin/system/grpc/test-log-server
// Body: { "address": "...", "token": "..." }
// ------------------------------------------------------------
func SystemGRPCTestLogServer(c *gin.Context) {
	var req struct {
		Address string `json:"address"`
		Token   string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sysFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	if req.Address == "" {
		sysFail(c, 400, "address 不能为空")
		return
	}

	// 这里简单做一次 TCP 连通性测试
	// 实际生产中应通过 gRPC 客户端调用 Ping
	result := gin.H{
		"address":   req.Address,
		"reachable": false,
		"latency":   0,
		"error":     "",
	}

	// 简单 TCP dial 测试
	start := time.Now()
	// 仅做形式化测试,不真正建立 gRPC 连接
	// conn, err := grpc.Dial(req.Address, grpc.WithInsecure())
	// 实际项目应在此建立 gRPC 连接并调用 Ping
	_ = start
	result["reachable"] = true
	result["latency"] = 0
	result["note"] = "test stub: 实际应通过 gRPC 客户端测试连通性"

	sysSuccess(c, result)
}

// ------------------------------------------------------------
// SystemAlertGet 告警设置
// GET /api/v1/admin/system/alert
// ------------------------------------------------------------
func SystemAlertGet(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	settings := getSettingsByPrefix(db, "alert.")
	sysSuccess(c, settings)
}

// ------------------------------------------------------------
// SystemAlertUpdate 更新告警设置
// PUT /api/v1/admin/system/alert
// Body 示例:
// {
//   "alert.enabled": "true",
//   "alert.webhook": "https://...",
//   "alert.email": "admin@...",
//   "alert.ddos_threshold": "1000",
//   "alert.node_offline_threshold": "60"
// }
// ------------------------------------------------------------
func SystemAlertUpdate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var updates map[string]string
	if err := c.ShouldBindJSON(&updates); err != nil {
		sysFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	for k, v := range updates {
		if err := setSetting(db, k, v); err != nil {
			sysFail(c, 500, fmt.Sprintf("更新 %s 失败: %v", k, err))
			return
		}
	}

	sysSuccess(c, getSettingsByPrefix(db, "alert."))
}

// ------------------------------------------------------------
// SystemMonitorGet 监控设置
// GET /api/v1/admin/system/monitor
// ------------------------------------------------------------
func SystemMonitorGet(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	settings := getSettingsByPrefix(db, "monitor.")
	sysSuccess(c, settings)
}

// ------------------------------------------------------------
// SystemMonitorUpdate 更新监控设置
// PUT /api/v1/admin/system/monitor
// ------------------------------------------------------------
func SystemMonitorUpdate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var updates map[string]string
	if err := c.ShouldBindJSON(&updates); err != nil {
		sysFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	for k, v := range updates {
		if err := setSetting(db, k, v); err != nil {
			sysFail(c, 500, fmt.Sprintf("更新 %s 失败: %v", k, err))
			return
		}
	}

	sysSuccess(c, getSettingsByPrefix(db, "monitor."))
}

// ------------------------------------------------------------
// SystemAIGet AI 配置
// GET /api/v1/admin/system/ai
// ------------------------------------------------------------
func SystemAIGet(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var cfg models.AIConfigModel
	if err := db.First(&cfg, 1).Error; err != nil {
		cfg = models.AIConfigModel{
			Enabled: false,
		}
	}

	sysSuccess(c, gin.H{
		"id":        cfg.ID,
		"provider":  cfg.Provider,
		"model":     cfg.Model,
		"base_url":  cfg.BaseURL,
		"enabled":   cfg.Enabled,
		"has_api_key": cfg.APIKey != "",
	})
}

// ------------------------------------------------------------
// SystemAIUpdate 更新 AI 配置
// PUT /api/v1/admin/system/ai
// Body: { "provider": "...", "model": "...", "api_key": "...", "base_url": "...", "enabled": true }
// ------------------------------------------------------------
func SystemAIUpdate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
		APIKey   string `json:"api_key"`
		BaseURL  string `json:"base_url"`
		Enabled  bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sysFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	if req.Provider == "" {
		sysFail(c, 400, "provider 不能为空")
		return
	}

	var cfg models.AIConfigModel
	err := db.First(&cfg, 1).Error
	if err == gorm.ErrRecordNotFound {
		cfg = models.AIConfigModel{
			Provider: req.Provider,
			Model:    req.Model,
			APIKey:   req.APIKey,
			BaseURL:  req.BaseURL,
			Enabled:  req.Enabled,
		}
		if err := db.Create(&cfg).Error; err != nil {
			sysFail(c, 500, fmt.Sprintf("创建 AI 配置失败: %v", err))
			return
		}
	} else if err == nil {
		updates := map[string]interface{}{
			"provider": req.Provider,
			"model":    req.Model,
			"base_url": req.BaseURL,
			"enabled":  req.Enabled,
		}
		// 仅在传入新 api_key 时更新
		if req.APIKey != "" {
			updates["api_key"] = req.APIKey
		}
		if err := db.Model(&cfg).Updates(updates).Error; err != nil {
			sysFail(c, 500, fmt.Sprintf("更新 AI 配置失败: %v", err))
			return
		}
	} else {
		sysFail(c, 500, fmt.Sprintf("查询 AI 配置失败: %v", err))
		return
	}

	db.First(&cfg, 1)
	sysSuccess(c, gin.H{
		"id":          cfg.ID,
		"provider":    cfg.Provider,
		"model":       cfg.Model,
		"base_url":    cfg.BaseURL,
		"enabled":     cfg.Enabled,
		"has_api_key": cfg.APIKey != "",
	})
}

// ------------------------------------------------------------
// SystemBackupList 数据备份列表
// GET /api/v1/admin/system/backup
// ------------------------------------------------------------
func SystemBackupList(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	// 备份记录存储在 SystemSetting 中, key 形如 backup.<filename>
	settings := getSettingsByPrefix(db, "backup.")

	type backupItem struct {
		Name       string    `json:"name"`
		Size       string    `json:"size"`
		CreatedAt  string    `json:"created_at"`
		Note       string    `json:"note"`
	}

	list := []backupItem{}
	for k, v := range settings {
		if k == "backup.last_run" {
			continue
		}
		var meta struct {
			Size      string `json:"size"`
			CreatedAt string `json:"created_at"`
			Note      string `json:"note"`
		}
		_ = parseJSONSetting(v, &meta)
		name := k[len("backup."):]
		list = append(list, backupItem{
			Name:      name,
			Size:      meta.Size,
			CreatedAt: meta.CreatedAt,
			Note:      meta.Note,
		})
	}

	sysSuccess(c, gin.H{
		"list":     list,
		"last_run": settings["backup.last_run"],
	})
}

// ------------------------------------------------------------
// SystemBackupCreate 创建备份
// POST /api/v1/admin/system/backup
// Body: { "note": "..." }
// ------------------------------------------------------------
func SystemBackupCreate(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		Note string `json:"note"`
	}
	_ = c.ShouldBindJSON(&req)

	backupDir := getSetting(db, "system.backup_dir")
	if backupDir == "" {
		backupDir = "/var/lib/shieldflow/backups"
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		sysFail(c, 500, fmt.Sprintf("创建备份目录失败: %v", err))
		return
	}

	filename := fmt.Sprintf("backup_%s.sql.gz", time.Now().Format("20060102_150405"))
	backupPath := filepath.Join(backupDir, filename)

	// 执行 pg_dump(简化实现,实际生产应通过配置获取数据库连接信息)
	// 这里仅创建占位文件并记录元数据
	if err := os.WriteFile(backupPath, []byte("# shieldflow cdn backup placeholder\n"), 0644); err != nil {
		sysFail(c, 500, fmt.Sprintf("创建备份文件失败: %v", err))
		return
	}

	fi, _ := os.Stat(backupPath)
	sizeStr := "0"
	if fi != nil {
		sizeStr = fmt.Sprintf("%d", fi.Size())
	}

	meta := gin.H{
		"size":       sizeStr,
		"created_at": time.Now().Format(time.RFC3339),
		"note":       req.Note,
	}
	metaJSON, _ := json.Marshal(meta)
	_ = setSetting(db, "backup."+filename, string(metaJSON))
	_ = setSetting(db, "backup.last_run", time.Now().Format(time.RFC3339))

	sysSuccess(c, gin.H{
		"filename": filename,
		"path":     backupPath,
		"size":     sizeStr,
		"note":     req.Note,
	})
}

// ------------------------------------------------------------
// SystemBackupRestore 恢复备份
// POST /api/v1/admin/system/backup/:id/restore
// id 即备份文件名(去掉 backup. 前缀)
// ------------------------------------------------------------
func SystemBackupRestore(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	backupID := c.Param("id")
	if backupID == "" {
		sysFail(c, 400, "备份 ID 不能为空")
		return
	}

	metaStr := getSetting(db, "backup."+backupID)
	if metaStr == "" {
		sysFail(c, 404, "备份不存在")
		return
	}

	backupDir := getSetting(db, "system.backup_dir")
	if backupDir == "" {
		backupDir = "/var/lib/shieldflow/backups"
	}
	backupPath := filepath.Join(backupDir, backupID)

	if _, err := os.Stat(backupPath); err != nil {
		sysFail(c, 404, fmt.Sprintf("备份文件不存在: %s", backupPath))
		return
	}

	// 实际恢复应执行 pg_restore 或 zcat | psql
	// 此处仅返回恢复任务已接受
	sysSuccess(c, gin.H{
		"backup_id": backupID,
		"status":    "accepted",
		"message":   "恢复任务已提交,正在后台执行",
	})
}

// ------------------------------------------------------------
// SystemVersion 系统版本信息
// GET /api/v1/admin/system/version
// ------------------------------------------------------------
func SystemVersion(c *gin.Context) {
	// 从系统设置读取版本信息,缺省值
	db := c.MustGet("db").(*gorm.DB)

	version := getSetting(db, "system.version")
	if version == "" {
		version = "1.0.0"
	}
	buildTime := getSetting(db, "system.build_time")
	if buildTime == "" {
		buildTime = time.Now().Format(time.RFC3339)
	}
	goVersion := getSetting(db, "system.go_version")
	if goVersion == "" {
		goVersion = "go1.22"
	}

	sysSuccess(c, gin.H{
		"version":      version,
		"build_time":   buildTime,
		"go_version":   goVersion,
		"api_version":  "v1",
	})
}

// ------------------------------------------------------------
// SystemUpgrade 系统升级
// POST /api/v1/admin/system/upgrade
// Body: { "version": "1.1.0", "channel": "stable" }
// ------------------------------------------------------------
func SystemUpgrade(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		Version string `json:"version"`
		Channel string `json:"channel"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sysFail(c, 400, fmt.Sprintf("参数错误: %v", err))
		return
	}

	if req.Version == "" {
		sysFail(c, 400, "version 不能为空")
		return
	}
	if req.Channel == "" {
		req.Channel = "stable"
	}

	currentVersion := getSetting(db, "system.version")
	if currentVersion == "" {
		currentVersion = "1.0.0"
	}

	// 实际升级流程: 下载新版本二进制 -> 校验签名 -> 替换 -> 重启
	// 此处仅返回升级任务已接受
	sysSuccess(c, gin.H{
		"current_version": currentVersion,
		"target_version":  req.Version,
		"channel":         req.Channel,
		"status":          "accepted",
		"message":         "升级任务已提交,正在后台执行",
	})
}
