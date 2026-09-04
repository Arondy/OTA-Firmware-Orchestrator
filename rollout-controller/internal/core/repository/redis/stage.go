package redis

import (
	"context"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
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

func (r *StageCacheRepo) GetStageStats(ctx context.Context, id uuid.UUID) (domain.StageStats, error) {
	key := fmt.Sprintf("%s:%s:stats", r.key, id)

	cmd := r.client.HGetAll(ctx, key)
	if err := cmd.Err(); err != nil {
		return domain.StageStats{}, fmt.Errorf("failed to get stage stats: %w", err)
	}

	fields := cmd.Val()
	if len(fields) != 2 {
		return domain.StageStats{}, fmt.Errorf("unexpected stage stats keys number: %d", len(fields))
	}

	var stats domain.StageStats
	err := cmd.Scan(&stats)
	if err != nil {
		return domain.StageStats{}, fmt.Errorf("failed to scan stage stats: %w", err)
	}

	return stats, nil
}
