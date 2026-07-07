package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/shieldflow/shieldflow/internal/pkg/storage"
	"go.uber.org/zap"
)

// ============ 配置结构 ============

// LogServerConfig log-server 专用配置
type LogServerConfig struct {
	Server     ServerConfig     `mapstructure:"server"`
	ClickHouse ClickHouseConfig `mapstructure:"clickhouse"`
	Writer     WriterConfig     `mapstructure:"writer"`
	Log        LogConfig        `mapstructure:"log"`
}

type ServerConfig struct {
	Port      int    `mapstructure:"port"`
	AuthToken string `mapstructure:"auth_token"`
}

type ClickHouseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type WriterConfig struct {
	BufferSize    int `mapstructure:"buffer_size"`
	BatchSize     int `mapstructure:"batch_size"`
	FlushInterval int `mapstructure:"flush_interval"`
	Retry         int `mapstructure:"retry"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Output string `mapstructure:"output"`
}

// loadConfig 加载 log-server 配置
func loadConfig(path string) (*LogServerConfig, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	if path != "" {
		v.SetConfigFile(path)
	} else {
		// 默认路径
		v.SetConfigName("log-server")
		v.AddConfigPath("/etc/shieldflow")
		v.AddConfigPath(".")
	}

	// 设置默认值
	v.SetDefault("server.port", 9529)
	v.SetDefault("server.auth_token", "")
	v.SetDefault("clickhouse.host", "localhost")
	v.SetDefault("clickhouse.port", 9000)
	v.SetDefault("clickhouse.database", "shieldflow_cdn")
	v.SetDefault("clickhouse.username", "default")
	v.SetDefault("clickhouse.password", "")
	v.SetDefault("writer.buffer_size", 10000)
	v.SetDefault("writer.batch_size", 1000)
	v.SetDefault("writer.flush_interval", 5)
	v.SetDefault("writer.retry", 3)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.output", "/var/log/shieldflow/log-server.log")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config failed: %w", err)
	}

	var cfg LogServerConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}
	return &cfg, nil
}

// initLogger 初始化 zap logger
func initLogger(cfg LogConfig) (*zap.Logger, error) {
	zapCfg := zap.NewProductionConfig()

	switch strings.ToLower(cfg.Level) {
	case "debug":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// 输出到文件
	if cfg.Output != "" {
		if err := os.MkdirAll(filepathDir(cfg.Output), 0755); err != nil {
			return nil, fmt.Errorf("create log dir failed: %w", err)
		}
		zapCfg.OutputPaths = []string{cfg.Output}
		zapCfg.ErrorOutputPaths = []string{cfg.Output}
	}

	return zapCfg.Build()
}

// filepathDir 提取父目录（避免引入 path/filepath 仅为此一处）
func filepathDir(p string) string {
	idx := strings.LastIndex(p, "/")
	if idx < 0 {
		return "."
	}
	if idx == 0 {
		return "/"
	}
	return p[:idx]
}

// initClickHouse 初始化 ClickHouse 连接
func initClickHouse(cfg ClickHouseConfig) (driver.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		DialTimeout:     10 * time.Second,
		Compression:     &clickhouse.Compression{Method: clickhouse.CompressionLZ4},
	})
	if err != nil {
		return nil, fmt.Errorf("open clickhouse failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping clickhouse failed: %w", err)
	}
	return conn, nil
}

// ============ 应用结构 ============

type App struct {
	cfg     *LogServerConfig
	logger  *zap.Logger
	chConn  driver.Conn
	writer  *storage.LogWriter
	querier *storage.LogQuerier
	srv     *http.Server
}

func main() {
	configPath := flag.String("config", "/etc/shieldflow/log-server.yaml", "配置文件路径")
	flag.Parse()

	// 1. 加载配置
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	zapLog, err := initLogger(cfg.Log)
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer zapLog.Sync()

	zapLog.Info("ShieldFlow Log Server 启动中...",
		zap.Int("port", cfg.Server.Port),
		zap.String("version", "1.2.0"),
	)

	// 3. 连接 ClickHouse
	chConn, err := initClickHouse(cfg.ClickHouse)
	if err != nil {
		zapLog.Fatal("ClickHouse 连接失败", zap.Error(err))
	}
	zapLog.Info("ClickHouse 连接成功",
		zap.String("host", cfg.ClickHouse.Host),
		zap.Int("port", cfg.ClickHouse.Port),
		zap.String("database", cfg.ClickHouse.Database),
	)

	// 4. 初始化写入器
	writer := storage.NewLogWriter(chConn, storage.LogWriterConfig{
		BufferSize:    cfg.Writer.BufferSize,
		BatchSize:     cfg.Writer.BatchSize,
		FlushInterval: cfg.Writer.FlushInterval,
		Retry:         cfg.Writer.Retry,
	}, zapLog)
	writer.Start()

	// 5. 初始化查询器
	querier := storage.NewLogQuerier(chConn, zapLog)

	app := &App{
		cfg:     cfg,
		logger:  zapLog,
		chConn:  chConn,
		writer:  writer,
		querier: querier,
	}

	// 6. 启动 HTTP 服务
	router := app.setupRouter()
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	app.srv = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		zapLog.Info("HTTP 服务监听", zap.String("addr", addr))
		if err := app.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLog.Fatal("HTTP 服务启动失败", zap.Error(err))
		}
	}()

	// 7. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zapLog.Info("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.srv.Shutdown(ctx); err != nil {
		zapLog.Error("服务关闭失败", zap.Error(err))
	}

	writer.Stop()
	zapLog.Info("服务已关闭")
}

// ============ 路由 ============

func (a *App) setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 健康检查（无需鉴权）
	r.GET("/health", a.healthCheck)

	// 鉴权中间件保护的 API 组
	api := r.Group("/")
	api.Use(a.authMiddleware())

	// 日志接收
	api.POST("/v1/logs/access", a.receiveAccess)
	api.POST("/v1/logs/attack", a.receiveAttack)
	api.POST("/v1/logs/ddos", a.receiveDDoS)
	api.POST("/v1/logs/layer4", a.receiveLayer4)
	api.POST("/v1/logs/layer4-intercept", a.receiveLayer4Intercept)
	api.POST("/v1/logs/ai", a.receiveAI)

	// 批量接收（多种日志混合）
	api.POST("/v1/logs/batch", a.receiveBatch)

	// 日志查询
	api.GET("/v1/logs/access", a.queryAccess)
	api.GET("/v1/logs/attack", a.queryAttack)
	api.GET("/v1/logs/layer4", a.queryLayer4)
	api.GET("/v1/logs/layer4-intercept", a.queryLayer4Intercept)
	api.GET("/v1/logs/ai", a.queryAI)

	// 统计与分析
	api.GET("/v1/stats/traffic", a.queryTrafficStats)
	api.GET("/v1/stats/top", a.queryTopN)
	api.GET("/v1/stats/geo", a.queryGeoMap)

	// 日志导出
	api.GET("/v1/export/access", a.exportAccess)

	// 写入器指标
	api.GET("/v1/metrics", a.getWriterMetrics)

	return r
}

// ============ 中间件 ============

// authMiddleware 校验 Authorization: Bearer <token>
func (a *App) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 配置为空时跳过鉴权（开发模式）
		if a.cfg.Server.AuthToken == "" {
			c.Next()
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1, "message": "missing authorization"})
			return
		}

		// 支持 "Bearer <token>" 或纯 token
		token := strings.TrimPrefix(auth, "Bearer ")
		token = strings.TrimSpace(token)
		if token != a.cfg.Server.AuthToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1, "message": "invalid token"})
			return
		}
		c.Next()
	}
}

// ============ 健康检查 ============

func (a *App) healthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	chOK := true
	if err := a.chConn.Ping(ctx); err != nil {
		chOK = false
	}

	m := a.writer.GetMetrics()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data": gin.H{
			"clickhouse":     chOK,
			"version":        "1.2.0",
			"total_received": m.TotalReceived,
			"total_written":  m.TotalWritten,
			"total_errors":   m.TotalErrors,
			"buffer_dropped": m.BufferDropped,
			"batches_flushed": m.BatchesFlushed,
		},
	})
}

// ============ 日志接收 handlers ============

// receiveAccess 接收单条访问日志
// POST /v1/logs/access
// Body: 单个 AccessLog JSON 或数组
func (a *App) receiveAccess(c *gin.Context) {
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid body: " + err.Error()})
		return
	}

	trimmed := strings.TrimSpace(string(raw))
	count := 0
	if strings.HasPrefix(trimmed, "[") {
		var logs []storage.AccessLog
		if err := json.Unmarshal(raw, &logs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid access log array: " + err.Error()})
			return
		}
		for _, l := range logs {
			a.writer.WriteEntry(l)
			count++
		}
	} else {
		var l storage.AccessLog
		if err := json.Unmarshal(raw, &l); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid access log: " + err.Error()})
			return
		}
		a.writer.WriteEntry(l)
		count++
	}

	c.JSON(http.StatusAccepted, gin.H{"code": 0, "message": "accepted", "data": gin.H{"count": count}})
}

func (a *App) receiveAttack(c *gin.Context) {
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid body: " + err.Error()})
		return
	}

	trimmed := strings.TrimSpace(string(raw))
	count := 0
	if strings.HasPrefix(trimmed, "[") {
		var logs []storage.AttackLog
		if err := json.Unmarshal(raw, &logs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid attack log array: " + err.Error()})
			return
		}
		for _, l := range logs {
			a.writer.WriteEntry(l)
			count++
		}
	} else {
		var l storage.AttackLog
		if err := json.Unmarshal(raw, &l); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid attack log: " + err.Error()})
			return
		}
		a.writer.WriteEntry(l)
		count++
	}

	c.JSON(http.StatusAccepted, gin.H{"code": 0, "message": "accepted", "data": gin.H{"count": count}})
}

func (a *App) receiveDDoS(c *gin.Context) {
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid body: " + err.Error()})
		return
	}

	trimmed := strings.TrimSpace(string(raw))
	count := 0
	if strings.HasPrefix(trimmed, "[") {
		var logs []storage.DDoSLog
		if err := json.Unmarshal(raw, &logs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid ddos log array: " + err.Error()})
			return
		}
		for _, l := range logs {
			a.writer.WriteEntry(l)
			count++
		}
	} else {
		var l storage.DDoSLog
		if err := json.Unmarshal(raw, &l); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid ddos log: " + err.Error()})
			return
		}
		a.writer.WriteEntry(l)
		count++
	}

	c.JSON(http.StatusAccepted, gin.H{"code": 0, "message": "accepted", "data": gin.H{"count": count}})
}

func (a *App) receiveLayer4(c *gin.Context) {
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid body: " + err.Error()})
		return
	}

	trimmed := strings.TrimSpace(string(raw))
	count := 0
	if strings.HasPrefix(trimmed, "[") {
		var logs []storage.Layer4Log
		if err := json.Unmarshal(raw, &logs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid layer4 log array: " + err.Error()})
			return
		}
		for _, l := range logs {
			a.writer.WriteEntry(l)
			count++
		}
	} else {
		var l storage.Layer4Log
		if err := json.Unmarshal(raw, &l); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid layer4 log: " + err.Error()})
			return
		}
		a.writer.WriteEntry(l)
		count++
	}

	c.JSON(http.StatusAccepted, gin.H{"code": 0, "message": "accepted", "data": gin.H{"count": count}})
}

func (a *App) receiveLayer4Intercept(c *gin.Context) {
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid body: " + err.Error()})
		return
	}

	trimmed := strings.TrimSpace(string(raw))
	count := 0
	if strings.HasPrefix(trimmed, "[") {
		var logs []storage.Layer4InterceptLog
		if err := json.Unmarshal(raw, &logs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid layer4 intercept log array: " + err.Error()})
			return
		}
		for _, l := range logs {
			a.writer.WriteEntry(l)
			count++
		}
	} else {
		var l storage.Layer4InterceptLog
		if err := json.Unmarshal(raw, &l); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid layer4 intercept log: " + err.Error()})
			return
		}
		a.writer.WriteEntry(l)
		count++
	}

	c.JSON(http.StatusAccepted, gin.H{"code": 0, "message": "accepted", "data": gin.H{"count": count}})
}

func (a *App) receiveAI(c *gin.Context) {
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid body: " + err.Error()})
		return
	}

	trimmed := strings.TrimSpace(string(raw))
	count := 0
	if strings.HasPrefix(trimmed, "[") {
		var logs []storage.AILog
		if err := json.Unmarshal(raw, &logs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid ai log array: " + err.Error()})
			return
		}
		for _, l := range logs {
			a.writer.WriteEntry(l)
			count++
		}
	} else {
		var l storage.AILog
		if err := json.Unmarshal(raw, &l); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid ai log: " + err.Error()})
			return
		}
		a.writer.WriteEntry(l)
		count++
	}

	c.JSON(http.StatusAccepted, gin.H{"code": 0, "message": "accepted", "data": gin.H{"count": count}})
}

// receiveBatch 批量接收混合日志
// POST /v1/logs/batch
// Body: {"access":[...], "attack":[...], "ddos":[...], "layer4":[...],
//        "layer4_intercept":[...], "ai":[...]}
func (a *App) receiveBatch(c *gin.Context) {
	var payload struct {
		Access          []storage.AccessLog          `json:"access"`
		Attack          []storage.AttackLog          `json:"attack"`
		DDoS            []storage.DDoSLog            `json:"ddos"`
		Layer4          []storage.Layer4Log          `json:"layer4"`
		Layer4Intercept []storage.Layer4InterceptLog `json:"layer4_intercept"`
		AI              []storage.AILog              `json:"ai"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "invalid body: " + err.Error()})
		return
	}

	count := 0
	for _, l := range payload.Access {
		a.writer.WriteEntry(l)
		count++
	}
	for _, l := range payload.Attack {
		a.writer.WriteEntry(l)
		count++
	}
	for _, l := range payload.DDoS {
		a.writer.WriteEntry(l)
		count++
	}
	for _, l := range payload.Layer4 {
		a.writer.WriteEntry(l)
		count++
	}
	for _, l := range payload.Layer4Intercept {
		a.writer.WriteEntry(l)
		count++
	}
	for _, l := range payload.AI {
		a.writer.WriteEntry(l)
		count++
	}

	c.JSON(http.StatusAccepted, gin.H{"code": 0, "message": "accepted", "data": gin.H{"count": count}})
}

// ============ 日志查询 handlers ============

func (a *App) queryAccess(c *gin.Context) {
	page, pageSize := parsePagination(c)
	req := storage.AccessLogQuery{
		Domain:    c.Query("domain"),
		ClientIP:  c.Query("ip"),
		URL:       c.Query("url"),
		Method:    c.Query("method"),
		Status:    c.Query("status"),
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
		EdgeNode:  c.Query("edge_node"),
		Page:      page,
		PageSize:  pageSize,
	}
	result, err := a.querier.QueryAccessLogs(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

func (a *App) queryAttack(c *gin.Context) {
	page, pageSize := parsePagination(c)
	req := storage.AttackLogQuery{
		Domain:     c.Query("domain"),
		ClientIP:   c.Query("ip"),
		AttackType: c.Query("attack_type"),
		Action:     c.Query("action"),
		StartTime:  c.Query("start_time"),
		EndTime:    c.Query("end_time"),
		Page:       page,
		PageSize:   pageSize,
	}
	result, err := a.querier.QueryAttackLogs(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

func (a *App) queryLayer4(c *gin.Context) {
	page, pageSize := parsePagination(c)
	req := storage.Layer4LogQuery{
		ClientIP:   c.Query("ip"),
		TargetIP:   c.Query("target_ip"),
		ListenPort: c.Query("listen_port"),
		Protocol:   c.Query("protocol"),
		StartTime:  c.Query("start_time"),
		EndTime:    c.Query("end_time"),
		Page:       page,
		PageSize:   pageSize,
	}
	result, err := a.querier.QueryLayer4Logs(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

func (a *App) queryLayer4Intercept(c *gin.Context) {
	page, pageSize := parsePagination(c)
	req := storage.Layer4InterceptLogQuery{
		ClientIP:   c.Query("ip"),
		TargetIP:   c.Query("target_ip"),
		ListenPort: c.Query("listen_port"),
		Protocol:   c.Query("protocol"),
		Reason:     c.Query("reason"),
		StartTime:  c.Query("start_time"),
		EndTime:    c.Query("end_time"),
		Page:       page,
		PageSize:   pageSize,
	}
	result, err := a.querier.QueryLayer4InterceptLogs(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

func (a *App) queryAI(c *gin.Context) {
	page, pageSize := parsePagination(c)
	req := storage.AILogQuery{
		Domain:    c.Query("domain"),
		ClientIP:  c.Query("ip"),
		AIType:    c.Query("ai_type"),
		Model:     c.Query("model"),
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
		Page:      page,
		PageSize:  pageSize,
	}
	result, err := a.querier.QueryAILogs(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// ============ 统计 handlers ============

func (a *App) queryTrafficStats(c *gin.Context) {
	req := storage.TrafficStatsQuery{
		Domain:    c.Query("domain"),
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
	}
	result, err := a.querier.QueryTrafficStats(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

func (a *App) queryTopN(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	req := storage.TopNQuery{
		Domain:    c.Query("domain"),
		Field:     c.DefaultQuery("field", "ip"),
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
		Limit:     limit,
	}
	result, err := a.querier.QueryTopN(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

func (a *App) queryGeoMap(c *gin.Context) {
	req := storage.TrafficStatsQuery{
		Domain:    c.Query("domain"),
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
	}
	result, err := a.querier.QueryGeoMap(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}

// ============ 导出 handlers ============

func (a *App) exportAccess(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	page, pageSize := parsePagination(c)
	// 导出最多 10000 条（覆盖分页）
	if pageSize == 0 || pageSize > 10000 {
		pageSize = 10000
	}
	req := storage.AccessLogQuery{
		Domain:    c.Query("domain"),
		ClientIP:  c.Query("ip"),
		URL:       c.Query("url"),
		Method:    c.Query("method"),
		Status:    c.Query("status"),
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
		EdgeNode:  c.Query("edge_node"),
		Page:      page,
		PageSize:  pageSize,
	}
	result, err := a.querier.ExportAccessLogs(c.Request.Context(), req, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}

	contentType := "text/csv"
	if format == "json" {
		contentType = "application/json"
	}
	c.Data(http.StatusOK, contentType, result.Data)
}

// ============ 指标 handlers ============

func (a *App) getWriterMetrics(c *gin.Context) {
	m := a.writer.GetMetrics()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    m,
	})
}

// ============ 工具函数 ============

// parsePagination 解析分页参数
func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if pageSize <= 0 || pageSize > 1000 {
		pageSize = 50
	}
	return page, pageSize
}
