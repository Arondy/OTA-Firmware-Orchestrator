package dto

import (
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/service/device"
	"github.com/google/uuid"
)

type DeviceResponse struct {
	ID             uuid.UUID           `json:"id"`
	DeviceModel    string              `json:"device_model"`
	CurrentVersion string              `json:"current_version"`
	Status         domain.DeviceStatus `json:"status"`
	LastSeen       *time.Time          `json:"last_seen,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
}

func DeviceFromDomain(d domain.Device) DeviceResponse {
	return DeviceResponse{
		ID:             d.ID,
		DeviceModel:    d.DeviceModel,
		CurrentVersion: d.CurrentVersion,
		Status:         d.Status,
		LastSeen:       d.LastSeen,
		CreatedAt:      d.CreatedAt,
	}
}

type CreateDeviceRequest struct {
	DeviceModel    string `json:"device_model" validate:"required,min=2,max=64"`
	CurrentVersion string `json:"current_version" validate:"required,max=64,semver"`
}

func (r CreateDeviceRequest) ToDomain() domain.Device {
	return domain.Device{
		DeviceModel:    r.DeviceModel,
		CurrentVersion: r.CurrentVersion,
	}
}

type ListDevicesResponse struct {
	Devices []DeviceResponse `json:"devices"`
}

type CheckinDeviceResponse struct {
	UpdateAvailable bool       `json:"update_available"`
	StageID         *uuid.UUID `json:"stage_id,omitempty"`
	BinaryUrl       string     `json:"binary_url,omitempty"`
	FWChecksum      string     `json:"fw_checksum,omitempty"`
}

func CheckinResponseFromDomain(c device.CheckinResult) CheckinDeviceResponse {
	return CheckinDeviceResponse{
		UpdateAvailable: c.UpdateAvailable,
		StageID:         c.StageID,
		BinaryUrl:       c.BinaryUrl,
		FWChecksum:      c.FWChecksum,
	}
}

type CheckinDeviceRequest struct {
	CurrentVersion string `json:"current_version" validate:"required,max=64,semver"`
}

func (r CheckinDeviceRequest) ToDomainWithID(id uuid.UUID) domain.Device {
	return domain.Device{
		ID:             id,
		CurrentVersion: r.CurrentVersion,
	}
}
