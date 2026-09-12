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

	list, err := repo.List(context.Background(), "", domain.Pagination{})
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestFirmwareVersionListWithPagination(t *testing.T) {
	resetDB(t)
	repo := NewFirmwareVersionRepo(testDB)
	ctx := context.Background()

	for _, v := range []string{"1.0.0", "2.0.0", "3.0.0"} {
		_, err := repo.Create(ctx, domain.FirmwareVersion{
			DeviceModel: "model-a",
			FWVersion:   v,
			FWChecksum:  "abc",
			BinaryUrl:   "http://example/fw.bin",
		})
		require.NoError(t, err)
	}

	list, err := repo.List(ctx, "", domain.Pagination{Page: 1, Limit: 2})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, "3.0.0", list[0].FWVersion)
	require.Equal(t, "2.0.0", list[1].FWVersion)

	list, err = repo.List(ctx, "", domain.Pagination{Page: 2, Limit: 2})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "1.0.0", list[0].FWVersion)

	list, err = repo.List(ctx, "", domain.Pagination{Page: 3, Limit: 2})
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestFirmwareVersionListWithDeviceModelFilter(t *testing.T) {
	resetDB(t)
	repo := NewFirmwareVersionRepo(testDB)
	ctx := context.Background()

	_, err := repo.Create(ctx, domain.FirmwareVersion{
		DeviceModel: "model-a",
		FWVersion:   "1.0.0",
		FWChecksum:  "abc",
		BinaryUrl:   "http://example/fw.bin",
	})
	require.NoError(t, err)
	_, err = repo.Create(ctx, domain.FirmwareVersion{
		DeviceModel: "model-b",
		FWVersion:   "1.0.0",
		FWChecksum:  "def",
		BinaryUrl:   "http://example/fw2.bin",
	})
	require.NoError(t, err)

	list, err := repo.List(ctx, "model-a", domain.Pagination{})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "model-a", list[0].DeviceModel)

	list, err = repo.List(ctx, "model-missing", domain.Pagination{})
	require.NoError(t, err)
	require.Empty(t, list)
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
