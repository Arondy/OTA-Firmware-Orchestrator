package device

import (
	"context"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
)

type DeviceService interface {
	List(ctx context.Context) ([]domain.Device, error)
	Create(ctx context.Context, device domain.Device) (domain.Device, error)
	Decommission(ctx context.Context, id uuid.UUID) (domain.Device, error)
}

type UpdateService interface {
	Checkin(ctx context.Context, device domain.Device) (domain.CheckinResult, error)
	Report(ctx context.Context, updateAttempt domain.UpdateAttempt) (domain.UpdateAttempt, error)
}

type DeviceHandler struct {
	deviceSvc DeviceService
	updateSvc UpdateService
}

func NewDeviceHandler(deviceSvc DeviceService, updateSvc UpdateService) *DeviceHandler {
	return &DeviceHandler{
		deviceSvc: deviceSvc,
		updateSvc: updateSvc,
	}
}

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
