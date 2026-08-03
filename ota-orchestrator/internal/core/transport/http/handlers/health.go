package handlers

import (
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/config"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())
	WriteJSON(w, logger, http.StatusOK, map[string]string{"status": "OK"})
}
