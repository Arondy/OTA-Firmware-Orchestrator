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

func stageMinSampleKey(id uuid.UUID) string {
	return fmt.Sprintf("stage:%s:min_sample_size", id)
}

func stageThresholdKey(id uuid.UUID) string {
	return fmt.Sprintf("stage:%s:success_threshold", id)
}

func TestStageCacheRepo_SetAndDeleteStageStats(t *testing.T) {
	resetRedis(t)
	repo := NewStageCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetStageStats(ctx, id, domain.StageStats{MinSampleSize: 42, SuccessThreshold: 0.95}))

	minVal, err := testRDB.Get(ctx, stageMinSampleKey(id)).Int()
	require.NoError(t, err)
	assert.Equal(t, 42, minVal)

	thrVal, err := testRDB.Get(ctx, stageThresholdKey(id)).Float32()
	require.NoError(t, err)
	assert.InDelta(t, 0.95, thrVal, 0.001)

	require.NoError(t, repo.DeleteStageStats(ctx, id))
	exists, err := testRDB.Exists(ctx, stageMinSampleKey(id), stageThresholdKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}

func TestStageCacheRepo_KeysHaveNoTTL(t *testing.T) {
	resetRedis(t)
	repo := NewStageCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetStageStats(ctx, id, domain.StageStats{MinSampleSize: 10, SuccessThreshold: 0.9}))

	ttlMin, err := testRDB.TTL(ctx, stageMinSampleKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(-1), int64(ttlMin))

	ttlThr, err := testRDB.TTL(ctx, stageThresholdKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(-1), int64(ttlThr))
}

func TestStageCacheRepo_Overwrite(t *testing.T) {
	resetRedis(t)
	repo := NewStageCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetStageStats(ctx, id, domain.StageStats{MinSampleSize: 10, SuccessThreshold: 0.9}))
	require.NoError(t, repo.SetStageStats(ctx, id, domain.StageStats{MinSampleSize: 20, SuccessThreshold: 0.95}))

	minVal, err := testRDB.Get(ctx, stageMinSampleKey(id)).Int()
	require.NoError(t, err)
	assert.Equal(t, 20, minVal)

	thrVal, err := testRDB.Get(ctx, stageThresholdKey(id)).Float32()
	require.NoError(t, err)
	assert.InDelta(t, 0.95, thrVal, 0.001)
}
