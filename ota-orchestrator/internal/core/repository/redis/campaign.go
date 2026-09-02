package redis

import (
	"context"
	"fmt"

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

func (r *CampaignCacheRepo) GetCurrentStage(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	key := fmt.Sprintf("%s:%s:current_stage", r.key, id)
	value, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to get campaign current_stage: %w", err)
	}

	return uuid.FromBytes(value)
}

func (r *CampaignCacheRepo) SetCurrentStage(ctx context.Context, id uuid.UUID, stageID uuid.UUID) error {
	key := fmt.Sprintf("%s:%s:current_stage", r.key, id)
	return r.client.Set(ctx, key, stageID, 0).Err()
}

func (r *CampaignCacheRepo) DeleteCurrentStage(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("%s:%s:current_stage", r.key, id)
	return r.client.Del(ctx, key).Err()
}

func (r *CampaignCacheRepo) GetCurrentTargetPercent(ctx context.Context, id uuid.UUID) (int, error) {
	key := fmt.Sprintf("%s:%s:current_target_percent", r.key, id)
	return r.client.Get(ctx, key).Int()
}

func (r *CampaignCacheRepo) SetCurrentTargetPercent(ctx context.Context, id uuid.UUID, percent int) error {
	key := fmt.Sprintf("%s:%s:current_target_percent", r.key, id)
	return r.client.Set(ctx, key, percent, 0).Err()
}

func (r *CampaignCacheRepo) DeleteCurrentTargetPercent(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("%s:%s:current_target_percent", r.key, id)
	return r.client.Del(ctx, key).Err()
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
