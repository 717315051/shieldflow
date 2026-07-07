package storage

import (
	"fmt"
	"time"

	"github.com/shieldflow/shieldflow/internal/config"
	"github.com/shieldflow/shieldflow/internal/models"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitPostgreSQL 初始化 PostgreSQL 连接
func InitPostgreSQL(cfg *config.DatabaseConfig, zapLog *zap.Logger) (*gorm.DB, error) {
	dsn := cfg.GetDSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect postgresql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移
	if err := db.AutoMigrate(
		&models.User{},
		&models.Node{},
		&models.NodeGroup{},
		&models.Domain{},
		&models.Package{},
		&models.UserPackage{},
		&models.Certificate{},
		&models.AcmeAccount{},
		&models.DNSAccount{},
		&models.BlacklistEntry{},
		&models.ProtectionTemplate{},
		&models.Layer4Forward{},
		&models.CacheTask{},
		&models.SystemSetting{},
		&models.DDoSRule{},
		&models.DDoSBlacklistEntry{},
		&models.LogServerConfig{},
		&models.AIConfigModel{},
		&models.Order{},
		&models.OperationLog{},
		&models.TrafficPackage{},
		&models.DomainPackage{},
		&models.CertificateRequest{},
	); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	zapLog.Info("PostgreSQL connected and migrated successfully",
		zap.String("host", cfg.Host),
		zap.String("database", cfg.Name),
	)

	return db, nil
}
