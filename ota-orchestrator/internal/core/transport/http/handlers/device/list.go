package device

import (
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
)

type ListDevicesResponse struct {
	Devices []DeviceResponse `json:"devices"`
}

type ListDevicesParams struct {
	DeviceModel string              `form:"device_model" validate:"omitempty,min=2,max=64"`
	Status      domain.DeviceStatus `form:"status" validate:"omitempty,device_status"`
	handlers.PaginationParams
}

func (p ListDevicesParams) ToFilters() domain.DeviceFilters {
	return domain.DeviceFilters{
		DeviceModel: p.DeviceModel,
		Status:      p.Status,
	}
}

func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	params := ListDevicesParams{}
	if !handlers.DecodeQueryParams(w, r, logger, &params) || !handlers.ValidateRequest(w, logger, params) {
		return
	}

	devices, err := h.deviceSvc.List(r.Context(), params.ToFilters(), params.PaginationParams.ToDomain())
	if err != nil {
		logger.Errorw("failed to list devices", "error", err)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	response := ListDevicesResponse{
		Devices: make([]DeviceResponse, len(devices)),
	}

	for i, device := range devices {
		response.Devices[i] = DeviceFromDomain(device)
	}

	handlers.WriteJSON(w, logger, http.StatusOK, response)
}
