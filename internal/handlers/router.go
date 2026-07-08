package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/config"
	"github.com/shieldflow/shieldflow/internal/middleware"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// SetupRouter 设置路由
func SetupRouter(cfg *config.Config, db *gorm.DB, ch interface{}, rdb interface{}) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// 中间件
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.JWTMiddleware(&cfg.JWT))
	r.Use(middleware.RateLimitMiddleware())

	// 注入依赖
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("config", cfg)
		if ch != nil {
			c.Set("ch", ch)
		}
		if rdb != nil {
			c.Set("rdb", rdb)
		}
		c.Next()
	})

	// 静态文件
	r.Static("/static", "./web/dist")

	// 健康检查
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": "1.2.0"})
	})

	api := r.Group("/api/v1")

	// ==================== 认证 ====================
	var redisClient *redis.Client
	if rdb != nil {
		redisClient = rdb.(*redis.Client)
	}
	authHandler := NewAuthHandler(cfg, redisClient)
	auth := api.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register)
		auth.POST("/logout", authHandler.Logout)
		auth.GET("/captcha", authHandler.Captcha)
		auth.POST("/verify-code", authHandler.VerifyCode)
		auth.POST("/send-email-code", authHandler.SendEmailCodeHandler)
		auth.GET("/profile", authHandler.GetProfile)
		auth.PUT("/profile", authHandler.UpdateProfile)
		auth.PUT("/password", authHandler.ChangePassword)
		auth.POST("/realname", authHandler.Realname)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)
	}

	// ==================== 域名管理 ====================
	domainHandler := NewDomainHandler()
	domain := api.Group("/domains")
	{
		domain.GET("", domainHandler.ListDomains)
		domain.POST("", domainHandler.CreateDomain)
		domain.POST("/batch", domainHandler.BatchCreateDomains)
		domain.GET("/:id", domainHandler.GetDomain)
		domain.PUT("/:id", domainHandler.UpdateDomain)
		domain.DELETE("/:id", domainHandler.DeleteDomain)
		domain.PUT("/:id/status", domainHandler.UpdateDomainStatus)
		domain.PUT("/:id/package", domainHandler.ChangeDomainPackage)
		domain.GET("/:id/config", domainHandler.GetDomainConfig)
		domain.PUT("/:id/basic", domainHandler.SaveDomainBasic)
		domain.PUT("/:id/protection", domainHandler.SaveDomainProtection)
		domain.PUT("/:id/custom-pages", domainHandler.SaveDomainCustomPages)
		domain.POST("/:id/certificate", domainHandler.ApplyCertificate)
		domain.POST("/batch-certificate", domainHandler.BatchApplyCertificate)
	}

	// ==================== SSL证书 ====================
	cert := api.Group("/certificates")
	{
		cert.GET("", ListCertificates)
		cert.POST("/upload", UploadCertificate)
		cert.GET("/:id", GetCertificate)
		cert.DELETE("/:id", DeleteCertificate)
		cert.GET("/:id/download", DownloadCertificate)
		cert.GET("/requests", ListCertificateRequests)
		cert.POST("/apply", ApplyCertificate)
		cert.GET("/requests/:id", GetCertificateRequest)
		cert.GET("/requests/:id/log", GetCertificateRequestLog)
		cert.DELETE("/requests/:id", DeleteCertificateRequest)
		cert.GET("/acme-accounts", ListAcmeAccounts)
		cert.POST("/acme-accounts", CreateAcmeAccount)
		cert.GET("/dns-accounts", ListDNSAccounts)
		cert.POST("/dns-accounts", CreateDNSAccount)
	}

	// ==================== ACME / DNS 账户（文档 API.md 顶置路由）====================
	// 顶层别名，方便前端按规范文档直接调用 /api/v1/acme-accounts、/api/v1/dns-accounts
	{
		api.GET("/acme-accounts", ListAcmeAccounts)
		api.POST("/acme-accounts", CreateAcmeAccount)
		api.GET("/dns-accounts", ListDNSAccounts)
		api.POST("/dns-accounts", CreateDNSAccount)
	}

	// ==================== 日志管理 ====================
	logs := api.Group("/logs")
	{
		logs.GET("/access", GetAccessLogs)
		logs.GET("/attack", GetAttackLogs)
		logs.GET("/layer4", GetLayer4Logs)
		logs.GET("/layer4-intercept", GetLayer4InterceptLogs)
		logs.GET("/ai", GetAILogs)
		logs.POST("/export", ExportLogs)
		logs.GET("/map", GetLogMap)
	}

	// ==================== 流量统计 ====================
	traffic := api.Group("/traffic")
	{
		traffic.GET("/stats", GetTrafficStats)
		traffic.GET("/ranking", GetTrafficRanking)
		traffic.GET("/bandwidth", GetBandwidthTrend)
		traffic.GET("/cache-hit", GetCacheStats)
	}

	// ==================== 缓存管理 ====================
	cache := api.Group("/cache")
	{
		cache.POST("/file-refresh", FileRefresh)
		cache.POST("/dir-refresh", DirRefresh)
		cache.POST("/file-preheat", FilePreheat)
		cache.GET("/tasks", ListCacheTasks)
		cache.GET("/tasks/:id", GetCacheTask)
		cache.POST("/tasks/:id/cancel", CancelCacheTask)
	}

	// ==================== 四层转发 ====================
	layer4 := api.Group("/layer4")
	{
		layer4.GET("", ListLayer4)
		layer4.POST("", CreateLayer4)
		layer4.PUT("/:id", UpdateLayer4)
		layer4.DELETE("/:id", DeleteLayer4)
		layer4.PUT("/:id/status", UpdateLayer4Status)
	}

	// ==================== 防护管理 ====================
	prot := api.Group("/protection")
	{
		prot.GET("/templates", ProtectionTemplateList)
		prot.POST("/templates", ProtectionTemplateCreate)
		prot.PUT("/templates/:id", ProtectionTemplateUpdate)
		prot.DELETE("/templates/:id", ProtectionTemplateDelete)
		prot.POST("/templates/:id/apply", ProtectionTemplateApply)
		prot.GET("/templates/system", ProtectionTemplateSystemList)
		prot.POST("/templates/system/:id/apply", ProtectionTemplateSystemApply)
		prot.GET("/blacklists", BlacklistList)
		prot.POST("/blacklists", BlacklistCreate)
		prot.DELETE("/blacklists/:id", BlacklistDelete)
		prot.POST("/blacklists/import", BlacklistImport)
		prot.GET("/blacklists/export", BlacklistExport)
		prot.GET("/whitelist", WhitelistList)
		prot.POST("/whitelist", WhitelistCreate)
		prot.DELETE("/whitelist/:id", WhitelistDelete)
	}

	// ==================== 套餐管理 ====================
	pkgHandler := NewPackageHandler(db)
	pkgMarketHandler := NewPackageMarketHandler(db)
	pkg := api.Group("/packages")
	{
		pkg.GET("", pkgHandler.List)
		pkg.GET("/traffic", pkgHandler.TrafficPackages)
		pkg.GET("/domain", pkgHandler.DomainPackages)
		pkg.GET("/market", pkgMarketHandler.Market)
		pkg.POST("/:id/purchase", pkgHandler.Purchase)
		pkg.POST("/traffic/:id/purchase", pkgHandler.PurchaseTraffic)
		pkg.POST("/domain/:id/purchase", pkgHandler.PurchaseDomain)
	}

	userPkg := api.Group("/user-packages")
	{
		userPkg.GET("", pkgHandler.MyPackages)
		userPkg.GET("/:id", pkgHandler.PackageDetail)
		userPkg.POST("/:id/renew", pkgHandler.Renew)
	}

	orders := api.Group("/orders")
	{
		orders.GET("", pkgHandler.Orders)
		orders.GET("/:id", pkgHandler.OrderDetail)
	}

	balance := api.Group("/balance")
	{
		balance.GET("", pkgHandler.Balance)
		balance.POST("/recharge", pkgHandler.Recharge)
	}

	// ==================== 仪表盘 ====================
	dash := api.Group("/dashboard")
	{
		h := NewDashboardHandler(db, ch)
		dash.GET("/analysis", h.Analysis)
	}

	// ==================== 管理端 ====================
	admin := api.Group("/admin")
	admin.Use(middleware.AdminOnly())
	{
		// 用户管理
		userH := NewUserHandler()
		admin.GET("/users", userH.ListUsers)
		admin.POST("/users", userH.CreateUser)
		admin.PUT("/users/:id", userH.UpdateUser)
		admin.DELETE("/users/:id", userH.DeleteUser)
		admin.PUT("/users/:id/status", userH.UpdateUserStatus)
		admin.GET("/users/:id/packages", userH.ListUserPackages)

		// 节点管理
		nodeH := NewNodeHandler()
		admin.GET("/nodes", nodeH.ListNodes)
		admin.POST("/nodes", nodeH.CreateNode)
		admin.DELETE("/nodes/:id", nodeH.DeleteNode)
		admin.GET("/nodes/:id", nodeH.GetNode)
		admin.PUT("/nodes/:id", nodeH.UpdateNode)
		admin.POST("/nodes/:id/install", nodeH.InstallNode)
		admin.POST("/nodes/:id/ssh-install", nodeH.SSHInstallNode)
		admin.POST("/nodes/:id/upgrade", nodeH.UpgradeNode)
		admin.POST("/nodes/batch-upgrade", nodeH.BatchUpgradeNodes)
		admin.GET("/nodes/:id/status", nodeH.GetNodeStatus)
		admin.GET("/node-groups", nodeH.ListNodeGroups)
		admin.POST("/node-groups", nodeH.CreateNodeGroup)
		admin.PUT("/node-groups/:id", nodeH.UpdateNodeGroup)
		admin.DELETE("/node-groups/:id", nodeH.DeleteNodeGroup)

		// 套餐管理
		admin.GET("/packages", pkgHandler.AdminList)
		admin.POST("/packages", pkgHandler.AdminCreate)
		admin.PUT("/packages/:id", pkgHandler.AdminUpdate)
		admin.DELETE("/packages/:id", pkgHandler.AdminDelete)
		admin.POST("/packages/traffic", pkgHandler.AdminCreateTraffic)
		admin.POST("/packages/domain", pkgHandler.AdminCreateDomain)

		// DDoS防护
		admin.GET("/ddos/dashboard", DDoSDashboard)
		admin.GET("/ddos/rules", DDoSRuleList)
		admin.POST("/ddos/rules", DDoSRuleCreate)
		admin.PUT("/ddos/rules/:id", DDoSRuleUpdate)
		admin.DELETE("/ddos/rules/:id", DDoSRuleDelete)
		admin.GET("/ddos/blacklist", DDoSBlacklistList)
		admin.POST("/ddos/blacklist", DDoSBlacklistCreate)
		admin.DELETE("/ddos/blacklist/:id", DDoSBlacklistDelete)
		admin.GET("/ddos/whitelist", DDoSWhitelistList)
		admin.POST("/ddos/whitelist", DDoSWhitelistCreate)
		admin.DELETE("/ddos/whitelist/:id", DDoSWhitelistDelete)
		admin.GET("/ddos/logs", DDoSConnectionLogs)
		admin.GET("/ddos/intercept-logs", DDoSInterceptLogs)

		// WAF防护管理
		admin.GET("/waf/dashboard", WAFDashboard)
		admin.GET("/waf/config", WAFConfigGet)
		admin.PUT("/waf/config", WAFConfigUpdate)
		admin.GET("/waf/logs", WAFLogs)
		admin.GET("/waf/analysis", WAFAttackAnalysis)

		// AI管理
		admin.GET("/ai/dashboard", AIDashboard)
		admin.GET("/ai/token-stats", AITokenStats)
		admin.GET("/ai/cost-analysis", AICostAnalysis)
		admin.GET("/ai/models", AIModelList)
		admin.POST("/ai/models", AIModelCreate)
		admin.PUT("/ai/models/:id", AIModelUpdate)
		admin.DELETE("/ai/models/:id", AIModelDelete)

		// 系统设置
		admin.GET("/system/settings", SystemSettingsGet)
		admin.PUT("/system/settings", SystemSettingsUpdate)
		admin.GET("/system/dns", SystemDNSGet)
		admin.PUT("/system/dns", SystemDNSUpdate)
		admin.GET("/system/acme", SystemACMEGet)
		admin.PUT("/system/acme", SystemACMEUpdate)
		admin.GET("/system/grpc", SystemGRPCGet)
		admin.PUT("/system/grpc", SystemGRPCUpdate)
		admin.POST("/system/grpc/test-log-server", SystemGRPCTestLogServer)
		admin.GET("/system/alert", SystemAlertGet)
		admin.PUT("/system/alert", SystemAlertUpdate)
		admin.GET("/system/monitor", SystemMonitorGet)
		admin.PUT("/system/monitor", SystemMonitorUpdate)
		admin.GET("/system/ai", SystemAIGet)
		admin.PUT("/system/ai", SystemAIUpdate)
		admin.GET("/system/backup", SystemBackupList)
		admin.POST("/system/backup", SystemBackupCreate)
		admin.POST("/system/backup/:id/restore", SystemBackupRestore)
		admin.GET("/system/backup/:id/download", SystemBackupDownload)
		admin.DELETE("/system/backup/:id", SystemBackupDelete)
		admin.GET("/system/version", SystemVersion)
		admin.POST("/system/upgrade", SystemUpgrade)
	}

	return r
}
