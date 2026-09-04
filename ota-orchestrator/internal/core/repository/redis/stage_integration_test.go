//go:build integration

package redis

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stageStatsKey(id uuid.UUID) string {
	return fmt.Sprintf("stage:%s:stats", id)
}

func TestStageCacheRepo_SetAndDeleteStageStats(t *testing.T) {
	resetRedis(t)
	repo := NewStageCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetStageStats(ctx, id, domain.StageStats{MinSampleSize: 42, SuccessThreshold: 0.95}))

	fields, err := testRDB.HGetAll(ctx, stageStatsKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, "42", fields["min_sample_size"])
	thrVal, err := strconv.ParseFloat(fields["success_threshold"], 32)
	require.NoError(t, err)
	assert.InDelta(t, 0.95, thrVal, 0.001)

	require.NoError(t, repo.DeleteStageStats(ctx, id))
	exists, err := testRDB.Exists(ctx, stageStatsKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}

func TestStageCacheRepo_KeysHaveNoTTL(t *testing.T) {
	resetRedis(t)
	repo := NewStageCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetStageStats(ctx, id, domain.StageStats{MinSampleSize: 10, SuccessThreshold: 0.9}))

	ttl, err := testRDB.TTL(ctx, stageStatsKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(-1), int64(ttl))
}

func TestStageCacheRepo_Overwrite(t *testing.T) {
	resetRedis(t)
	repo := NewStageCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetStageStats(ctx, id, domain.StageStats{MinSampleSize: 10, SuccessThreshold: 0.9}))
	require.NoError(t, repo.SetStageStats(ctx, id, domain.StageStats{MinSampleSize: 20, SuccessThreshold: 0.95}))

	fields, err := testRDB.HGetAll(ctx, stageStatsKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, "20", fields["min_sample_size"])
	thrVal, err := strconv.ParseFloat(fields["success_threshold"], 32)
	require.NoError(t, err)
	assert.InDelta(t, 0.95, thrVal, 0.001)
}
