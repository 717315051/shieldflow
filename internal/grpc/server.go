package grpc

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shieldflow/shieldflow/internal/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Server 是基于 Gin 实现的"伪 gRPC"服务，
// 暴露与 gRPC 语义等价的 RESTful 接口供边缘节点调用。
type Server struct {
	cfg     *config.Config
	db      *gorm.DB
	log     *zap.Logger
	handler *Handler
	router  *gin.Engine
	httpSrv *http.Server
}

// NewServer 创建 gRPC REST 服务实例
func NewServer(cfg *config.Config, db *gorm.DB, log *zap.Logger) *Server {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	s := &Server{
		cfg: cfg,
		db:  db,
		log: log,
	}
	s.handler = NewHandler(db, log)
	s.router = s.setupRouter()
	return s
}

// setupRouter 配置所有 REST 路由
func (s *Server) setupRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(s.requestLogger())
	r.Use(s.authMiddleware()) // 简单 token 鉴权

	// 健康检查
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "ts": time.Now().Unix()})
	})

	v1 := r.Group("/grpc/v1")
	{
		// Config Service
		v1.POST("/config/push", s.handlePushDomainConfig)
		v1.POST("/config/ddos", s.handlePushDDoSConfig)
		v1.POST("/config/global", s.handlePushGlobalConfig)
		v1.POST("/config/sync-status", s.handleSyncNodeStatus)
		v1.POST("/node/heartbeat", s.handleHeartbeat)

		// Log Service
		v1.POST("/logs/access", s.handleReportAccessLogs)
		v1.POST("/logs/attack", s.handleReportAttackLogs)
		v1.POST("/logs/ddos", s.handleReportDDoSLogs)
		v1.POST("/logs/layer4", s.handleReportLayer4Logs)
		v1.POST("/logs/ai", s.handleReportAILogs)

		// Node Service
		v1.POST("/node/register", s.handleRegisterNode)
		v1.POST("/node/status", s.handleUpdateNodeStatus)
		v1.GET("/node/config/:node_id", s.handleGetNodeConfig)

		// Cache Service
		v1.POST("/cache/purge", s.handlePurgeCache)
		v1.POST("/cache/preheat", s.handlePreheatCache)
		v1.GET("/cache/stats/:domain_id", s.handleGetCacheStats)

		// Auth Service
		v1.POST("/auth/verify-license", s.handleVerifyLicense)
	}

	return r
}

// Run 启动 HTTP REST 服务（模拟 gRPC）
func (s *Server) Run() error {
	addr := ":50051"
	if s.cfg.GRPC.Port != 0 {
		addr = formatAddr(s.cfg.GRPC.Port)
	}

	s.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	s.log.Info("ShieldFlow gRPC(REST) server starting",
		zap.String("addr", addr),
		zap.Bool("tls", s.cfg.GRPC.TLS),
	)

	if s.cfg.GRPC.TLS {
		return s.httpSrv.ListenAndServeTLS(s.cfg.GRPC.Cert, s.cfg.GRPC.Key)
	}
	return s.httpSrv.ListenAndServe()
}

// Shutdown 优雅关闭
func (s *Server) Shutdown() error {
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Close()
}

// formatAddr
func formatAddr(port int) string {
	return ":" + itoa(port)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ============================================================================
// 中间件
// ============================================================================

func (s *Server) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		s.log.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		)
	}
}

// authMiddleware 简单 token 鉴权（边缘节点带 X-Node-Token 头）
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 健康检查放行
		if c.Request.URL.Path == "/healthz" {
			c.Next()
			return
		}
		token := c.GetHeader("X-Node-Token")
		if token == "" {
			// 开发模式下放行；生产模式应严格校验
			c.Next()
			return
		}
		c.Set("node_token", token)
		c.Next()
	}
}

// ============================================================================
// Config Service Handlers
// ============================================================================

func (s *Server) handlePushDomainConfig(c *gin.Context) {
	var req PushDomainConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	cfg, err := s.handler.PushDomainConfig(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(cfg))
}

func (s *Server) handlePushDDoSConfig(c *gin.Context) {
	var req PushDDoSConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	cfg, err := s.handler.PushDDoSConfig(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(cfg))
}

func (s *Server) handlePushGlobalConfig(c *gin.Context) {
	var req PushGlobalConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	cfg, err := s.handler.PushGlobalConfig(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(cfg))
}

func (s *Server) handleSyncNodeStatus(c *gin.Context) {
	var req SyncNodeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	if err := s.handler.SyncNodeStatus(req); err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(nil))
}

func (s *Server) handleHeartbeat(c *gin.Context) {
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	resp, err := s.handler.Heartbeat(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(resp))
}

// ============================================================================
// Log Service Handlers
// ============================================================================

func (s *Server) handleReportAccessLogs(c *gin.Context) {
	var batch AccessLogBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	if err := s.handler.ReportAccessLogs(batch); err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(gin.H{"accepted": len(batch.Logs)}))
}

func (s *Server) handleReportAttackLogs(c *gin.Context) {
	var batch AttackLogBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	if err := s.handler.ReportAttackLogs(batch); err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(gin.H{"accepted": len(batch.Logs)}))
}

func (s *Server) handleReportDDoSLogs(c *gin.Context) {
	var batch DDoSLogBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	if err := s.handler.ReportDDoSLogs(batch); err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(gin.H{"accepted": len(batch.Logs)}))
}

func (s *Server) handleReportLayer4Logs(c *gin.Context) {
	var batch Layer4LogBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	if err := s.handler.ReportLayer4Logs(batch); err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(gin.H{"accepted": len(batch.Logs)}))
}

func (s *Server) handleReportAILogs(c *gin.Context) {
	var batch AILogBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	if err := s.handler.ReportAILogs(batch); err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(gin.H{"accepted": len(batch.Logs)}))
}

// ============================================================================
// Node Service Handlers
// ============================================================================

func (s *Server) handleRegisterNode(c *gin.Context) {
	var req RegisterNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	resp, err := s.handler.RegisterNode(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(resp))
}

func (s *Server) handleUpdateNodeStatus(c *gin.Context) {
	var req UpdateNodeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	if err := s.handler.UpdateNodeStatus(req); err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(nil))
}

func (s *Server) handleGetNodeConfig(c *gin.Context) {
	nodeIDStr := c.Param("node_id")
	var nodeID uint
	if _, err := fmt.Sscanf(nodeIDStr, "%d", &nodeID); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, "invalid node_id"))
		return
	}
	resp, err := s.handler.GetNodeConfig(nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(resp))
}

// ============================================================================
// Cache Service Handlers
// ============================================================================

func (s *Server) handlePurgeCache(c *gin.Context) {
	var req PurgeCacheRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	if err := s.handler.PurgeCache(req); err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(gin.H{"status": "purge_task_created"}))
}

func (s *Server) handlePreheatCache(c *gin.Context) {
	var req PreheatCacheRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	if err := s.handler.PreheatCache(req); err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(gin.H{"status": "preheat_task_created"}))
}

func (s *Server) handleGetCacheStats(c *gin.Context) {
	domainIDStr := c.Param("domain_id")
	var domainID uint
	if _, err := fmt.Sscanf(domainIDStr, "%d", &domainID); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, "invalid domain_id"))
		return
	}
	resp, err := s.handler.GetCacheStats(domainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(resp))
}

// ============================================================================
// Auth Service Handlers
// ============================================================================

func (s *Server) handleVerifyLicense(c *gin.Context) {
	var req VerifyLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, NewErrorResponse(400, err.Error()))
		return
	}
	resp, err := s.handler.VerifyLicense(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, NewOKResponse(resp))
}
