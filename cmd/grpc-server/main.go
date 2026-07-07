package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shieldflow/shieldflow/internal/config"
	grpcserver "github.com/shieldflow/shieldflow/internal/grpc"
	"github.com/shieldflow/shieldflow/internal/storage"
	"go.uber.org/zap"
)

// Version 版本号
const Version = "1.0.0"

func main() {
	configPath := flag.String("config", "/etc/shieldflow/grpc.yaml", "配置文件路径")
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

	zapLog.Info("ShieldFlow gRPC Server 启动中...",
		zap.String("version", Version),
		zap.Int("port", cfg.GRPC.Port),
		zap.Bool("tls", cfg.GRPC.TLS),
	)

	// Supervisor 管理名
	zapLog.Info("supervisor name", zap.String("name", "shieldflow-master:shieldflow-grpc-server"))

	// 初始化 PostgreSQL
	db, err := storage.InitPostgreSQL(&cfg.Database, zapLog)
	if err != nil {
		zapLog.Fatal("PostgreSQL 初始化失败", zap.Error(err))
	}

	// 初始化 ClickHouse (可选，用于日志写入)
	_, err = storage.InitClickHouse(&cfg.ClickHouse, zapLog)
	if err != nil {
		zapLog.Warn("ClickHouse 连接失败（日志写入功能将降级到 stdout）", zap.Error(err))
	}

	// 创建 gRPC REST 服务
	srv := grpcserver.NewServer(cfg, db, zapLog)

	// 启动服务
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Run(); err != nil {
			zapLog.Error("gRPC server 运行错误", zap.Error(err))
			errCh <- err
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		zapLog.Info("收到关闭信号，正在停止服务...")
	case err := <-errCh:
		zapLog.Error("服务异常退出", zap.Error(err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = ctx

	if err := srv.Shutdown(); err != nil {
		zapLog.Error("服务关闭失败", zap.Error(err))
		os.Exit(1)
	}

	zapLog.Info("gRPC Server 已关闭")
}
