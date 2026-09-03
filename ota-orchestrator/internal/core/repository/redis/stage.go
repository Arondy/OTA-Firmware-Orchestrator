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
	minSampleSizeKey := fmt.Sprintf("%s:%s:min_sample_size", r.key, id)
	successThresholdKey := fmt.Sprintf("%s:%s:success_threshold", r.key, id)
	var minSampleSizeCmd *redis.StatusCmd
	var successThresholdCmd *redis.StatusCmd

	_, err := r.client.Pipelined(ctx, func(p redis.Pipeliner) error {
		minSampleSizeCmd = p.Set(ctx, minSampleSizeKey, stats.MinSampleSize, 0)
		successThresholdCmd = p.Set(ctx, successThresholdKey, stats.SuccessThreshold, 0)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to set stage stats in pipeline: %w", err)
	}

	if err = minSampleSizeCmd.Err(); err != nil {
		return fmt.Errorf("failed to set min sample size in pipeline: %w", err)
	}
	if err = successThresholdCmd.Err(); err != nil {
		return fmt.Errorf("failed to set success threshold in pipeline: %w", err)
	}

	return nil
}

func (r *StageCacheRepo) DeleteStageStats(ctx context.Context, id uuid.UUID) error {
	minSampleSizeKey := fmt.Sprintf("%s:%s:min_sample_size", r.key, id)
	successThresholdKey := fmt.Sprintf("%s:%s:success_threshold", r.key, id)
	var minSampleSizeCmd *redis.IntCmd
	var successThresholdCmd *redis.IntCmd

	_, err := r.client.Pipelined(ctx, func(p redis.Pipeliner) error {
		minSampleSizeCmd = p.Del(ctx, minSampleSizeKey)
		successThresholdCmd = p.Del(ctx, successThresholdKey)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete stage stats in pipeline: %w", err)
	}

	if err = minSampleSizeCmd.Err(); err != nil {
		return fmt.Errorf("failed to delete min sample size in pipeline: %w", err)
	}
	if err = successThresholdCmd.Err(); err != nil {
		return fmt.Errorf("failed to delete success threshold in pipeline: %w", err)
	}

	return nil
}
