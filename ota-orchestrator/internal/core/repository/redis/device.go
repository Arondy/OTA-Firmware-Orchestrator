package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type DeviceCacheRepo struct {
	*redis.Client
	key               string
	lastSeenTTL       time.Duration
	currentVersionTTL time.Duration
}

func NewDeviceCacheRepo(rdb *redis.Client, config config.CacheConfig) *DeviceCacheRepo {
	return &DeviceCacheRepo{
		Client:            rdb,
		key:               "device",
		lastSeenTTL:       config.DeviceLastSeenTTL,
		currentVersionTTL: config.DeviceCurrentVersionTTL,
	}
}

func (r *DeviceCacheRepo) GetCurrentVersion(ctx context.Context, id uuid.UUID) (string, error) {
	key := fmt.Sprintf("%s:%s:current_version", r.key, id)
	return r.Get(ctx, key).Result()
}

func (r *DeviceCacheRepo) SetCurrentVersion(ctx context.Context, id uuid.UUID, currentVersion string) error {
	key := fmt.Sprintf("%s:%s:current_version", r.key, id)
	return r.Set(ctx, key, currentVersion, r.currentVersionTTL).Err()
}

func (r *DeviceCacheRepo) GetLastSeen(ctx context.Context, id uuid.UUID) (time.Time, error) {
	key := fmt.Sprintf("%s:%s:last_seen", r.key, id)
	return r.Get(ctx, key).Time()
}

func (r *DeviceCacheRepo) SetLastSeen(ctx context.Context, id uuid.UUID, lastSeen time.Time) error {
	key := fmt.Sprintf("%s:%s:last_seen", r.key, id)
	return r.Set(ctx, key, lastSeen, r.lastSeenTTL).Err()
}
