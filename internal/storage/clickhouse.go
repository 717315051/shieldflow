package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/shieldflow/shieldflow/internal/config"
	"go.uber.org/zap"
)

// InitClickHouse 初始化 ClickHouse 连接
func InitClickHouse(cfg *config.ClickHouseConfig, zapLog *zap.Logger) (driver.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		MaxOpenConns:      50,
		MaxIdleConns:      10,
		ConnMaxLifetime:   time.Hour,
		DialTimeout:       10 * time.Second,
		Compression:       &clickhouse.Compression{Method: clickhouse.CompressionLZ4},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open clickhouse: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	zapLog.Info("ClickHouse connected successfully",
		zap.String("host", cfg.Host),
		zap.String("database", cfg.Database),
	)

	return conn, nil
}
