package health

import (
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())
	handlers.WriteJSON(w, logger, http.StatusOK, map[string]string{"status": "OK"})
}
