//go:build integration

package redis

import (
	"context"
	"testing"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDeviceCacheRepo(t *testing.T, checkinDataTTL time.Duration) *DeviceCacheRepo {
	t.Helper()
	return NewDeviceCacheRepo(testRDB, config.CacheConfig{
		DeviceCheckinDataTTL: checkinDataTTL,
	})
}

func deviceCheckinKey(id uuid.UUID) string {
	return "device:" + id.String() + ":checkin_data"
}

func TestDeviceCacheRepo_SetAndGetCheckinData(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour)
	ctx := context.Background()
	id := uuid.New()

	ts := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, repo.SetCheckinData(ctx, id, domain.DeviceCheckinData{CurrentVersion: "1.2.3", LastSeen: ts}))

	got, err := repo.ListDeviceCheckinData(ctx, []uuid.UUID{id})
	require.NoError(t, err)

	data, ok := got[id]
	require.True(t, ok, "result should contain id")
	assert.Equal(t, "1.2.3", data.CurrentVersion)
	assert.WithinDuration(t, ts, data.LastSeen, time.Second)

	fields, err := testRDB.HGetAll(ctx, deviceCheckinKey(id)).Result()
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", fields["current_version"])
	assert.NotEmpty(t, fields["last_seen"])
}

func TestDeviceCacheRepo_SetOverwrites(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour)
	ctx := context.Background()
	id := uuid.New()

	ts := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, repo.SetCheckinData(ctx, id, domain.DeviceCheckinData{CurrentVersion: "1.0.0", LastSeen: ts}))
	require.NoError(t, repo.SetCheckinData(ctx, id, domain.DeviceCheckinData{CurrentVersion: "2.0.0", LastSeen: ts.Add(time.Minute)}))

	got, err := repo.ListDeviceCheckinData(ctx, []uuid.UUID{id})
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", got[id].CurrentVersion)
	assert.WithinDuration(t, ts.Add(time.Minute), got[id].LastSeen, time.Second)
}

func TestDeviceCacheRepo_CheckinData_ExpiresAfterTTL(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 1*time.Second)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetCheckinData(ctx, id, domain.DeviceCheckinData{CurrentVersion: "1.2.3", LastSeen: time.Now()}))

	require.Eventually(t, func() bool {
		got, err := repo.ListDeviceCheckinData(ctx, []uuid.UUID{id})
		if err != nil {
			return false
		}
		_, ok := got[id]
		return !ok
	}, 5*time.Second, 100*time.Millisecond)
}

func TestDeviceCacheRepo_Get_MissingKey_ReturnsEmpty(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour)
	ctx := context.Background()
	id := uuid.New()

	got, err := repo.ListDeviceCheckinData(ctx, []uuid.UUID{id})
	require.NoError(t, err)
	assert.Empty(t, got)
	_, ok := got[id]
	assert.False(t, ok)
}

func TestDeviceCacheRepo_ListDeviceCheckinData_BatchAndEmpty(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour)
	ctx := context.Background()

	// empty ids
	got, err := repo.ListDeviceCheckinData(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = repo.ListDeviceCheckinData(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Empty(t, got)

	// batch with partial hits
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New() // missing
	ts := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, repo.SetCheckinData(ctx, id1, domain.DeviceCheckinData{CurrentVersion: "1.0.0", LastSeen: ts}))
	require.NoError(t, repo.SetCheckinData(ctx, id2, domain.DeviceCheckinData{CurrentVersion: "2.0.0", LastSeen: ts}))
	// id3 missing

	got, err = repo.ListDeviceCheckinData(ctx, []uuid.UUID{id1, id2, id3})
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", got[id1].CurrentVersion)
	assert.Equal(t, "2.0.0", got[id2].CurrentVersion)
	_, ok := got[id3]
	assert.False(t, ok)

	assert.WithinDuration(t, ts, got[id1].LastSeen, time.Second)
	assert.WithinDuration(t, ts, got[id2].LastSeen, time.Second)
}

func TestDeviceCacheRepo_SetAppliesTTL(t *testing.T) {
	resetRedis(t)
	repo := newDeviceCacheRepo(t, 24*time.Hour)
	ctx := context.Background()
	id := uuid.New()

	require.NoError(t, repo.SetCheckinData(ctx, id, domain.DeviceCheckinData{CurrentVersion: "1.0.0", LastSeen: time.Now().UTC()}))
	ttl, err := testRDB.TTL(ctx, deviceCheckinKey(id)).Result()
	require.NoError(t, err)
	assert.Greater(t, int64(ttl), int64(0))
	assert.LessOrEqual(t, int64(ttl), int64(24*time.Hour))
}
