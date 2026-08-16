package redis

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context, config config.CacheConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         net.JoinHostPort(config.Host, strconv.Itoa(config.Port)),
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed redis ping: %w", err)
	}

	return rdb, nil
}
