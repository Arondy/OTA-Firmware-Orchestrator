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

type ReportDeviceResponse struct {
	ID         uuid.UUID                   `json:"id"`
	DeviceID   uuid.UUID                   `json:"device_id"`
	CampaignID uuid.UUID                   `json:"campaign_id"`
	StageID    uuid.UUID                   `json:"stage_id"`
	Result     domain.UpdateAttemptsResult `json:"result"`
	ReportedAt time.Time                   `json:"reported_at"`
}

func ReportResponseFromDomain(r domain.UpdateAttempt) ReportDeviceResponse {
	return ReportDeviceResponse{
		ID:         r.ID,
		DeviceID:   r.DeviceID,
		CampaignID: r.CampaignID,
		StageID:    r.StageID,
		Result:     r.Result,
		ReportedAt: r.ReportedAt,
	}
}

type ReportDeviceRequest struct {
	CampaignID   uuid.UUID                   `json:"campaign_id" validate:"required"`
	StageID      uuid.UUID                   `json:"stage_id" validate:"required"`
	Result       domain.UpdateAttemptsResult `json:"result" validate:"required,update_attempt_result"`
	ErrorMessage string                      `json:"error_message,omitempty" validate:"max=1024"`
}

func (r ReportDeviceRequest) ToDomainWithDeviceID(deviceID uuid.UUID) domain.UpdateAttempt {
	return domain.UpdateAttempt{
		DeviceID:   deviceID,
		CampaignID: r.CampaignID,
		StageID:    r.StageID,
		Result:     r.Result,
	}
}
