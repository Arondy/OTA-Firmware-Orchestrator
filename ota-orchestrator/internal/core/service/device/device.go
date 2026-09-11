package device

import (
	"context"

	"github.com/google/uuid"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
)

type DeviceRepo interface {
	List(ctx context.Context, filters domain.DeviceFilters) ([]domain.Device, error)
	Create(ctx context.Context, device domain.Device) (domain.Device, error)
	Decommission(ctx context.Context, id uuid.UUID) (domain.Device, error)
}

type DeviceCacheRepo interface {
	ListDeviceCheckinData(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.DeviceCheckinData, error)
}

type DeviceService struct {
	repo  DeviceRepo
	cache DeviceCacheRepo
}

func NewService(repo DeviceRepo, cache DeviceCacheRepo) *DeviceService {
	return &DeviceService{
		repo:  repo,
		cache: cache,
	}
}

func (s *DeviceService) List(ctx context.Context, filters domain.DeviceFilters) ([]domain.Device, error) {
	devices, err := s.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(devices))
	for i, device := range devices {
		ids[i] = device.ID
	}

	checkinData, err := s.cache.ListDeviceCheckinData(ctx, ids)
	if err != nil {
		logger := config.LoggerFromContext(ctx)
		logger.Errorw("failed to list device checkin data", "error", err)
		return devices, nil
	}

	for i, device := range devices {
		data, exists := checkinData[device.ID]
		if !exists {
			continue
		}
		if data.CurrentVersion != "" {
			devices[i].CurrentVersion = data.CurrentVersion
		}
		if !data.LastSeen.IsZero() {
			seen := data.LastSeen
			devices[i].LastSeen = &seen
		}
	}

	return devices, nil
}

func (s *DeviceService) Create(ctx context.Context, device domain.Device) (domain.Device, error) {
	return s.repo.Create(ctx, device)
}

func (s *DeviceService) Decommission(ctx context.Context, id uuid.UUID) (domain.Device, error) {
	return s.repo.Decommission(ctx, id)
}
