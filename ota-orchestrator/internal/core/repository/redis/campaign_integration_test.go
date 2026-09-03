//go:build integration

package redis

import (
	"context"
	"fmt"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func campaignStageKey(id uuid.UUID) string {
	return fmt.Sprintf("campaign:%s:current_stage", id)
}

func campaignTargetKey(id uuid.UUID) string {
	return fmt.Sprintf("campaign:%s:current_target_percent", id)
}

func TestCampaignCacheRepo_SetAndGetCheckinData(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()
	stageID := uuid.New()

	require.NoError(t, repo.SetCheckinData(ctx, id, domain.CheckinData{StageID: stageID, TargetPercent: 42}))

	got, err := repo.GetCheckinData(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, stageID, got.StageID)
	assert.Equal(t, 42, got.TargetPercent)
}

func TestCampaignCacheRepo_DeleteCheckinData_RemovesKeys(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()
	stageID := uuid.New()

	require.NoError(t, repo.SetCheckinData(ctx, id, domain.CheckinData{StageID: stageID, TargetPercent: 42}))
	require.NoError(t, repo.DeleteCheckinData(ctx, id))

	_, err := repo.GetCheckinData(ctx, id)
	require.Error(t, err)

	exists, err := testRDB.Exists(ctx, campaignStageKey(id), campaignTargetKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}

func TestCampaignCacheRepo_GetCheckinData_Missing_ReturnsError(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	_, err := repo.GetCheckinData(ctx, id)
	require.Error(t, err)
}

func TestCampaignCacheRepo_GetCheckinData_PartialMissing_ReturnsError(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()
	stageID := uuid.New()

	require.NoError(t, testRDB.Set(ctx, campaignStageKey(id), stageID, 0).Err())

	_, err := repo.GetCheckinData(ctx, id)
	require.Error(t, err)
}

func TestCampaignCacheRepo_KeysHaveNoTTL(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()
	stageID := uuid.New()

	require.NoError(t, repo.SetCheckinData(ctx, id, domain.CheckinData{StageID: stageID, TargetPercent: 10}))

	ttlStage, err := testRDB.TTL(ctx, campaignStageKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(-1), int64(ttlStage))

	ttlPercent, err := testRDB.TTL(ctx, campaignTargetKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(-1), int64(ttlPercent))
}
