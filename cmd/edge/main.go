// ShieldFlow Edge 边缘节点入口。
//
// 功能：
//   - 加载 /etc/shieldflow/edge.yaml 配置
//   - 启动反向代理（HTTP 80 / HTTPS 443）
//   - 启动健康检查服务（9527 /ping）
//   - 连接主控 gRPC（定时拉取配置、上报日志）
//   - Supervisor: shieldflow:shieldflow-edge
//
// 进程名：shieldflow:shieldflow-edge
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"github.com/shieldflow/shieldflow/internal/grpc"
	"github.com/shieldflow/shieldflow/internal/pkg/bot"
	"github.com/shieldflow/shieldflow/internal/pkg/cache"
	"github.com/shieldflow/shieldflow/internal/pkg/ddos"
	"github.com/shieldflow/shieldflow/internal/pkg/proxy"
	"github.com/shieldflow/shieldflow/internal/pkg/waf"
	zcc "github.com/shieldflow/shieldflow/internal/waf"
	"go.uber.org/zap"
)

// EdgeConfig 边缘节点配置（对应 edge.yaml）。
type EdgeConfig struct {
	Node   NodeConfig   `mapstructure:"node"`
	GRPC   GRPCConfig   `mapstructure:"grpc"`
	Proxy  ProxyConfig  `mapstructure:"proxy"`
	WAF    WAFConfig    `mapstructure:"waf"`
	Cache  CacheConfig  `mapstructure:"cache"`
	DDoS   DDoSConfig   `mapstructure:"ddos"`
	CC     CCConfig     `mapstructure:"cc"`
	Bot    BotConfig    `mapstructure:"bot"`
	Origins []OriginConfig `mapstructure:"origins"`
}

type NodeConfig struct {
	ID       string `mapstructure:"id"`
	Region   string `mapstructure:"region"`
	LicenseKey string `mapstructure:"license_key"`
}

type GRPCConfig struct {
	Server string `mapstructure:"server"`
	TLS    bool   `mapstructure:"tls"`
	Cert   string `mapstructure:"cert"`
	Key    string `mapstructure:"key"`
	Token  string `mapstructure:"token"`
}

type ProxyConfig struct {
	HTTPPort  int    `mapstructure:"http_port"`
	HTTPSPort int    `mapstructure:"https_port"`
	CertFile  string `mapstructure:"cert_file"`
	KeyFile   string `mapstructure:"key_file"`
}

type WAFConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Mode      string `mapstructure:"mode"`       // block / detect
	Threshold int    `mapstructure:"threshold"`  // 0-100
}

type CacheConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Path     string `mapstructure:"path"`
	MaxSize  string `mapstructure:"max_size"`
	TTL      string `mapstructure:"ttl"`
	Compress string `mapstructure:"compress"` // gzip / br
}

type DDoSConfig struct {
	Enabled               bool `mapstructure:"enabled"`
	MaxConnectionsPerIP   int  `mapstructure:"max_connections_per_ip"`
	NewConnectionsPerSec  int  `mapstructure:"new_connections_per_sec"`
	MaxPacketsPerSec      int  `mapstructure:"max_packets_per_sec"`
	AutoBanEnabled        bool `mapstructure:"auto_ban_enabled"`
	BanThresholdConnections int `mapstructure:"ban_threshold_connections"`
	BanThresholdPackets     int `mapstructure:"ban_threshold_packets"`
	BanDurationSeconds      int `mapstructure:"ban_duration_seconds"`
	Blacklist             []string `mapstructure:"blacklist"`
	Whitelist             []string `mapstructure:"whitelist"`
}

type CCConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	GlobalRateLimit int   `mapstructure:"global_rate_limit"`
	GlobalWindow   string `mapstructure:"global_window"`
	ChallengeType  string `mapstructure:"challenge_type"`
	WaitingRoom    WaitingRoomConfig `mapstructure:"waiting_room"`
}

type WaitingRoomConfig struct {
	Enabled       bool `mapstructure:"enabled"`
	MaxConcurrent int  `mapstructure:"max_concurrent"`
	BaseWaitMs    int  `mapstructure:"base_wait_ms"`
	IncrementMs   int  `mapstructure:"increment_ms"`
	MaxWaitMs     int  `mapstructure:"max_wait_ms"`
}

type BotConfig struct {
	Enabled            bool `mapstructure:"enabled"`
	AllowSearchEngines bool `mapstructure:"allow_search_engines"`
	BlockScanners      bool `mapstructure:"block_scanners"`
	BlockScrapers      bool `mapstructure:"block_scrapers"`
	BlockNoUA          bool `mapstructure:"block_no_ua"`
}

type OriginConfig struct {
	Addr   string `mapstructure:"addr"`
	Weight int    `mapstructure:"weight"`
	Scheme string `mapstructure:"scheme"`
	Host   string `mapstructure:"host"`
}

func main() {
	// 设置进程标题（Supervisor 识别为 shieldflow:shieldflow-edge）。
	setProcessTitle("shieldflow:shieldflow-edge")

	configPath := flag.String("config", "/etc/shieldflow/edge.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置。
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志。
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("ShieldFlow Edge 启动中",
		zap.String("node_id", cfg.Node.ID),
		zap.String("region", cfg.Node.Region),
		zap.String("config", *configPath),
	)

	// 初始化各防护组件。
	ddosGuard := initDDoS(cfg, logger)
	ccEngine := initCC(cfg, logger)
	wafEngine := initWAF(cfg, logger)
	botEngine := initBot(cfg, logger)
	cacheStore := initCache(cfg, logger)

	// 构建源站池。
	rp, err := buildReverseProxy(cfg, logger)
	if err != nil {
		logger.Fatal("反向代理初始化失败", zap.Error(err))
	}

	// 构建中间件链。
	chain := proxy.NewMiddlewareChain(ddosGuard, ccEngine, wafEngine, botEngine, cacheStore)
	handler := chain.Handler(rp)

	// 启动 HTTP 服务。
	httpSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Proxy.HTTPPort),
		Handler: handler,
	}

	// 启动 HTTPS 服务。
	var httpsSrv *http.Server
	if cfg.Proxy.HTTPSPort > 0 {
		httpsSrv = &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Proxy.HTTPSPort),
			Handler: handler,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
	}

	// 启动健康检查服务。
	healthSrv := startHealthCheck(cfg, logger)

	// 连接主控 gRPC。
	go connectController(cfg, logger, ddosGuard, ccEngine, wafEngine, botEngine, cacheStore)

	// 启动 HTTP。
	go func() {
		logger.Info("HTTP 监听", zap.Int("port", cfg.Proxy.HTTPPort))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP 服务异常", zap.Error(err))
		}
	}()

	// 启动 HTTPS。
	if httpsSrv != nil {
		go func() {
			logger.Info("HTTPS 监听", zap.Int("port", cfg.Proxy.HTTPSPort))
			if err := httpsSrv.ListenAndServeTLS(cfg.Proxy.CertFile, cfg.Proxy.KeyFile); err != nil && err != http.ErrServerClosed {
				logger.Error("HTTPS 服务异常", zap.Error(err))
			}
		}()
	}

	// 等待退出信号。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	logger.Info("收到退出信号，正在关闭...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	if httpsSrv != nil {
		_ = httpsSrv.Shutdown(ctx)
	}
	_ = healthSrv.Shutdown(ctx)
	logger.Info("ShieldFlow Edge 已关闭")
}

// loadConfig 加载 edge.yaml 配置文件。
func loadConfig(path string) (*EdgeConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// 默认值。
	v.SetDefault("proxy.http_port", 80)
	v.SetDefault("proxy.https_port", 443)
	v.SetDefault("waf.enabled", true)
	v.SetDefault("waf.mode", "block")
	v.SetDefault("waf.threshold", 50)
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.path", "/var/cache/shieldflow")
	v.SetDefault("cache.max_size", "10GB")
	v.SetDefault("cache.ttl", "10m")
	v.SetDefault("cache.compress", "br")
	v.SetDefault("ddos.enabled", true)
	v.SetDefault("ddos.max_connections_per_ip", 1000)
	v.SetDefault("ddos.new_connections_per_sec", 50)
	v.SetDefault("ddos.max_packets_per_sec", 2000)
	v.SetDefault("ddos.auto_ban_enabled", true)
	v.SetDefault("ddos.ban_threshold_connections", 2000)
	v.SetDefault("ddos.ban_threshold_packets", 5000)
	v.SetDefault("ddos.ban_duration_seconds", 3600)
	v.SetDefault("cc.enabled", true)
	v.SetDefault("cc.global_rate_limit", 60)
	v.SetDefault("cc.global_window", "1m")
	v.SetDefault("cc.challenge_type", "5s_shield")
	v.SetDefault("bot.enabled", true)
	v.SetDefault("bot.allow_search_engines", true)
	v.SetDefault("bot.block_scanners", true)
	v.SetDefault("bot.block_scrapers", false)
	v.SetDefault("bot.block_no_ua", true)

	// 环境变量覆盖（前缀 ZHIYUN_）。
	v.SetEnvPrefix("ZHIYUN")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}

	var cfg EdgeConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	return &cfg, nil
}

// initDDoS 初始化 DDoS 防护。
func initDDoS(cfg *EdgeConfig, log *zap.Logger) *ddos.Guard {
	if !cfg.DDoS.Enabled {
		return nil
	}
	g := ddos.NewGuard(ddos.Config{
		MaxConnectionsPerIP:    cfg.DDoS.MaxConnectionsPerIP,
		NewConnectionsPerSec:   cfg.DDoS.NewConnectionsPerSec,
		MaxPacketsPerSec:       cfg.DDoS.MaxPacketsPerSec,
		AutoBanEnabled:         cfg.DDoS.AutoBanEnabled,
		BanThresholdConnections: cfg.DDoS.BanThresholdConnections,
		BanThresholdPackets:     cfg.DDoS.BanThresholdPackets,
		BanDurationSeconds:      cfg.DDoS.BanDurationSeconds,
		Blacklist:              cfg.DDoS.Blacklist,
		Whitelist:              cfg.DDoS.Whitelist,
	})
	log.Info("DDoS 防护已启用")
	return g
}

// initCC 初始化 CC 防护。
func initCC(cfg *EdgeConfig, log *zap.Logger) *zcc.Engine {
	if !cfg.CC.Enabled {
		return nil
	}
	window, err := time.ParseDuration(cfg.CC.GlobalWindow)
	if err != nil || window <= 0 {
		window = time.Minute
	}
	e := zcc.NewEngine(zcc.CCConfig{
		GlobalRateLimit: cfg.CC.GlobalRateLimit,
		GlobalWindow:    window,
		ChallengeType:   zcc.ChallengeType(cfg.CC.ChallengeType),
		WaitingRoom: zcc.WaitingRoomConfig{
			Enabled:       cfg.CC.WaitingRoom.Enabled,
			MaxConcurrent: cfg.CC.WaitingRoom.MaxConcurrent,
			BaseWaitMs:    cfg.CC.WaitingRoom.BaseWaitMs,
			IncrementMs:   cfg.CC.WaitingRoom.IncrementMs,
			MaxWaitMs:     cfg.CC.WaitingRoom.MaxWaitMs,
		},
	})
	log.Info("CC 防护已启用")
	return e
}

// initWAF 初始化 WAF。
func initWAF(cfg *EdgeConfig, log *zap.Logger) *waf.Engine {
	if !cfg.WAF.Enabled {
		return nil
	}
	mode := waf.ModeBlock
	if cfg.WAF.Mode == "detect" || cfg.WAF.Mode == "observe" {
		mode = waf.ModeDetect
	}
	e := waf.NewEngine(mode, cfg.WAF.Threshold)
	log.Info("WAF 已启用", zap.String("mode", string(mode)), zap.Int("threshold", cfg.WAF.Threshold))
	return e
}

// initBot 初始化 Bot 检测。
func initBot(cfg *EdgeConfig, log *zap.Logger) *bot.Engine {
	if !cfg.Bot.Enabled {
		return nil
	}
	e := bot.NewEngine()
	e.SetPolicy(cfg.Bot.AllowSearchEngines, cfg.Bot.BlockScanners, cfg.Bot.BlockScrapers, cfg.Bot.BlockNoUA)
	log.Info("Bot 检测已启用")
	return e
}

// initCache 初始化缓存。
func initCache(cfg *EdgeConfig, log *zap.Logger) *cache.Store {
	if !cfg.Cache.Enabled {
		return nil
	}
	s, err := cache.NewStore(cache.Config{
		Enabled:  cfg.Cache.Enabled,
		Path:     cfg.Cache.Path,
		MaxSize:  cfg.Cache.MaxSize,
		TTL:      cfg.Cache.TTL,
		Compress: cfg.Cache.Compress,
	})
	if err != nil {
		log.Error("缓存初始化失败", zap.Error(err))
		return nil
	}
	log.Info("缓存已启用", zap.String("path", cfg.Cache.Path), zap.String("ttl", cfg.Cache.TTL))
	return s
}

// buildReverseProxy 构建反向代理。
func buildReverseProxy(cfg *EdgeConfig, log *zap.Logger) (*proxy.ReverseProxy, error) {
	if len(cfg.Origins) == 0 {
		return nil, fmt.Errorf("未配置源站")
	}
	origins := make([]*proxy.Origin, 0, len(cfg.Origins))
	for _, o := range cfg.Origins {
		origins = append(origins, &proxy.Origin{
			Addr:   o.Addr,
			Weight: o.Weight,
			Scheme: o.Scheme,
			Host:   o.Host,
		})
	}
	pool := proxy.NewOriginPool(proxy.StrategyIPHash, origins)
	pool.StartHealthCheck("/healthz", 10*time.Second)
	return proxy.NewReverseProxy(pool)
}

// startHealthCheck 启动健康检查 HTTP 服务（端口 9527）。
func startHealthCheck(cfg *EdgeConfig, log *zap.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "ok",
			"node_id":  cfg.Node.ID,
			"region":   cfg.Node.Region,
			"version":  "1.0.0",
			"uptime":   int64(time.Since(time.Now()).Seconds()), // 简化
			"goroutines": runtime.NumGoroutine(),
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: ":9527", Handler: mux}
	go func() {
		log.Info("健康检查服务监听", zap.String("addr", ":9527"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("健康检查服务异常", zap.Error(err))
		}
	}()
	return srv
}

// connectController 连接主控 gRPC 服务，定时拉取配置与上报日志。
//
// 注意：本项目的 gRPC 实际是基于 HTTP+JSON 的 REST 接口（见 internal/grpc）。
// 这里作为边缘节点客户端，定期向主控拉取配置和上报日志。
func connectController(
	cfg *EdgeConfig,
	log *zap.Logger,
	ddosGuard *ddos.Guard,
	ccEngine *zcc.Engine,
	wafEngine *waf.Engine,
	botEngine *bot.Engine,
	cacheStore *cache.Store,
) {
	if cfg.GRPC.Server == "" {
		log.Warn("未配置主控 gRPC 地址，跳过连接")
		return
	}

	// 解析 node_id（假设为整数）。
	var nodeID uint
	fmt.Sscanf(cfg.Node.ID, "%d", &nodeID)

	client := grpc.NewClient("http://"+cfg.GRPC.Server, cfg.GRPC.Token)

	var wg sync.WaitGroup

	// 1. 定时拉取配置（每 30 秒）。
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			resp, err := client.GetNodeConfig(nodeID)
			if err != nil {
				log.Warn("拉取节点配置失败", zap.Error(err))
				continue
			}
			if resp.Code == 0 && resp.Data != nil {
				applyNodeConfig(resp.Data, cfg, log, ddosGuard, ccEngine, wafEngine, botEngine)
			}
		}
	}()

	// 2. 定时心跳上报（每 10 秒）。
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			metrics := map[string]interface{}{
				"node_id":    nodeID,
				"region":     cfg.Node.Region,
				"goroutines": runtime.NumGoroutine(),
				"timestamp":  time.Now().Unix(),
			}
			if _, err := client.SendHeartbeat(nodeID, metrics); err != nil {
				log.Warn("心跳上报失败", zap.Error(err))
			}
		}
	}()

	// 3. 定时上报访问日志（每 60 秒，简化为空批次示例）。
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			// 实际应从日志缓冲区批量取出。
			batch := grpc.AccessLogBatch{NodeID: nodeID, Logs: []grpc.AccessLogEntry{}}
			if _, err := client.ReportAccessLogs(batch); err != nil {
				log.Warn("访问日志上报失败", zap.Error(err))
			}
		}
	}()

	wg.Wait()
}

// applyNodeConfig 应用主控下发的节点配置。
func applyNodeConfig(data interface{}, cfg *EdgeConfig, log *zap.Logger,
	ddosGuard *ddos.Guard, ccEngine *zcc.Engine, wafEngine *waf.Engine, botEngine *bot.Engine) {
	// data 是 GetNodeConfigResponse 的 JSON。
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	var resp struct {
		Domains      json.RawMessage `json:"domains"`
		DDoSRules    json.RawMessage `json:"ddos_rules"`
		ConfigVersion string         `json:"config_version"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		log.Warn("解析节点配置失败", zap.Error(err))
		return
	}
	log.Info("应用节点配置", zap.String("config_version", resp.ConfigVersion))

	// 应用 DDoS 规则。
	if ddosGuard != nil && len(resp.DDoSRules) > 0 {
		var ddosCfg ddos.Config
		if err := json.Unmarshal(resp.DDoSRules, &ddosCfg); err == nil {
			ddosGuard.UpdateConfig(ddosCfg)
		}
	}
}

// setProcessTitle 设置进程标题（Linux 下通过 argv[0] 修改）。
//
// Supervisor 可通过进程名识别为 shieldflow:shieldflow-edge。
func setProcessTitle(title string) {
	// Go 标准库未提供直接设置进程名的 API；
	// 在 Linux 下可通过修改 argv[0] 实现，但需要 cgo。
	// 这里仅设置环境变量标记，实际进程名由 systemd/supervisor 配置保证。
	_ = os.Setenv("ShieldFlow_PROCESS_NAME", title)
}
