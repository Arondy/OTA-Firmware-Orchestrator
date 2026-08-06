package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/transport/http/dto"
)

type FirmwareVersionService interface {
	List(ctx context.Context) ([]domain.FirmwareVersion, error)
	Create(ctx context.Context, firmwareVersion domain.FirmwareVersion) (domain.FirmwareVersion, error)
}

type FirmwareVersionHandler struct {
	svc FirmwareVersionService
}

func NewFirmwareVersionHandler(svc FirmwareVersionService) *FirmwareVersionHandler {
	return &FirmwareVersionHandler{
		svc: svc,
	}
}

func (h *FirmwareVersionHandler) List(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	firmwareVersions, err := h.svc.List(r.Context())
	if err != nil {
		logger.Errorw("failed to list firmware versions", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.ListFirmwareVersionsResponse{
		FirmwareVersions: make([]dto.FirmwareVersionResponse, len(firmwareVersions)),
	}

	for i, fwVersion := range firmwareVersions {
		response.FirmwareVersions[i] = dto.FirmwareVersionFromDomain(fwVersion)
	}

	WriteJSON(w, logger, http.StatusOK, response)
}

func (h *FirmwareVersionHandler) Create(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())
	firmwareVersionReq := dto.CreateFirmwareVersionRequest{}

	if !DecodeJSONBody(w, r, logger, &firmwareVersionReq) {
		return
	}

	if !ValidateRequest(w, logger, firmwareVersionReq) {
		return
	}

	firmwareVersion := firmwareVersionReq.ToDomain()
	createdFirmwareVersion, err := h.svc.Create(r.Context(), firmwareVersion)
	if errors.Is(err, domain.ErrFirmwareVersionAlreadyExists) {
		logger.Warnw("such firmware version already exists", "error", err)
		WriteError(w, logger, http.StatusConflict, domain.ErrFirmwareVersionAlreadyExists.Error())
		return
	} else if err != nil {
		logger.Errorw("failed to create firmware version", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.FirmwareVersionFromDomain(createdFirmwareVersion)
	WriteJSON(w, logger, http.StatusCreated, response)
}
