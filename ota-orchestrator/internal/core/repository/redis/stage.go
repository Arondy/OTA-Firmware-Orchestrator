package redis

import (
	"context"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type StageCacheRepo struct {
	client *redis.Client
	key    string
}

func NewStageCacheRepo(rdb *redis.Client) *StageCacheRepo {
	return &StageCacheRepo{
		client: rdb,
		key:    "stage",
	}
}

func (r *StageCacheRepo) SetStageStats(ctx context.Context, id uuid.UUID, stats domain.StageStats) error {
	key := fmt.Sprintf("%s:%s:stats", r.key, id)

	err := r.client.HSet(ctx, key, stats).Err()
	if err != nil {
		return fmt.Errorf("failed to set stage stats: %w", err)
	}

	return nil
}

func (r *StageCacheRepo) DeleteStageStats(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("%s:%s:stats", r.key, id)

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete stage stats: %w", err)
	}

	return nil
}
