package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/redis/go-redis/v9"
)

type CampaignCacheRepo struct {
	*redis.Client
	key                    string
	campaignEventIDSeenTTL time.Duration
}

func NewCampaignCacheRepo(rdb *redis.Client, config config.CacheConfig) *CampaignCacheRepo {
	return &CampaignCacheRepo{
		Client:                 rdb,
		key:                    "campaign",
		campaignEventIDSeenTTL: config.CampaignEventIDSeenTTL,
	}
}

func (r *CampaignCacheRepo) UpdateStageResults(ctx context.Context, event domain.UpdateResultsEvent) (int, error) {
	eventSeenKey := fmt.Sprintf("%s:%s:stage:%s:seen:%s", r.key, event.CampaignID, event.StageID, event.EventID)
	stageResultsKey := fmt.Sprintf("%s:%s:stage:%s:%s", r.key, event.CampaignID, event.StageID, event.Result)
	keys := []string{eventSeenKey, stageResultsKey}
	ttl := int64(r.campaignEventIDSeenTTL.Seconds())

	return updateResultScript.Run(ctx, r.Client, keys, ttl).Int()
}

func (r *CampaignCacheRepo) GetCampaignStats(ctx context.Context, id uuid.UUID) (domain.CampaignStats, error) {
	stageID, err := r.getCurrentStage(ctx, id)
	if err != nil {
		return domain.CampaignStats{}, err
	}

	successKey := fmt.Sprintf("%s:%s:stage:%s:%s", r.key, id, stageID, domain.UpdateAttemptsResultSuccess)
	failureKey := fmt.Sprintf("%s:%s:stage:%s:%s", r.key, id, stageID, domain.UpdateAttemptsResultFailure)
	timeoutKey := fmt.Sprintf("%s:%s:stage:%s:%s", r.key, id, stageID, domain.UpdateAttemptsResultTimeout)
	vals, err := r.MGet(ctx, successKey, failureKey, timeoutKey).Result()
	if err != nil {
		return domain.CampaignStats{}, fmt.Errorf("failed to get stats: %w", err)
	}

	stats := make([]int, len(vals))

	for i, val := range vals {
		statStr, ok := val.(string)
		if !ok {
			continue
		}

		statInt, err := strconv.Atoi(statStr)
		if err != nil {
			return domain.CampaignStats{}, fmt.Errorf("failed to parse stat %d (%s): %w", i, val, err)
		}

		stats[i] = statInt
	}

	sum := 0
	for _, val := range stats {
		sum += val
	}

	if sum == 0 {
		return domain.CampaignStats{
			ActiveStageID: stageID,
			SuccessRate:   0,
			SampleSize:    0,
		}, nil
	}

	return domain.CampaignStats{
		ActiveStageID: stageID,
		SuccessRate:   float32(stats[0]) / float32(sum),
		SampleSize:    sum,
	}, nil
}

func (r *CampaignCacheRepo) getCurrentStage(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	key := fmt.Sprintf("%s:%s:current_stage", r.key, id)
	value, err := r.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return uuid.UUID{}, domain.ErrCurrentStageNotFound
	} else if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to get campaign current_stage: %w", err)
	}

	return uuid.FromBytes(value)
}
