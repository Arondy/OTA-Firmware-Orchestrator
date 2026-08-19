//go:build integration

package redis

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func campaignStageKey(id, stageID uuid.UUID) string {
	return fmt.Sprintf("campaign:%s:current_stage", id)
}

func campaignTargetKey(id uuid.UUID) string {
	return fmt.Sprintf("campaign:%s:current_target_percent", id)
}

func TestCampaignCacheRepo_SetAndGetCurrentStage(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()
	stageID := uuid.New()

	require.NoError(t, repo.SetCurrentStage(ctx, id, stageID))

	got, err := repo.GetCurrentStage(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, stageID, got)
}

func TestCampaignCacheRepo_DeleteCurrentStage_RemovesKey(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()
	stageID := uuid.New()

	require.NoError(t, repo.SetCurrentStage(ctx, id, stageID))
	require.NoError(t, repo.DeleteCurrentStage(ctx, id))

	_, err := repo.GetCurrentStage(ctx, id)
	require.Error(t, err)
}

func TestCampaignCacheRepo_GetCurrentStage_Missing_ReturnsError(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	_, err := repo.GetCurrentStage(ctx, id)
	require.Error(t, err)
}

func TestCampaignCacheRepo_SetAndGetCurrentTargetPercent(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	const percent = 42
	require.NoError(t, repo.SetCurrentTargetPercent(ctx, id, percent))

	got, err := repo.GetCurrentTargetPercent(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, percent, got)
}

func TestCampaignCacheRepo_DeleteCurrentTargetPercent_RemovesKey(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetCurrentTargetPercent(ctx, id, 42))
	require.NoError(t, repo.DeleteCurrentTargetPercent(ctx, id))

	_, err := repo.GetCurrentTargetPercent(ctx, id)
	require.Error(t, err)
}

func TestCampaignCacheRepo_GetCurrentTargetPercent_Missing_ReturnsError(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	_, err := repo.GetCurrentTargetPercent(ctx, id)
	require.Error(t, err)
}

func TestCampaignCacheRepo_KeysHaveNoTTL(t *testing.T) {
	resetRedis(t)
	repo := NewCampaignCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()
	stageID := uuid.New()

	require.NoError(t, repo.SetCurrentStage(ctx, id, stageID))
	require.NoError(t, repo.SetCurrentTargetPercent(ctx, id, 10))

	ttlStage, err := testRDB.TTL(ctx, campaignStageKey(id, stageID)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(-1), int64(ttlStage))

	ttlPercent, err := testRDB.TTL(ctx, campaignTargetKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(-1), int64(ttlPercent))
}
