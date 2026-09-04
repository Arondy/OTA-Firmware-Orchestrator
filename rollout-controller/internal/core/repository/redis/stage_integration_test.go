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

func stageStatsKey(id uuid.UUID) string {
	return fmt.Sprintf("stage:%s:stats", id)
}

func setStageStats(t *testing.T, id uuid.UUID, minSampleSize int, successThreshold float32) {
	t.Helper()
	require.NoError(t, testRDB.HSet(context.Background(), stageStatsKey(id),
		"min_sample_size", minSampleSize,
		"success_threshold", successThreshold,
	).Err())
}

func TestStageCacheRepo_GetStageStats(t *testing.T) {
	resetRedis(t)
	repo := NewStageCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	setStageStats(t, id, 42, 0.95)

	got, err := repo.GetStageStats(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, 42, got.MinSampleSize)
	assert.InDelta(t, 0.95, got.SuccessThreshold, 0.001)
}

func TestStageCacheRepo_GetStageStats_Missing_ReturnsError(t *testing.T) {
	resetRedis(t)
	repo := NewStageCacheRepo(testRDB)
	ctx := context.Background()

	_, err := repo.GetStageStats(ctx, uuid.New())
	require.Error(t, err)
}
