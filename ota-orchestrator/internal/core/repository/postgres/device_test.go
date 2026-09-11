//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeviceCreateGetList(t *testing.T) {
	resetDB(t)
	repo := NewDeviceRepo(testDB)

	created, err := repo.Create(context.Background(), domain.Device{
		DeviceModel:    "model-a",
		CurrentVersion: "1.0.0",
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, domain.DeviceStatusActive, created.Status)
	require.NotZero(t, created.CreatedAt)

	got, err := repo.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "model-a", got.DeviceModel)
	require.Equal(t, "1.0.0", got.CurrentVersion)

	list, err := repo.List(context.Background(), domain.DeviceFilters{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, created.ID, list[0].ID)
}

func TestDevicesAreListedNewestFirst(t *testing.T) {
	resetDB(t)
	repo := NewDeviceRepo(testDB)

	first, err := repo.Create(context.Background(), domain.Device{DeviceModel: "m", CurrentVersion: "1"})
	require.NoError(t, err)
	second, err := repo.Create(context.Background(), domain.Device{DeviceModel: "m", CurrentVersion: "2"})
	require.NoError(t, err)

	list, err := repo.List(context.Background(), domain.DeviceFilters{})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, second.ID, list[0].ID)
	require.Equal(t, first.ID, list[1].ID)
}

func TestDeviceListWithFilters(t *testing.T) {
	resetDB(t)
	repo := NewDeviceRepo(testDB)
	ctx := context.Background()

	activeA, err := repo.Create(ctx, domain.Device{DeviceModel: "model-a", CurrentVersion: "1.0.0"})
	require.NoError(t, err)
	activeB, err := repo.Create(ctx, domain.Device{DeviceModel: "model-b", CurrentVersion: "1.0.0"})
	require.NoError(t, err)
	decommissioned, err := repo.Create(ctx, domain.Device{DeviceModel: "model-a", CurrentVersion: "1.0.0"})
	require.NoError(t, err)
	_, err = repo.Decommission(ctx, decommissioned.ID)
	require.NoError(t, err)

	list, err := repo.List(ctx, domain.DeviceFilters{DeviceModel: "model-a"})
	require.NoError(t, err)
	require.Len(t, list, 2)

	list, err = repo.List(ctx, domain.DeviceFilters{Status: domain.DeviceStatusActive})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.ElementsMatch(t, []uuid.UUID{activeA.ID, activeB.ID}, []uuid.UUID{list[0].ID, list[1].ID})

	list, err = repo.List(ctx, domain.DeviceFilters{DeviceModel: "model-a", Status: domain.DeviceStatusActive})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, activeA.ID, list[0].ID)

	list, err = repo.List(ctx, domain.DeviceFilters{DeviceModel: "model-b", Status: domain.DeviceStatusDecommissioned})
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestDeviceGetNotFound(t *testing.T) {
	resetDB(t)
	repo := NewDeviceRepo(testDB)

	_, err := repo.Get(context.Background(), uuid.New())
	require.ErrorIs(t, err, domain.ErrDeviceNotFound)
}

func TestDeviceDecommission(t *testing.T) {
	resetDB(t)
	repo := NewDeviceRepo(testDB)

	created, err := repo.Create(context.Background(), domain.Device{
		DeviceModel:    "model-b",
		CurrentVersion: "2.0.0",
	})
	require.NoError(t, err)

	decomm, err := repo.Decommission(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.DeviceStatusDecommissioned, decomm.Status)

	got, err := repo.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.DeviceStatusDecommissioned, got.Status)

	_, err = repo.Decommission(context.Background(), uuid.New())
	require.ErrorIs(t, err, domain.ErrDeviceNotFound)
}
