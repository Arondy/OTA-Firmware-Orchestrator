package firmware

import (
	"context"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
)

type FirmwareVersionRepo interface {
	List(ctx context.Context) ([]domain.FirmwareVersion, error)
	Create(ctx context.Context, firmwareVersion domain.FirmwareVersion) (domain.FirmwareVersion, error)
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
	return s.repo.List(ctx)
}

func (s *FirmwareVersionService) Create(ctx context.Context, firmwareVersion domain.FirmwareVersion) (domain.FirmwareVersion, error) {
	return s.repo.Create(ctx, firmwareVersion)
}
