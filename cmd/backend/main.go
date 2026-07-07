package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shieldflow/shieldflow/internal/config"
	"github.com/shieldflow/shieldflow/internal/handlers"
	"github.com/shieldflow/shieldflow/internal/storage"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	configPath := flag.String("config", "", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	var zapLog *zap.Logger
	if cfg.IsProduction() {
		zapLog, err = zap.NewProduction()
	} else {
		zapLog, err = zap.NewDevelopment()
	}
	if err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer zapLog.Sync()

	zapLog.Info("ShieldFlow 后端服务启动中...",
		zap.String("mode", cfg.Server.Mode),
		zap.String("version", "1.2.0"),
	)

	// 初始化 PostgreSQL
	db, err := storage.InitPostgreSQL(&cfg.Database, zapLog)
	if err != nil {
		zapLog.Fatal("PostgreSQL 初始化失败", zap.Error(err))
	}

	// 初始化 ClickHouse (可选)
	var ch interface{}
	chConn, err := storage.InitClickHouse(&cfg.ClickHouse, zapLog)
	if err != nil {
		zapLog.Warn("ClickHouse 连接失败（日志功能将不可用）", zap.Error(err))
	} else {
		ch = chConn
	}

	// 初始化 Redis (可选)
	var rdb interface{}
	redisClient, err := storage.InitRedis(&cfg.Redis, zapLog)
	if err != nil {
		zapLog.Warn("Redis 连接失败（缓存功能将降级）", zap.Error(err))
	} else {
		rdb = redisClient
	}

	// 创建默认管理员账号
	createDefaultAdmin(db, zapLog)

	// 设置路由
	router := handlers.SetupRouter(cfg, db, ch, rdb)

	// 启动 HTTP 服务
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// 优雅启动
	go func() {
		zapLog.Info("HTTP 服务监听", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLog.Fatal("HTTP 服务启动失败", zap.Error(err))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zapLog.Info("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zapLog.Fatal("服务关闭失败", zap.Error(err))
	}

	zapLog.Info("服务已关闭")
}

// createDefaultAdmin 创建默认管理员账号
func createDefaultAdmin(db *gorm.DB, log *zap.Logger) {
	var count int64
	db.Table("users").Where("role = ?", "admin").Count(&count)
	if count > 0 {
		return
	}

	// 创建默认管理员 admin / admin123
	hashedPassword, _ := bcryptPassword("admin123")
	result := db.Exec(
		`INSERT INTO users (username, password_hash, role, status, created_at, updated_at) VALUES (?, ?, 'admin', 'active', NOW(), NOW())`,
		"admin", hashedPassword,
	)
	if result.Error != nil {
		log.Error("创建默认管理员失败", zap.Error(result.Error))
		return
	}
	log.Info("默认管理员账号已创建", zap.String("username", "admin"), zap.String("password", "admin123"))
}
