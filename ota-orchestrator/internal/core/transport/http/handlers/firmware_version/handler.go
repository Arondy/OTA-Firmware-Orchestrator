package firmware_version

import (
	"context"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
)

type FirmwareVersionService interface {
	List(ctx context.Context, deviceModel string, pagination domain.Pagination) ([]domain.FirmwareVersion, error)
	Create(ctx context.Context, firmwareVersion domain.FirmwareVersion) (domain.FirmwareVersion, error)
}

type FirmwareVersionHandler struct {
	svc FirmwareVersionService
}

func NewFirmwareVersionHandler(svc FirmwareVersionService) *FirmwareVersionHandler {
	return &FirmwareVersionHandler{
		svc: svc,
	}
}

type FirmwareVersionResponse struct {
	ID          uuid.UUID `json:"id"`
	DeviceModel string    `json:"device_model"`
	FWVersion   string    `json:"fw_version"`
	FWChecksum  string    `json:"fw_checksum"`
	BinaryUrl   string    `json:"binary_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func FirmwareVersionFromDomain(fwv domain.FirmwareVersion) FirmwareVersionResponse {
	return FirmwareVersionResponse{
		ID:          fwv.ID,
		DeviceModel: fwv.DeviceModel,
		FWVersion:   fwv.FWVersion,
		FWChecksum:  fwv.FWChecksum,
		BinaryUrl:   fwv.BinaryUrl,
		CreatedAt:   fwv.CreatedAt,
	}
}
