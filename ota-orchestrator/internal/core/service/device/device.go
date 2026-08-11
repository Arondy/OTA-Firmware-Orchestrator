package device

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
)

type DeviceRepo interface {
	List(ctx context.Context) ([]domain.Device, error)
	Create(ctx context.Context, device domain.Device) (domain.Device, error)
	Decommission(ctx context.Context, id uuid.UUID) (domain.Device, error)
}

type DeviceCacheRepo interface {
	GetCurrentVersion(ctx context.Context, id uuid.UUID) (string, error)
	GetLastSeen(ctx context.Context, id uuid.UUID) (time.Time, error)
}

type DeviceService struct {
	deviceRepo  DeviceRepo
	deviceCache DeviceCacheRepo
}

func NewService(deviceRepo DeviceRepo, deviceCache DeviceCacheRepo) *DeviceService {
	return &DeviceService{
		deviceRepo:  deviceRepo,
		deviceCache: deviceCache,
	}
}

func (s *DeviceService) List(ctx context.Context) ([]domain.Device, error) {
	devices, err := s.deviceRepo.List(ctx)
	for i, device := range devices {
		currentVersion, err := s.deviceCache.GetCurrentVersion(ctx, device.ID)
		if err != nil {
			continue
		}

		lastSeen, err := s.deviceCache.GetLastSeen(ctx, device.ID)
		if err != nil {
			continue
		}

		devices[i].CurrentVersion = currentVersion
		devices[i].LastSeen = &lastSeen
	}

	return devices, err
}

func (s *DeviceService) Create(ctx context.Context, device domain.Device) (domain.Device, error) {
	return s.deviceRepo.Create(ctx, device)
}

func (s *DeviceService) Decommission(ctx context.Context, id uuid.UUID) (domain.Device, error) {
	return s.deviceRepo.Decommission(ctx, id)
}
