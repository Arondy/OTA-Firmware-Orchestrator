package device_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/device"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/device/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newDeviceMocks(t *testing.T) (*mocks.MockDeviceRepo, *mocks.MockDeviceCacheRepo) {
	return mocks.NewMockDeviceRepo(t), mocks.NewMockDeviceCacheRepo(t)
}

func newService(repo *mocks.MockDeviceRepo, cache *mocks.MockDeviceCacheRepo) *device.DeviceService {
	return device.NewService(repo, cache)
}

func expectCheckinData(cache *mocks.MockDeviceCacheRepo, versions map[uuid.UUID]string, lastSeen map[uuid.UUID]time.Time, err error) {
	cache.EXPECT().ListDeviceCheckinData(mock.Anything, mock.Anything).Return(versions, lastSeen, err)
}

func TestList_RepoFails_ReturnsError(t *testing.T) {
	t.Parallel()
	repo, cache := newDeviceMocks(t)
	repo.EXPECT().List(mock.Anything).Return(nil, errors.New("boom"))

	_, err := newService(repo, cache).List(context.Background())
	require.Error(t, err)
}

func TestList_NoCacheData_ReturnsPostgresValues(t *testing.T) {
	t.Parallel()
	repo, cache := newDeviceMocks(t)
	id := uuid.New()
	pgDevice := domain.Device{ID: id, DeviceModel: "m", CurrentVersion: "1.0.0", LastSeen: nil}

	repo.EXPECT().List(mock.Anything).Return([]domain.Device{pgDevice}, nil)
	expectCheckinData(cache, map[uuid.UUID]string{}, map[uuid.UUID]time.Time{}, nil)

	devices, err := newService(repo, cache).List(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "1.0.0", devices[0].CurrentVersion)
	assert.Nil(t, devices[0].LastSeen)
}

func TestList_CacheHasBothFields_OverridesPostgres(t *testing.T) {
	t.Parallel()
	repo, cache := newDeviceMocks(t)
	id := uuid.New()
	pgDevice := domain.Device{ID: id, DeviceModel: "m", CurrentVersion: "1.0.0", LastSeen: nil}
	cachedVersion := "2.0.0"
	cachedLastSeen := time.Now()

	repo.EXPECT().List(mock.Anything).Return([]domain.Device{pgDevice}, nil)
	expectCheckinData(cache, map[uuid.UUID]string{id: cachedVersion}, map[uuid.UUID]time.Time{id: cachedLastSeen}, nil)

	devices, err := newService(repo, cache).List(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, cachedVersion, devices[0].CurrentVersion)
	require.NotNil(t, devices[0].LastSeen)
	assert.Equal(t, cachedLastSeen.Unix(), devices[0].LastSeen.Unix())
}

func TestList_CacheHasVersionOnly_OverridesVersion(t *testing.T) {
	t.Parallel()
	repo, cache := newDeviceMocks(t)
	id := uuid.New()
	pgDevice := domain.Device{ID: id, DeviceModel: "m", CurrentVersion: "1.0.0", LastSeen: nil}

	repo.EXPECT().List(mock.Anything).Return([]domain.Device{pgDevice}, nil)
	expectCheckinData(cache, map[uuid.UUID]string{id: "2.0.0"}, map[uuid.UUID]time.Time{}, nil)

	devices, err := newService(repo, cache).List(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "2.0.0", devices[0].CurrentVersion)
	assert.Nil(t, devices[0].LastSeen)
}

func TestList_CacheMiss_KeepsPostgresValues(t *testing.T) {
	t.Parallel()
	repo, cache := newDeviceMocks(t)
	pgDevice := domain.Device{ID: uuid.New(), DeviceModel: "m", CurrentVersion: "1.0.0", LastSeen: nil}

	repo.EXPECT().List(mock.Anything).Return([]domain.Device{pgDevice}, nil)
	expectCheckinData(cache, map[uuid.UUID]string{}, map[uuid.UUID]time.Time{}, nil)

	devices, err := newService(repo, cache).List(context.Background())
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "1.0.0", devices[0].CurrentVersion)
	assert.Nil(t, devices[0].LastSeen)
}

func TestCreate_DelegatesToRepo(t *testing.T) {
	t.Parallel()
	repo, cache := newDeviceMocks(t)
	in := domain.Device{DeviceModel: "m", CurrentVersion: "1.0.0"}

	repo.EXPECT().Create(mock.Anything, in).RunAndReturn(func(_ context.Context, d domain.Device) (domain.Device, error) {
		d.ID = uuid.New()
		return d, nil
	})

	created, err := newService(repo, cache).Create(context.Background(), in)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.ID)
}

func TestDecommission_DelegatesToRepo(t *testing.T) {
	t.Parallel()
	repo, cache := newDeviceMocks(t)
	id := uuid.New()

	repo.EXPECT().Decommission(mock.Anything, id).RunAndReturn(func(_ context.Context, d uuid.UUID) (domain.Device, error) {
		return domain.Device{ID: d, Status: domain.DeviceStatusDecommissioned}, nil
	})

	decommissioned, err := newService(repo, cache).Decommission(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.DeviceStatusDecommissioned, decommissioned.Status)
}
