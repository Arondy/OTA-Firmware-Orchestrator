package rollout_campaign

import (
	"errors"
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
)

func (h *RolloutCampaignHandler) Resume(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := handlers.ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	campaign, err := h.svc.Resume(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrRolloutCampaignNotFound) {
			logger.Warnw("nonexistent id was received", "error", err)
			handlers.WriteError(w, logger, http.StatusNotFound, domain.ErrRolloutCampaignNotFound.Error())
			return
		} else if errors.Is(err, domain.ErrCampaignAlreadyRunning) {
			logger.Warnw("another rollout campaign is already running", "error", err)
			handlers.WriteError(w, logger, http.StatusConflict, domain.ErrCampaignAlreadyRunning.Error())
			return
		} else if errors.Is(err, domain.ErrRolloutCampaignWrongStatus) {
			logger.Warnw("wrong status", "error", err)
			handlers.WriteError(w, logger, http.StatusBadRequest, "can't resume non-paused rollout campaign")
			return
		}
		logger.Errorw("failed to resume rollout campaign", "error", err)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	response := RolloutCampaignFromDomain(campaign)
	handlers.WriteJSON(w, logger, http.StatusOK, response)
}
