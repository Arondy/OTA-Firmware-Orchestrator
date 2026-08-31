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

func stageMinSampleKey(id uuid.UUID) string {
	return fmt.Sprintf("stage:%s:min_sample_size", id)
}

func stageThresholdKey(id uuid.UUID) string {
	return fmt.Sprintf("stage:%s:success_threshold", id)
}

func TestStageCacheRepo_SetAndDeleteMinSampleSize(t *testing.T) {
	resetRedis(t)
	repo := NewStageCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetMinSampleSize(ctx, id, 42))
	val, err := testRDB.Get(ctx, stageMinSampleKey(id)).Int()
	require.NoError(t, err)
	assert.Equal(t, 42, val)

	require.NoError(t, repo.DeleteMinSampleSize(ctx, id))
	exists, err := testRDB.Exists(ctx, stageMinSampleKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}

func TestStageCacheRepo_SetAndDeleteSuccessThreshold(t *testing.T) {
	resetRedis(t)
	repo := NewStageCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetSuccessThreshold(ctx, id, 0.95))
	val, err := testRDB.Get(ctx, stageThresholdKey(id)).Float32()
	require.NoError(t, err)
	assert.InDelta(t, 0.95, val, 0.001)

	require.NoError(t, repo.DeleteSuccessThreshold(ctx, id))
	exists, err := testRDB.Exists(ctx, stageThresholdKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}

func TestStageCacheRepo_KeysHaveNoTTL(t *testing.T) {
	resetRedis(t)
	repo := NewStageCacheRepo(testRDB)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetMinSampleSize(ctx, id, 10))
	require.NoError(t, repo.SetSuccessThreshold(ctx, id, 0.9))

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

	require.NoError(t, repo.SetMinSampleSize(ctx, id, 10))
	require.NoError(t, repo.SetMinSampleSize(ctx, id, 20))
	val, err := testRDB.Get(ctx, stageMinSampleKey(id)).Int()
	require.NoError(t, err)
	assert.Equal(t, 20, val)
}
