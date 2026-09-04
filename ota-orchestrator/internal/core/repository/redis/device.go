package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type DeviceCacheRepo struct {
	client         *redis.Client
	key            string
	checkinDataTTL time.Duration
}

func NewDeviceCacheRepo(rdb *redis.Client, config config.CacheConfig) *DeviceCacheRepo {
	return &DeviceCacheRepo{
		client:         rdb,
		key:            "device",
		checkinDataTTL: config.DeviceCheckinDataTTL,
	}
}

func (r *DeviceCacheRepo) ListDeviceCheckinData(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.DeviceCheckinData, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	result := make(map[uuid.UUID]domain.DeviceCheckinData, len(ids))
	cmds := make([]*redis.MapStringStringCmd, len(ids))

	_, err := r.client.Pipelined(ctx, func(p redis.Pipeliner) error {
		for i, id := range ids {
			key := fmt.Sprintf("%s:%s:checkin_data", r.key, id)
			cmds[i] = p.HGetAll(ctx, key)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("redis pipeline hgetall failed: %w", err)
	}

	for i, id := range ids {
		if len(cmds[i].Val()) == 0 {
			continue
		}

		var data domain.DeviceCheckinData
		if err := cmds[i].Scan(&data); err != nil {
			return nil, fmt.Errorf("redis scan device checkin data failed: %w", err)
		}
		result[id] = data
	}

	return result, nil
}

func (r *DeviceCacheRepo) SetCheckinData(ctx context.Context, id uuid.UUID, data domain.DeviceCheckinData) error {
	key := fmt.Sprintf("%s:%s:checkin_data", r.key, id)

	_, err := r.client.Pipelined(ctx, func(p redis.Pipeliner) error {
		p.HSet(ctx, key, data)
		p.Expire(ctx, key, r.checkinDataTTL)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to set device checkin data: %w", err)
	}

	return nil
}
