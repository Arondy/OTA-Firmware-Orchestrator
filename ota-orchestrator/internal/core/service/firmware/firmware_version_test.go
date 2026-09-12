package firmware_test

import (
	"context"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/firmware"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/firmware/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestList_DelegatesToRepo(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockFirmwareVersionRepo(t)
	pagination := domain.Pagination{Page: 2, Limit: 10}
	repo.EXPECT().List(mock.Anything, "model-a", pagination).Return([]domain.FirmwareVersion{{ID: uuid.New()}}, nil)

	result, err := firmware.NewService(repo).List(context.Background(), "model-a", pagination)
	require.NoError(t, err)
	require.Len(t, result, 1)
}

func TestList_EmptyFilter_DelegatesToRepo(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockFirmwareVersionRepo(t)
	repo.EXPECT().List(mock.Anything, "", domain.Pagination{}).Return([]domain.FirmwareVersion{}, nil)

	result, err := firmware.NewService(repo).List(context.Background(), "", domain.Pagination{})
	require.NoError(t, err)
	require.Empty(t, result)
}

func TestCreate_DelegatesToRepo(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockFirmwareVersionRepo(t)
	in := domain.FirmwareVersion{DeviceModel: "m", FWVersion: "1.0.0"}

	repo.EXPECT().Create(mock.Anything, in).RunAndReturn(func(_ context.Context, fw domain.FirmwareVersion) (domain.FirmwareVersion, error) {
		fw.ID = uuid.New()
		return fw, nil
	})

	created, err := firmware.NewService(repo).Create(context.Background(), in)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, created.ID)
}
