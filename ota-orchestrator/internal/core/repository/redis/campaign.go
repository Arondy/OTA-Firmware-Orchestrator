package redis

import (
	"context"
	"fmt"
	"strconv"

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

func (r *CampaignCacheRepo) GetCheckinData(ctx context.Context, id uuid.UUID) (domain.CampaignCheckinData, error) {
	key := fmt.Sprintf("%s:%s:checkin_data", r.key, id)

	cmd := r.client.HGetAll(ctx, key)
	if err := cmd.Err(); err != nil {
		return domain.CampaignCheckinData{}, fmt.Errorf("failed to get checkin data: %w", err)
	}

	fields := cmd.Val()
	if len(fields) != 2 {
		return domain.CampaignCheckinData{}, fmt.Errorf("unexpected checkin data keys number: %d", len(fields))
	}

	stageID, err := uuid.FromBytes([]byte(fields["stage_id"]))
	if err != nil {
		return domain.CampaignCheckinData{}, fmt.Errorf("failed to parse stage_id: %w", err)
	}

	targetPercent, err := strconv.Atoi(fields["target_percent"])
	if err != nil {
		return domain.CampaignCheckinData{}, fmt.Errorf("failed to parse target_percent: %w", err)
	}

	return domain.CampaignCheckinData{StageID: stageID, TargetPercent: targetPercent}, nil
}

func (r *CampaignCacheRepo) SetCheckinData(ctx context.Context, id uuid.UUID, data domain.CampaignCheckinData) error {
	key := fmt.Sprintf("%s:%s:checkin_data", r.key, id)

	err := r.client.HSet(ctx, key, data).Err()
	if err != nil {
		return fmt.Errorf("failed to set checkin data: %w", err)
	}

	return nil
}

func (r *CampaignCacheRepo) DeleteCheckinData(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("%s:%s:checkin_data", r.key, id)

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete checkin data: %w", err)
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
