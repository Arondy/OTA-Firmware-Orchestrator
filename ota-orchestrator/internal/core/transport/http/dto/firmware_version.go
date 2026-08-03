package dto

import (
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/google/uuid"
)

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

type ListFirmwareVersionsResponse struct {
	FirmwareVersions []FirmwareVersionResponse `json:"firmware_versions"`
}

type CreateFirmwareVersionRequest struct {
	DeviceModel string `json:"device_model" validate:"required,min=2,max=64"`
	FWVersion   string `json:"fw_version" validate:"required,max=64,semver"`
	FWChecksum  string `json:"fw_checksum" validate:"required,len=64,hexadecimal"`
	BinaryUrl   string `json:"binary_url" validate:"required,url"`
}

func (r CreateFirmwareVersionRequest) ToDomain() domain.FirmwareVersion {
	return domain.FirmwareVersion{
		DeviceModel: r.DeviceModel,
		FWVersion:   r.FWVersion,
		FWChecksum:  r.FWChecksum,
		BinaryUrl:   r.BinaryUrl,
	}
}
