package device

import (
	"errors"
	"net/http"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
	"github.com/google/uuid"
)

type ReportDeviceResponse struct {
	ID         uuid.UUID                   `json:"id"`
	DeviceID   uuid.UUID                   `json:"device_id"`
	CampaignID uuid.UUID                   `json:"campaign_id"`
	StageID    uuid.UUID                   `json:"stage_id"`
	Result     domain.UpdateAttemptsResult `json:"result"`
	ReportedAt time.Time                   `json:"reported_at"`
}

func ReportResponseFromDomain(r domain.UpdateAttempt) ReportDeviceResponse {
	return ReportDeviceResponse{
		ID:         r.ID,
		DeviceID:   r.DeviceID,
		CampaignID: r.CampaignID,
		StageID:    r.StageID,
		Result:     r.Result,
		ReportedAt: r.ReportedAt,
	}
}

type ReportDeviceRequest struct {
	CampaignID   uuid.UUID                   `json:"campaign_id" validate:"required"`
	StageID      uuid.UUID                   `json:"stage_id" validate:"required"`
	Result       domain.UpdateAttemptsResult `json:"result" validate:"required,update_attempt_result"`
	ErrorMessage string                      `json:"error_message,omitempty" validate:"max=1024"`
}

func (r ReportDeviceRequest) ToDomainWithDeviceID(deviceID uuid.UUID) domain.UpdateAttempt {
	return domain.UpdateAttempt{
		DeviceID:   deviceID,
		CampaignID: r.CampaignID,
		StageID:    r.StageID,
		Result:     r.Result,
	}
}

func (h *DeviceHandler) Report(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := handlers.ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	reportReq := ReportDeviceRequest{}

	if !handlers.DecodeJSONBody(w, r, logger, &reportReq) {
		return
	}

	if !handlers.ValidateRequest(w, logger, reportReq) {
		return
	}

	if reportReq.Result != domain.UpdateAttemptsResultSuccess && reportReq.ErrorMessage != "" {
		logger.Infow("update wasn't successful", "result", reportReq.Result, "error_message", reportReq.ErrorMessage)
	}

	updateAttempt := reportReq.ToDomainWithDeviceID(id)

	updateAttemptResult, err := h.updateSvc.Report(r.Context(), updateAttempt)
	if errors.Is(err, domain.ErrDeviceNotFound) {
		logger.Warnw("nonexistent id was received", "error", err)
		handlers.WriteError(w, logger, http.StatusNotFound, domain.ErrDeviceNotFound.Error())
		return
	} else if errors.Is(err, domain.ErrRolloutCampaignNotFound) {
		logger.Warnw("nonexistent rollout campaign id was received", "error", err, "rollout_campaign_id", updateAttempt.CampaignID)
		handlers.WriteError(w, logger, http.StatusBadRequest, domain.ErrRolloutCampaignNotFound.Error())
		return
	} else if errors.Is(err, domain.ErrRolloutStageNotFoundInCampaign) {
		logger.Warnw("no stage with such id in the rollout campaign", "error", err, "rollout_campaign_id", updateAttempt.CampaignID, "rollout_stage_id", updateAttempt.StageID)
		handlers.WriteError(w, logger, http.StatusBadRequest, domain.ErrRolloutStageNotFoundInCampaign.Error())
		return
	} else if errors.Is(err, domain.ErrWrongDeviceModel) {
		logger.Warnw("device's model mismatch with the rollout campaign's one", "error", err, "rollout_campaign_id", updateAttempt.CampaignID, "device_id", updateAttempt.DeviceID)
		handlers.WriteError(w, logger, http.StatusBadRequest, domain.ErrWrongDeviceModel.Error())
		return
	} else if errors.Is(err, domain.ErrUpdateResultNotProduced) {
		logger.Errorw("failed to produce update result for device", "error", err)
		handlers.WriteError(w, logger, http.StatusServiceUnavailable, "service unavailable")
		return
	} else if err != nil {
		logger.Errorw("failed to report device", "error", err)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	response := ReportResponseFromDomain(updateAttemptResult)
	handlers.WriteJSON(w, logger, http.StatusOK, response)
}
