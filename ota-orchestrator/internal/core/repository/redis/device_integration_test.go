//go:build integration

package redis

import (
	"context"
	"testing"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDeviceCacheRepo(t *testing.T, lastSeenTTL, currentVersionTTL time.Duration) *DeviceCacheRepo {
	t.Helper()
	return NewDeviceCacheRepo(testRDB, config.CacheConfig{
		DeviceLastSeenTTL:       lastSeenTTL,
		DeviceCurrentVersionTTL: currentVersionTTL,
	})
}

func TestDeviceCacheRepo_SetAndGetLastSeen(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour, 24*time.Hour)
	ctx := context.Background()
	id := uuid.New()

	ts := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, repo.SetLastSeen(ctx, id, ts))

	got, err := repo.GetLastSeen(ctx, id)
	require.NoError(t, err)
	assert.WithinDuration(t, ts, got, time.Second)
}

func TestDeviceCacheRepo_SetAndGetCurrentVersion(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour, 24*time.Hour)
	ctx := context.Background()
	id := uuid.New()

	const version = "1.2.3"
	require.NoError(t, repo.SetCurrentVersion(ctx, id, version))

	got, err := repo.GetCurrentVersion(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, version, got)
}

func TestDeviceCacheRepo_LastSeen_ExpiresAfterTTL(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 1*time.Second, 24*time.Hour)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetLastSeen(ctx, id, time.Now()))

	require.Eventually(t, func() bool {
		_, err := repo.GetLastSeen(ctx, id)
		return err != nil
	}, 5*time.Second, 100*time.Millisecond)
}

func TestDeviceCacheRepo_CurrentVersion_ExpiresAfterTTL(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour, 1*time.Second)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetCurrentVersion(ctx, id, "1.2.3"))

	require.Eventually(t, func() bool {
		_, err := repo.GetCurrentVersion(ctx, id)
		return err != nil
	}, 5*time.Second, 100*time.Millisecond)
}

func TestDeviceCacheRepo_Get_MissingKey_ReturnsError(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour, 24*time.Hour)
	ctx := context.Background()
	id := uuid.New()

	_, err := repo.GetLastSeen(ctx, id)
	require.Error(t, err)

	_, err = repo.GetCurrentVersion(ctx, id)
	require.Error(t, err)
}
