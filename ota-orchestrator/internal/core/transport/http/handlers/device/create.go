package device

import (
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
)

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

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())
	deviceReq := CreateDeviceRequest{}

	if !handlers.DecodeJSONBody(w, r, logger, &deviceReq) {
		return
	}

	if !handlers.ValidateRequest(w, logger, deviceReq) {
		return
	}

	device := deviceReq.ToDomain()
	createdDevice, err := h.deviceSvc.Create(r.Context(), device)
	if err != nil {
		logger.Errorw("failed to create device", "error", err)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	response := DeviceFromDomain(createdDevice)
	handlers.WriteJSON(w, logger, http.StatusCreated, response)
}
