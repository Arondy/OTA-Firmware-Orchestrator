package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
)

type DeviceRepo interface {
	ListDevices(ctx context.Context) ([]domain.Device, error)
	CreateDevice(ctx context.Context, device domain.Device) (domain.Device, error)
	DecommissionDevice(ctx context.Context, id uuid.UUID) (domain.Device, error)
}

type DeviceService struct {
	repo DeviceRepo
}

func NewDeviceService(repo DeviceRepo) *DeviceService {
	return &DeviceService{
		repo: repo,
	}
}

func (s *DeviceService) ListDevices(ctx context.Context) ([]domain.Device, error) {
	return s.repo.ListDevices(ctx)
}

func (s *DeviceService) CreateDevice(ctx context.Context, device domain.Device) (domain.Device, error) {
	return s.repo.CreateDevice(ctx, device)
}

func (s *DeviceService) DecommissionDevice(ctx context.Context, id uuid.UUID) (domain.Device, error) {
	return s.repo.DecommissionDevice(ctx, id)
}
