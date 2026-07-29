package service

import (
	"context"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/google/uuid"
)

type FirmwareVersionRepo interface {
	ListFirmwareVersions(ctx context.Context) ([]domain.FirmwareVersion, error)
	// Для rollout_campaign
	GetFirmwareVersion(ctx context.Context, id uuid.UUID) (domain.FirmwareVersion, error)
	CreateFirmwareVersion(ctx context.Context, firmwareVersion domain.FirmwareVersion) (domain.FirmwareVersion, error)
}

type FirmwareVersionService struct {
	repo FirmwareVersionRepo
}

func NewFirmwareVersionService(repo FirmwareVersionRepo) *FirmwareVersionService {
	return &FirmwareVersionService{
		repo: repo,
	}
}

func (s *FirmwareVersionService) ListFirmwareVersions(ctx context.Context) ([]domain.FirmwareVersion, error) {
	return s.repo.ListFirmwareVersions(ctx)
}

func (s *FirmwareVersionService) CreateFirmwareVersion(ctx context.Context, firmwareVersion domain.FirmwareVersion) (domain.FirmwareVersion, error) {
	return s.repo.CreateFirmwareVersion(ctx, firmwareVersion)
}
