package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/transport/http/dto"
	"github.com/google/uuid"
)

type RolloutCampaignService interface {
	List(ctx context.Context) ([]domain.RolloutCampaign, error)
	Get(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Create(ctx context.Context, campaign domain.RolloutCampaign) (domain.RolloutCampaign, error)
	Start(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Pause(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Resume(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	AdvanceStage(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
}

type RolloutCampaignHandler struct {
	svc RolloutCampaignService
}

func NewRolloutCampaignHandler(svc RolloutCampaignService) *RolloutCampaignHandler {
	return &RolloutCampaignHandler{
		svc: svc,
	}
}

func (h *RolloutCampaignHandler) List(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	rolloutCampaigns, err := h.svc.List(r.Context())
	if err != nil {
		logger.Errorw("failed to list rollout campaigns", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.ListRolloutCampaignsResponse{
		RolloutCampaigns: make([]dto.RolloutCampaignListItemResponse, len(rolloutCampaigns)),
	}

	for i, rc := range rolloutCampaigns {
		response.RolloutCampaigns[i] = dto.RolloutCampaignListItemFromDomain(rc)
	}

	WriteJSON(w, logger, http.StatusOK, response)
}

func (h *RolloutCampaignHandler) Get(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	rolloutCampaign, err := h.svc.Get(r.Context(), id)
	if errors.Is(err, domain.ErrRolloutCampaignNotFound) {
		logger.Warnw("nonexistent id was received", "error", err)
		WriteError(w, logger, http.StatusNotFound, domain.ErrRolloutCampaignNotFound.Error())
		return
	} else if err != nil {
		logger.Errorw("failed to get rollout campaign", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.RolloutCampaignFromDomain(rolloutCampaign)
	WriteJSON(w, logger, http.StatusOK, response)
}

func (h *RolloutCampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())
	rolloutCampaignReq := dto.CreateRolloutCampaignRequest{}

	if !DecodeJSONBody(w, r, logger, &rolloutCampaignReq) {
		return
	}

	if !ValidateRequest(w, logger, rolloutCampaignReq) {
		return
	}

	rolloutCampaign := rolloutCampaignReq.ToDomain()
	createdRolloutCampaign, err := h.svc.Create(r.Context(), rolloutCampaign)
	if errors.Is(err, domain.ErrRolloutStageAlreadyExists) {
		logger.Warnw("incorrect order indexes for rollout stage", "error", err)
		WriteError(w, logger, http.StatusBadRequest, domain.ErrRolloutStageAlreadyExists.Error())
		return
	} else if errors.Is(err, domain.ErrFirmwareVersionNotFound) {
		logger.Warnw("no firmware version with such id found", "error", err)
		WriteError(w, logger, http.StatusBadRequest, domain.ErrFirmwareVersionNotFound.Error())
		return
	} else if err != nil {
		logger.Errorw("failed to create rollout campaign", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.RolloutCampaignFromDomain(createdRolloutCampaign)
	WriteJSON(w, logger, http.StatusCreated, response)
}

func (h *RolloutCampaignHandler) Start(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	campaign, err := h.svc.Start(r.Context(), id)
	if errors.Is(err, domain.ErrRolloutCampaignNotFound) {
		logger.Warnw("nonexistent id was received", "error", err)
		WriteError(w, logger, http.StatusNotFound, domain.ErrRolloutCampaignNotFound.Error())
		return
	} else if errors.Is(err, domain.ErrCampaignAlreadyRunning) {
		logger.Warnw("another rollout campaign is already running", "error", err)
		WriteError(w, logger, http.StatusConflict, domain.ErrCampaignAlreadyRunning.Error())
		return
	} else if errors.Is(err, domain.ErrRolloutCampaignWrongStatus) {
		logger.Warnw("wrong status", "error", err)
		WriteError(w, logger, http.StatusBadRequest, "can't start non-draft rollout campaign")
		return
	} else if err != nil {
		logger.Errorw("failed to start rollout campaign", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.RolloutCampaignFromDomain(campaign)
	WriteJSON(w, logger, http.StatusOK, response)
}

func (h *RolloutCampaignHandler) Pause(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	campaign, err := h.svc.Pause(r.Context(), id)
	if errors.Is(err, domain.ErrRolloutCampaignNotFound) {
		logger.Warnw("nonexistent id was received", "error", err)
		WriteError(w, logger, http.StatusNotFound, domain.ErrRolloutCampaignNotFound.Error())
		return
	} else if errors.Is(err, domain.ErrRolloutCampaignWrongStatus) {
		logger.Warnw("wrong status", "error", err)
		WriteError(w, logger, http.StatusBadRequest, "can't pause non-running rollout campaign")
		return
	} else if err != nil {
		logger.Errorw("failed to pause rollout campaign", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.RolloutCampaignFromDomain(campaign)
	WriteJSON(w, logger, http.StatusOK, response)
}

func (h *RolloutCampaignHandler) Resume(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	campaign, err := h.svc.Resume(r.Context(), id)
	if errors.Is(err, domain.ErrRolloutCampaignNotFound) {
		logger.Warnw("nonexistent id was received", "error", err)
		WriteError(w, logger, http.StatusNotFound, domain.ErrRolloutCampaignNotFound.Error())
		return
	} else if errors.Is(err, domain.ErrCampaignAlreadyRunning) {
		logger.Warnw("another rollout campaign is already running", "error", err)
		WriteError(w, logger, http.StatusConflict, domain.ErrCampaignAlreadyRunning.Error())
		return
	} else if errors.Is(err, domain.ErrRolloutCampaignWrongStatus) {
		logger.Warnw("wrong status", "error", err)
		WriteError(w, logger, http.StatusBadRequest, "can't resume non-paused rollout campaign")
		return
	} else if err != nil {
		logger.Errorw("failed to resume rollout campaign", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.RolloutCampaignFromDomain(campaign)
	WriteJSON(w, logger, http.StatusOK, response)
}

func (h *RolloutCampaignHandler) AdvanceStage(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	campaign, err := h.svc.AdvanceStage(r.Context(), id)
	if errors.Is(err, domain.ErrRolloutCampaignNotFound) {
		logger.Warnw("nonexistent id was received", "error", err)
		WriteError(w, logger, http.StatusNotFound, domain.ErrRolloutCampaignNotFound.Error())
		return
	} else if errors.Is(err, domain.ErrRolloutCampaignWrongStatus) {
		logger.Warnw("wrong status", "error", err)
		WriteError(w, logger, http.StatusBadRequest, "can't advance non-running rollout campaign")
		return
	} else if err != nil {
		logger.Errorw("failed to advance rollout campaign", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.RolloutCampaignFromDomain(campaign)
	WriteJSON(w, logger, http.StatusOK, response)
}
