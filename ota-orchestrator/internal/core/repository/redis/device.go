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
	client            *redis.Client
	key               string
	lastSeenTTL       time.Duration
	currentVersionTTL time.Duration
}

func NewDeviceCacheRepo(rdb *redis.Client, config config.CacheConfig) *DeviceCacheRepo {
	return &DeviceCacheRepo{
		client:            rdb,
		key:               "device",
		lastSeenTTL:       config.DeviceLastSeenTTL,
		currentVersionTTL: config.DeviceCurrentVersionTTL,
	}
}

func (r *DeviceCacheRepo) ListDeviceCheckinData(ctx context.Context, ids []uuid.UUID) (versions map[uuid.UUID]string, lastSeen map[uuid.UUID]time.Time, err error) {
	versions = make(map[uuid.UUID]string, len(ids))
	lastSeen = make(map[uuid.UUID]time.Time, len(ids))
	if len(ids) == 0 {
		return versions, lastSeen, nil
	}

	allKeys := make([]string, 2*len(ids))
	for i, id := range ids {
		allKeys[i] = fmt.Sprintf("%s:%s:current_version", r.key, id)
		allKeys[i+len(ids)] = fmt.Sprintf("%s:%s:last_seen", r.key, id)
	}

	res, err := r.client.MGet(ctx, allKeys...).Result()
	if err != nil {
		return nil, nil, fmt.Errorf("redis mget failed: %w", err)
	}

	resVersions := res[:len(ids)]
	resLastSeen := res[len(ids):]

	for i, id := range ids {
		if resVersions[i] != nil {
			if v, ok := resVersions[i].(string); ok {
				versions[id] = v
			}
		}

		if resLastSeen[i] != nil {
			if timeStr, ok := resLastSeen[i].(string); ok {
				if parsedTime, err := time.Parse(time.RFC3339Nano, timeStr); err == nil {
					lastSeen[id] = parsedTime
				}
			}
		}
	}

	return versions, lastSeen, nil
}

func (r *DeviceCacheRepo) SetCurrentVersion(ctx context.Context, id uuid.UUID, currentVersion string) error {
	key := fmt.Sprintf("%s:%s:current_version", r.key, id)
	return r.client.Set(ctx, key, currentVersion, r.currentVersionTTL).Err()
}

func (r *DeviceCacheRepo) SetLastSeen(ctx context.Context, id uuid.UUID, lastSeen time.Time) error {
	key := fmt.Sprintf("%s:%s:last_seen", r.key, id)
	return r.client.Set(ctx, key, lastSeen, r.lastSeenTTL).Err()
}
