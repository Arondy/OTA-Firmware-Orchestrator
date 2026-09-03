package redis

import (
	"context"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type CampaignCacheRepo struct {
	client *redis.Client
	key    string
}

func NewCampaignCacheRepo(rdb *redis.Client) *CampaignCacheRepo {
	return &CampaignCacheRepo{
		client: rdb,
		key:    "campaign",
	}
}

func (r *CampaignCacheRepo) GetCheckinData(ctx context.Context, id uuid.UUID) (domain.CheckinData, error) {
	stageKey := fmt.Sprintf("%s:%s:current_stage", r.key, id)
	targetPercentKey := fmt.Sprintf("%s:%s:current_target_percent", r.key, id)
	var bytesCmd *redis.StringCmd
	var targetPercentCmd *redis.StringCmd

	_, err := r.client.Pipelined(ctx, func(p redis.Pipeliner) error {
		bytesCmd = p.Get(ctx, stageKey)
		targetPercentCmd = p.Get(ctx, targetPercentKey)
		return nil
	})
	if err != nil {
		return domain.CheckinData{}, fmt.Errorf("failed to get checkin data in pipeline: %w", err)
	}

	bytes, err := bytesCmd.Bytes()
	if err != nil {
		return domain.CheckinData{}, fmt.Errorf("failed to get bytes from cmd: %w", err)
	}

	targetPercent, err := targetPercentCmd.Int()
	if err != nil {
		return domain.CheckinData{}, fmt.Errorf("failed to get target percent from cmd: %w", err)
	}

	stageID, err := uuid.FromBytes(bytes)
	if err != nil {
		return domain.CheckinData{}, fmt.Errorf("failed to convert bytes to stage id for checkin data: %w", err)
	}

	return domain.CheckinData{
		StageID:       stageID,
		TargetPercent: targetPercent,
	}, nil
}

func (r *CampaignCacheRepo) SetCheckinData(ctx context.Context, id uuid.UUID, data domain.CheckinData) error {
	stageKey := fmt.Sprintf("%s:%s:current_stage", r.key, id)
	targetPercentKey := fmt.Sprintf("%s:%s:current_target_percent", r.key, id)
	var stageCmd *redis.StatusCmd
	var targetPercentCmd *redis.StatusCmd

	_, err := r.client.Pipelined(ctx, func(p redis.Pipeliner) error {
		stageCmd = p.Set(ctx, stageKey, data.StageID, 0)
		targetPercentCmd = p.Set(ctx, targetPercentKey, data.TargetPercent, 0)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to set checkin data in pipeline: %w", err)
	}

	if err = stageCmd.Err(); err != nil {
		return fmt.Errorf("failed to set stage in pipeline: %w", err)
	}
	if err = targetPercentCmd.Err(); err != nil {
		return fmt.Errorf("failed to set target percent in pipeline: %w", err)
	}

	return nil
}

func (r *CampaignCacheRepo) DeleteCheckinData(ctx context.Context, id uuid.UUID) error {
	stageKey := fmt.Sprintf("%s:%s:current_stage", r.key, id)
	targetPercentKey := fmt.Sprintf("%s:%s:current_target_percent", r.key, id)
	var stageCmd *redis.IntCmd
	var targetPercentCmd *redis.IntCmd

	_, err := r.client.Pipelined(ctx, func(p redis.Pipeliner) error {
		stageCmd = p.Del(ctx, stageKey)
		targetPercentCmd = p.Del(ctx, targetPercentKey)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete checkin data in pipeline: %w", err)
	}

	if err = stageCmd.Err(); err != nil {
		return fmt.Errorf("failed to delete stage in pipeline: %w", err)
	}
	if err = targetPercentCmd.Err(); err != nil {
		return fmt.Errorf("failed to delete target percent in pipeline: %w", err)
	}

	return nil
}

func (r *CampaignCacheRepo) AddRunningCampaigns(ctx context.Context, ids ...uuid.UUID) error {
	key := fmt.Sprintf("%s:running_campaigns", r.key)
	anySlice := uuidSliceToAny(ids)
	return r.client.SAdd(ctx, key, anySlice...).Err()
}

func (r *CampaignCacheRepo) RemoveRunningCampaigns(ctx context.Context, ids ...uuid.UUID) error {
	key := fmt.Sprintf("%s:running_campaigns", r.key)
	anySlice := uuidSliceToAny(ids)
	return r.client.SRem(ctx, key, anySlice...).Err()
}

func (r *CampaignCacheRepo) DeleteAllRunningCampaigns(ctx context.Context) error {
	key := fmt.Sprintf("%s:running_campaigns", r.key)
	return r.client.Del(ctx, key).Err()
}

func uuidSliceToAny(ids []uuid.UUID) []any {
	anySlice := make([]any, len(ids))
	for i, id := range ids {
		anySlice[i] = id
	}

	return anySlice
}
