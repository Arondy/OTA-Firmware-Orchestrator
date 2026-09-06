package rollout_campaign

import (
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
)

type ListRolloutCampaignsResponse struct {
	RolloutCampaigns []RolloutCampaignListItemResponse `json:"rollout_campaigns"`
}

func (h *RolloutCampaignHandler) List(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	rolloutCampaigns, err := h.campaignSvc.List(r.Context())
	if err != nil {
		logger.Errorw("failed to list rollout campaigns", "error", err)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	response := ListRolloutCampaignsResponse{
		RolloutCampaigns: make([]RolloutCampaignListItemResponse, len(rolloutCampaigns)),
	}

	for i, rc := range rolloutCampaigns {
		response.RolloutCampaigns[i] = RolloutCampaignListItemFromDomain(rc)
	}

	handlers.WriteJSON(w, logger, http.StatusOK, response)
}
