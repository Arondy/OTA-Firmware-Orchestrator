//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFirmwareVersionCreateGetList(t *testing.T) {
	resetDB(t)
	repo := NewFirmwareVersionRepo(testDB)

	created, err := repo.Create(context.Background(), domain.FirmwareVersion{
		DeviceModel: "model-a",
		FWVersion:   "1.0.0",
		FWChecksum:  "abc",
		BinaryUrl:   "http://example/fw.bin",
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, created.ID)

	got, err := repo.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "model-a", got.DeviceModel)
	require.Equal(t, "1.0.0", got.FWVersion)
	require.Equal(t, "abc", got.FWChecksum)
	require.Equal(t, "http://example/fw.bin", got.BinaryUrl)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestFirmwareVersionDuplicate(t *testing.T) {
	resetDB(t)
	repo := NewFirmwareVersionRepo(testDB)

	_, err := repo.Create(context.Background(), domain.FirmwareVersion{
		DeviceModel: "model-a",
		FWVersion:   "1.0.0",
		FWChecksum:  "abc",
		BinaryUrl:   "http://example/fw.bin",
	})
	require.NoError(t, err)

	_, err = repo.Create(context.Background(), domain.FirmwareVersion{
		DeviceModel: "model-a",
		FWVersion:   "1.0.0",
		FWChecksum:  "def",
		BinaryUrl:   "http://example/fw2.bin",
	})
	require.ErrorIs(t, err, domain.ErrFirmwareVersionAlreadyExists)
}

func TestFirmwareVersionGetNotFound(t *testing.T) {
	resetDB(t)
	repo := NewFirmwareVersionRepo(testDB)

	_, err := repo.Get(context.Background(), uuid.New())
	require.ErrorIs(t, err, domain.ErrFirmwareVersionNotFound)
}
