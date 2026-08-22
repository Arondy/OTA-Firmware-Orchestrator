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

	versions, lastSeen, err := repo.ListDeviceCheckinData(ctx, []uuid.UUID{id})
	require.NoError(t, err)
	assert.Empty(t, versions)
	got, ok := lastSeen[id]
	require.True(t, ok, "lastSeen should contain id")
	assert.WithinDuration(t, ts, got, time.Second)
}

func TestDeviceCacheRepo_SetAndGetCurrentVersion(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour, 24*time.Hour)
	ctx := context.Background()
	id := uuid.New()

	const version = "1.2.3"
	require.NoError(t, repo.SetCurrentVersion(ctx, id, version))

	versions, lastSeen, err := repo.ListDeviceCheckinData(ctx, []uuid.UUID{id})
	require.NoError(t, err)
	assert.Empty(t, lastSeen)
	got, ok := versions[id]
	require.True(t, ok, "versions should contain id")
	assert.Equal(t, version, got)
}

func TestDeviceCacheRepo_SetAndList_BothFields(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour, 24*time.Hour)
	ctx := context.Background()
	id := uuid.New()

	ts := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, repo.SetLastSeen(ctx, id, ts))
	require.NoError(t, repo.SetCurrentVersion(ctx, id, "2.0.0"))

	versions, lastSeen, err := repo.ListDeviceCheckinData(ctx, []uuid.UUID{id})
	require.NoError(t, err)

	gotVersion, ok := versions[id]
	require.True(t, ok)
	assert.Equal(t, "2.0.0", gotVersion)

	gotSeen, ok := lastSeen[id]
	require.True(t, ok)
	assert.WithinDuration(t, ts, gotSeen, time.Second)
}

func TestDeviceCacheRepo_LastSeen_ExpiresAfterTTL(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 1*time.Second, 24*time.Hour)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetLastSeen(ctx, id, time.Now()))

	require.Eventually(t, func() bool {
		_, lastSeen, err := repo.ListDeviceCheckinData(ctx, []uuid.UUID{id})
		if err != nil {
			return false
		}
		_, ok := lastSeen[id]
		return !ok
	}, 5*time.Second, 100*time.Millisecond)
}

func TestDeviceCacheRepo_CurrentVersion_ExpiresAfterTTL(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour, 1*time.Second)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetCurrentVersion(ctx, id, "1.2.3"))

	require.Eventually(t, func() bool {
		versions, _, err := repo.ListDeviceCheckinData(ctx, []uuid.UUID{id})
		if err != nil {
			return false
		}
		_, ok := versions[id]
		return !ok
	}, 5*time.Second, 100*time.Millisecond)
}

func TestDeviceCacheRepo_Get_MissingKey_ReturnsError(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour, 24*time.Hour)
	ctx := context.Background()
	id := uuid.New()

	versions, lastSeen, err := repo.ListDeviceCheckinData(ctx, []uuid.UUID{id})
	require.NoError(t, err)
	assert.Empty(t, versions)
	assert.Empty(t, lastSeen)
	_, okV := versions[id]
	assert.False(t, okV)
	_, okL := lastSeen[id]
	assert.False(t, okL)
}

func TestDeviceCacheRepo_ListDeviceCheckinData_BatchAndEmpty(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour, 24*time.Hour)
	ctx := context.Background()

	// empty ids
	versions, lastSeen, err := repo.ListDeviceCheckinData(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, versions)
	assert.Empty(t, lastSeen)

	versions, lastSeen, err = repo.ListDeviceCheckinData(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Empty(t, versions)
	assert.Empty(t, lastSeen)

	// batch with partial hits
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New() // missing
	ts := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, repo.SetCurrentVersion(ctx, id1, "1.0.0"))
	require.NoError(t, repo.SetLastSeen(ctx, id1, ts))
	require.NoError(t, repo.SetCurrentVersion(ctx, id2, "2.0.0"))
	// id2 has only version, id3 missing

	versions, lastSeen, err = repo.ListDeviceCheckinData(ctx, []uuid.UUID{id1, id2, id3})
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", versions[id1])
	assert.Equal(t, "2.0.0", versions[id2])
	_, ok := versions[id3]
	assert.False(t, ok)

	gotSeen, ok := lastSeen[id1]
	require.True(t, ok)
	assert.WithinDuration(t, ts, gotSeen, time.Second)
	_, ok = lastSeen[id2]
	assert.False(t, ok)
	_, ok = lastSeen[id3]
	assert.False(t, ok)
}
