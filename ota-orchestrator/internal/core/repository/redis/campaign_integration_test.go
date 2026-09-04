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

func campaignCheckinKey(id uuid.UUID) string {
	return fmt.Sprintf("campaign:%s:checkin_data", id)
}

func TestCampaignCacheRepo_SetAndGetCheckinData(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()
	stageID := uuid.New()

	require.NoError(t, repo.SetCheckinData(ctx, id, domain.CampaignCheckinData{StageID: stageID, TargetPercent: 42}))

	got, err := repo.GetCheckinData(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, stageID, got.StageID)
	assert.Equal(t, 42, got.TargetPercent)

	fields, err := testRDB.HGetAll(ctx, campaignCheckinKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, string(stageID[:]), fields["stage_id"])
	assert.Equal(t, "42", fields["target_percent"])
}

func TestCampaignCacheRepo_DeleteCheckinData_RemovesKeys(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()
	stageID := uuid.New()

	require.NoError(t, repo.SetCheckinData(ctx, id, domain.CampaignCheckinData{StageID: stageID, TargetPercent: 42}))
	require.NoError(t, repo.DeleteCheckinData(ctx, id))

	_, err := repo.GetCheckinData(ctx, id)
	require.Error(t, err)

	exists, err := testRDB.Exists(ctx, campaignCheckinKey(id)).Result()
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

	require.NoError(t, testRDB.HSet(ctx, campaignCheckinKey(id), "stage_id", stageID).Err())

	_, err := repo.GetCheckinData(ctx, id)
	require.Error(t, err)
}

func TestCampaignCacheRepo_KeysHaveNoTTL(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()
	stageID := uuid.New()

	require.NoError(t, repo.SetCheckinData(ctx, id, domain.CampaignCheckinData{StageID: stageID, TargetPercent: 10}))

	ttl, err := testRDB.TTL(ctx, campaignCheckinKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(-1), int64(ttl))
}
