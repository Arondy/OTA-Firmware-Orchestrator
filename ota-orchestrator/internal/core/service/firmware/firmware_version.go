package firmware

import (
	"context"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
)

type FirmwareVersionRepo interface {
	ListFirmwareVersions(ctx context.Context) ([]domain.FirmwareVersion, error)
	CreateFirmwareVersion(ctx context.Context, firmwareVersion domain.FirmwareVersion) (domain.FirmwareVersion, error)
}

type FirmwareVersionService struct {
	repo FirmwareVersionRepo
}

func NewService(repo FirmwareVersionRepo) *FirmwareVersionService {
	return &FirmwareVersionService{
		repo: repo,
	}
}

func (s *FirmwareVersionService) List(ctx context.Context) ([]domain.FirmwareVersion, error) {
	return s.repo.ListFirmwareVersions(ctx)
}

func (s *FirmwareVersionService) Create(ctx context.Context, firmwareVersion domain.FirmwareVersion) (domain.FirmwareVersion, error) {
	return s.repo.CreateFirmwareVersion(ctx, firmwareVersion)
}
