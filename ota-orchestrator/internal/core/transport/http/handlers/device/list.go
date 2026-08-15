package device

import (
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
)

type ListDevicesResponse struct {
	Devices []DeviceResponse `json:"devices"`
}

func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	devices, err := h.deviceSvc.List(r.Context())
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
