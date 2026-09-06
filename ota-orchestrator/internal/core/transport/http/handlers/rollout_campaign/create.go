package rollout_campaign

import (
	"errors"
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
	"github.com/google/uuid"
)

// RolloutStageRequest не самостоятелен, только в составе CreateRolloutCampaignRequest
type RolloutStageRequest struct {
	OrderIndex       int     `json:"order_index" validate:"min=0"`
	TargetPercent    int     `json:"target_percent" validate:"required,min=1,max=100"`
	MinSampleSize    int     `json:"min_sample_size" validate:"required,min=1"`
	SuccessThreshold float32 `json:"success_threshold" validate:"required,gt=0,max=1"`
}

func (r RolloutStageRequest) ToDomain() domain.RolloutStage {
	return domain.RolloutStage{
		OrderIndex:       r.OrderIndex,
		TargetPercent:    r.TargetPercent,
		MinSampleSize:    r.MinSampleSize,
		SuccessThreshold: r.SuccessThreshold,
	}
}

type CreateRolloutCampaignRequest struct {
	FirmwareVersionID uuid.UUID             `json:"firmware_version_id" validate:"required"`
	RolloutStages     []RolloutStageRequest `json:"rollout_stages" validate:"required,max=20,rollout_stages,dive"`
}

func (r CreateRolloutCampaignRequest) ToDomain() domain.RolloutCampaign {
	rolloutStages := make([]domain.RolloutStage, len(r.RolloutStages))
	for i, stage := range r.RolloutStages {
		rolloutStages[i] = stage.ToDomain()
	}

	return domain.RolloutCampaign{
		FirmwareVersionID: r.FirmwareVersionID,
		RolloutStages:     rolloutStages,
	}
}

func (h *RolloutCampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())
	rolloutCampaignReq := CreateRolloutCampaignRequest{}

	if !handlers.DecodeJSONBody(w, r, logger, &rolloutCampaignReq) {
		return
	}

	if !handlers.ValidateRequest(w, logger, rolloutCampaignReq) {
		return
	}

	rolloutCampaign := rolloutCampaignReq.ToDomain()
	createdRolloutCampaign, err := h.campaignSvc.Create(r.Context(), rolloutCampaign)
	if errors.Is(err, domain.ErrRolloutStageAlreadyExists) {
		logger.Warnw("incorrect order indexes for rollout stage", "error", err)
		handlers.WriteError(w, logger, http.StatusBadRequest, domain.ErrRolloutStageAlreadyExists.Error())
		return
	} else if errors.Is(err, domain.ErrFirmwareVersionNotFound) {
		logger.Warnw("no firmware version with such id found", "error", err)
		handlers.WriteError(w, logger, http.StatusBadRequest, domain.ErrFirmwareVersionNotFound.Error())
		return
	} else if err != nil {
		logger.Errorw("failed to create rollout campaign", "error", err)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	response := RolloutCampaignFromDomain(createdRolloutCampaign)
	handlers.WriteJSON(w, logger, http.StatusCreated, response)
}
