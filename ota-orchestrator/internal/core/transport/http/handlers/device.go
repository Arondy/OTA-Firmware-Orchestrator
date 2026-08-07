package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/service/device"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/transport/http/dto"
	"github.com/google/uuid"
)

type DeviceService interface {
	List(ctx context.Context) ([]domain.Device, error)
	Create(ctx context.Context, device domain.Device) (domain.Device, error)
	Decommission(ctx context.Context, id uuid.UUID) (domain.Device, error)
	Checkin(ctx context.Context, device domain.Device) (device.CheckinResult, error)
	Report(ctx context.Context, updateAttempt domain.UpdateAttempt) (domain.UpdateAttempt, error)
}

type DeviceHandler struct {
	svc DeviceService
}

func NewDeviceHandler(svc DeviceService) *DeviceHandler {
	return &DeviceHandler{
		svc: svc,
	}
}

func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	devices, err := h.svc.List(r.Context())
	if err != nil {
		logger.Errorw("failed to list devices", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.ListDevicesResponse{
		Devices: make([]dto.DeviceResponse, len(devices)),
	}

	for i, device := range devices {
		response.Devices[i] = dto.DeviceFromDomain(device)
	}

	WriteJSON(w, logger, http.StatusOK, response)
}

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())
	deviceReq := dto.CreateDeviceRequest{}

	if !DecodeJSONBody(w, r, logger, &deviceReq) {
		return
	}

	if !ValidateRequest(w, logger, deviceReq) {
		return
	}

	device := deviceReq.ToDomain()
	createdDevice, err := h.svc.Create(r.Context(), device)
	if err != nil {
		logger.Errorw("failed to create device", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.DeviceFromDomain(createdDevice)
	WriteJSON(w, logger, http.StatusCreated, response)
}

func (h *DeviceHandler) Decommission(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	device, err := h.svc.Decommission(r.Context(), id)
	if errors.Is(err, domain.ErrDeviceNotFound) {
		logger.Warnw("nonexistent id was received", "error", err)
		WriteError(w, logger, http.StatusNotFound, domain.ErrDeviceNotFound.Error())
		return
	} else if err != nil {
		logger.Errorw("failed to decommission device", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.DeviceFromDomain(device)
	WriteJSON(w, logger, http.StatusOK, response)
}

func (h *DeviceHandler) Checkin(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	checkinReq := dto.CheckinDeviceRequest{}

	if !DecodeJSONBody(w, r, logger, &checkinReq) {
		return
	}

	if !ValidateRequest(w, logger, checkinReq) {
		return
	}

	device := checkinReq.ToDomainWithID(id)

	checkinResult, err := h.svc.Checkin(r.Context(), device)
	if errors.Is(err, domain.ErrDeviceNotFound) {
		logger.Warnw("nonexistent id was received", "error", err)
		WriteError(w, logger, http.StatusNotFound, domain.ErrDeviceNotFound.Error())
		return
	} else if err != nil {
		logger.Errorw("failed to checkin device", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.CheckinResponseFromDomain(checkinResult)
	WriteJSON(w, logger, http.StatusOK, response)
}

func (h *DeviceHandler) Report(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	reportReq := dto.ReportDeviceRequest{}

	if !DecodeJSONBody(w, r, logger, &reportReq) {
		return
	}

	if !ValidateRequest(w, logger, reportReq) {
		return
	}

	if reportReq.Result != domain.UpdateAttemptsResultSuccess && reportReq.ErrorMessage != "" {
		logger.Infow("update wasn't successful", "result", reportReq.Result, "error_message", reportReq.ErrorMessage)
	}

	updateAttempt := reportReq.ToDomainWithDeviceID(id)

	updateAttemptResult, err := h.svc.Report(r.Context(), updateAttempt)
	if errors.Is(err, domain.ErrDeviceNotFound) {
		logger.Warnw("nonexistent id was received", "error", err)
		WriteError(w, logger, http.StatusNotFound, domain.ErrDeviceNotFound.Error())
		return
	} else if errors.Is(err, domain.ErrRolloutCampaignNotFound) {
		logger.Warnw("nonexistent rollout campaign id was received", "error", err, "rollout_campaign_id", updateAttempt.CampaignID)
		WriteError(w, logger, http.StatusBadRequest, domain.ErrRolloutCampaignNotFound.Error())
		return
	} else if errors.Is(err, domain.ErrRolloutStageNotFoundInCampaign) {
		logger.Warnw("no stage with such id in the rollout campaign", "error", err, "rollout_campaign_id", updateAttempt.CampaignID, "rollout_stage_id", updateAttempt.StageID)
		WriteError(w, logger, http.StatusBadRequest, domain.ErrRolloutStageNotFoundInCampaign.Error())
		return
	} else if errors.Is(err, domain.ErrWrongDeviceModel) {
		logger.Warnw("device's model mismatch with the rollout campaign's one", "error", err, "rollout_campaign_id", updateAttempt.CampaignID, "device_id", updateAttempt.DeviceID)
		WriteError(w, logger, http.StatusBadRequest, domain.ErrWrongDeviceModel.Error())
		return
	} else if err != nil {
		logger.Errorw("failed to report device", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.ReportResponseFromDomain(updateAttemptResult)
	WriteJSON(w, logger, http.StatusOK, response)
}
