package device

import (
	"errors"
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
)

func (h *DeviceHandler) Decommission(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := handlers.ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	device, err := h.deviceSvc.Decommission(r.Context(), id)
	if errors.Is(err, domain.ErrDeviceNotFound) {
		logger.Warnw("nonexistent id was received", "error", err)
		handlers.WriteError(w, logger, http.StatusNotFound, domain.ErrDeviceNotFound.Error())
		return
	} else if err != nil {
		logger.Errorw("failed to decommission device", "error", err)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	response := DeviceFromDomain(device)
	handlers.WriteJSON(w, logger, http.StatusOK, response)
}
