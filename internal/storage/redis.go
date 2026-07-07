package storage

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/shieldflow/shieldflow/internal/config"
	"go.uber.org/zap"
)

// InitRedis 初始化 Redis 连接
func InitRedis(cfg *config.RedisConfig, zapLog *zap.Logger) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	zapLog.Info("Redis connected successfully",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.Int("db", cfg.DB),
	)

	return rdb, nil
}
