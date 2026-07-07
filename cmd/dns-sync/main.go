package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"github.com/shieldflow/shieldflow/internal/config"
	"github.com/shieldflow/shieldflow/internal/models"
	"github.com/shieldflow/shieldflow/internal/pkg/dns"
	"github.com/shieldflow/shieldflow/internal/storage"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config 是 dns-sync 服务的配置
type Config struct {
	Server struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"server"`
	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Name     string `mapstructure:"name"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
	} `mapstructure:"database"`
	Sync struct {
		Interval int `mapstructure:"interval"`
		Retry    int `mapstructure:"retry"`
	} `mapstructure:"sync"`
	Providers struct {
		Cloudflare struct {
			Enabled  bool   `mapstructure:"enabled"`
			APIToken string `mapstructure:"api_token"`
		} `mapstructure:"cloudflare"`
		Aliyun struct {
			Enabled         bool   `mapstructure:"enabled"`
			AccessKeyID     string `mapstructure:"access_key_id"`
			AccessKeySecret string `mapstructure:"access_key_secret"`
		} `mapstructure:"aliyun"`
		Tencent struct {
			Enabled   bool   `mapstructure:"enabled"`
			SecretID  string `mapstructure:"secret_id"`
			SecretKey string `mapstructure:"secret_key"`
		} `mapstructure:"tencent"`
	} `mapstructure:"providers"`
	Log struct {
		Level  string `mapstructure:"level"`
		Output string `mapstructure:"output"`
	} `mapstructure:"log"`
}

// loadConfig 加载 dns-sync 配置文件
func loadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetDefault("server.port", 9528)
	v.SetDefault("sync.interval", 300)
	v.SetDefault("sync.retry", 3)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.output", "/var/log/shieldflow/dns-sync.log")

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("dns-sync")
		v.SetConfigType("yaml")
		v.AddConfigPath("/etc/shieldflow/")
		v.AddConfigPath(".")
	}
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

// initLogger 初始化 zap 日志
func initLogger(cfg *Config) (*zap.Logger, error) {
	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		level.SetLevel(zap.InfoLevel)
	}
	zcfg := zap.NewProductionConfig()
	zcfg.Level = level
	if cfg.Log.Output != "" {
		if err := os.MkdirAll("/var/log/shieldflow", 0755); err != nil {
			// 忽略目录创建失败，降级到 stderr
		}
		zcfg.OutputPaths = []string{cfg.Log.Output, "stderr"}
	}
	return zcfg.Build()
}

// buildProviders 根据配置构造 DNS Provider 映射
func buildProviders(cfg *Config, logger *zap.Logger) (map[string]dns.DNSProvider, string) {
	providers := make(map[string]dns.DNSProvider)
	var defaultProvider string

	if cfg.Providers.Cloudflare.Enabled {
		if cfg.Providers.Cloudflare.APIToken == "" {
			logger.Warn("cloudflare provider enabled but api_token is empty")
		} else {
			p := dns.NewCloudflareProvider(cfg.Providers.Cloudflare.APIToken)
			providers[p.ProviderName()] = p
			if defaultProvider == "" {
				defaultProvider = p.ProviderName()
			}
			logger.Info("DNS provider registered", zap.String("provider", p.ProviderName()))
		}
	}
	if cfg.Providers.Aliyun.Enabled {
		if cfg.Providers.Aliyun.AccessKeyID == "" || cfg.Providers.Aliyun.AccessKeySecret == "" {
			logger.Warn("aliyun provider enabled but access_key_id/secret is empty")
		} else {
			p := dns.NewAliyunProvider(cfg.Providers.Aliyun.AccessKeyID, cfg.Providers.Aliyun.AccessKeySecret)
			providers[p.ProviderName()] = p
			if defaultProvider == "" {
				defaultProvider = p.ProviderName()
			}
			logger.Info("DNS provider registered", zap.String("provider", p.ProviderName()))
		}
	}
	if cfg.Providers.Tencent.Enabled {
		if cfg.Providers.Tencent.SecretID == "" || cfg.Providers.Tencent.SecretKey == "" {
			logger.Warn("tencent provider enabled but secret_id/key is empty")
		} else {
			p := dns.NewTencentProvider(cfg.Providers.Tencent.SecretID, cfg.Providers.Tencent.SecretKey)
			providers[p.ProviderName()] = p
			if defaultProvider == "" {
				defaultProvider = p.ProviderName()
			}
			logger.Info("DNS provider registered", zap.String("provider", p.ProviderName()))
		}
	}

	if len(providers) == 0 {
		logger.Warn("no DNS provider enabled; sync will be no-op")
	}
	return providers, defaultProvider
}

// collectEntries 从数据库收集待同步的域名条目
func collectEntries(db *gorm.DB, logger *zap.Logger) []dns.DomainEntry {
	if db == nil {
		return nil
	}
	var domains []models.Domain
	if err := db.Where("status IN ?", []string{"pending", "active"}).Find(&domains).Error; err != nil {
		logger.Error("query domains from db failed", zap.Error(err))
		return nil
	}
	entries := make([]dns.DomainEntry, 0, len(domains))
	for _, d := range domains {
		if d.CNAME == "" {
			continue
		}
		domain, rr, err := splitDomain(d.DomainName)
		if err != nil {
			logger.Warn("skip domain with invalid name", zap.String("domain", d.DomainName), zap.Error(err))
			continue
		}
		entries = append(entries, dns.DomainEntry{
			Domain:     domain,
			CNAMEName:  rr,
			CNAMEValue: d.CNAME,
		})
	}
	return entries
}

// splitDomain 将完整域名拆分为 (rootDomain, subDomain)
// 例如 www.example.com -> ("example.com", "www")
//      example.com -> ("example.com", "@")
func splitDomain(full string) (string, string, error) {
	parts := splitLabels(full)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid domain %q", full)
	}
	if len(parts) == 2 {
		return full, "@", nil
	}
	root := parts[len(parts)-2] + "." + parts[len(parts)-1]
	sub := ""
	for i := 0; i < len(parts)-2; i++ {
		if i > 0 {
			sub += "."
		}
		sub += parts[i]
	}
	return root, sub, nil
}

// splitLabels 按 "." 切分域名标签（去尾部点）
func splitLabels(s string) []string {
	for len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func main() {
	configPath := flag.String("config", "/etc/shieldflow/dns-sync.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	logger, err := initLogger(cfg)
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("ShieldFlow DNS 同步服务启动中...",
		zap.Int("port", cfg.Server.Port),
		zap.Int("interval_sec", cfg.Sync.Interval),
		zap.String("config", *configPath))

	// 构造 Providers
	providers, defaultProvider := buildProviders(cfg, logger)

	// 初始化数据库（可选，用于从 domains 表读取待同步域名）
	var db *gorm.DB
	dbCfg := &config.DatabaseConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Name:     cfg.Database.Name,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
	}
	db, err = storage.InitPostgreSQL(dbCfg, logger)
	if err != nil {
		logger.Warn("数据库连接失败（将从 API 接收同步任务）", zap.Error(err))
		db = nil
	}

	// 创建 DNS 管理器
	mgr := dns.NewManager(providers, defaultProvider, dns.SyncConfig{
		Interval: cfg.Sync.Interval,
		Retry:    cfg.Sync.Retry,
	}, logger)

	// 定时同步
	interval := time.Duration(cfg.Sync.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 立即执行一次
	runSync := func() {
		entries := collectEntries(db, logger)
		if len(entries) == 0 {
			logger.Info("no entries to sync")
			return
		}
		mgr.Sync(entries)
	}
	go runSync()

	// HTTP API
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		go runSync()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"sync triggered"}`))
	})
	mux.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		report := mgr.LastReport()
		w.Header().Set("Content-Type", "application/json")
		if report == nil {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"no report yet"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(report)
	})
	mux.HandleFunc("/providers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"providers": mgr.Providers(),
			"default":   defaultProvider,
		})
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// 启动 HTTP 服务
	go func() {
		logger.Info("HTTP API 监听中", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP 服务失败", zap.Error(err))
		}
	}()

	// 定时同步循环
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runSync()
			}
		}
	}()

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("收到退出信号，正在关闭...", zap.String("signal", sig.String()))

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP 服务关闭失败", zap.Error(err))
	}
	cancel()
	logger.Info("DNS 同步服务已退出")
}


