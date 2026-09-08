package redis

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func NewRedisClient(ctx context.Context, config config.CacheConfig, logger *zap.SugaredLogger) (*redis.Client, error) {
	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	logger.Debugf("Connecting to Redis on %s", addr)

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed redis ping: %w", err)
	}

	return rdb, nil
}
