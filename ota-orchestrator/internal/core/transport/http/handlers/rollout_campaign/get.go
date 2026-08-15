package rollout_campaign

import (
	"errors"
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
)

func (h *RolloutCampaignHandler) Get(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := handlers.ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	rolloutCampaign, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrRolloutCampaignNotFound) {
			logger.Warnw("nonexistent id was received", "error", err)
			handlers.WriteError(w, logger, http.StatusNotFound, domain.ErrRolloutCampaignNotFound.Error())
			return
		}
		logger.Errorw("failed to get rollout campaign", "error", err)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	response := RolloutCampaignFromDomain(rolloutCampaign)
	handlers.WriteJSON(w, logger, http.StatusOK, response)
}
