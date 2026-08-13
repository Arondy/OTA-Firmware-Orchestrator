package device

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
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
	repo  DeviceRepo
	cache DeviceCacheRepo
}

func NewService(repo DeviceRepo, cache DeviceCacheRepo) *DeviceService {
	return &DeviceService{
		repo:  repo,
		cache: cache,
	}
}

func (s *DeviceService) List(ctx context.Context) ([]domain.Device, error) {
	devices, err := s.repo.List(ctx)
	for i, device := range devices {
		currentVersion, err := s.cache.GetCurrentVersion(ctx, device.ID)
		if err != nil {
			continue
		}

		lastSeen, err := s.cache.GetLastSeen(ctx, device.ID)
		if err != nil {
			continue
		}

		devices[i].CurrentVersion = currentVersion
		devices[i].LastSeen = &lastSeen
	}

	return devices, err
}

func (s *DeviceService) Create(ctx context.Context, device domain.Device) (domain.Device, error) {
	return s.repo.Create(ctx, device)
}

func (s *DeviceService) Decommission(ctx context.Context, id uuid.UUID) (domain.Device, error) {
	return s.repo.Decommission(ctx, id)
}
