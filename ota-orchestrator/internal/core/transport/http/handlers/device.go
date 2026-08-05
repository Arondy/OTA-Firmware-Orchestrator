package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/service"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/transport/http/dto"
	"github.com/google/uuid"
)

type DeviceService interface {
	ListDevices(ctx context.Context) ([]domain.Device, error)
	CreateDevice(ctx context.Context, device domain.Device) (domain.Device, error)
	DecommissionDevice(ctx context.Context, id uuid.UUID) (domain.Device, error)
	CheckinDevice(ctx context.Context, device domain.Device) (service.CheckinResult, error)
}

type DeviceHandler struct {
	svc DeviceService
}

func NewDeviceHandler(svc DeviceService) *DeviceHandler {
	return &DeviceHandler{
		svc: svc,
	}
}

func (h *DeviceHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	devices, err := h.svc.ListDevices(r.Context())
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

func (h *DeviceHandler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())
	deviceReq := dto.CreateDeviceRequest{}

	if !DecodeJSONBody(w, r, logger, &deviceReq) {
		return
	}

	if !ValidateRequest(w, logger, deviceReq) {
		return
	}

	device := deviceReq.ToDomain()
	createdDevice, err := h.svc.CreateDevice(r.Context(), device)
	if err != nil {
		logger.Errorw("failed to create device", "error", err)
		WriteInternalServerError(w, logger)
		return
	}

	response := dto.DeviceFromDomain(createdDevice)
	WriteJSON(w, logger, http.StatusCreated, response)
}

func (h *DeviceHandler) DecommissionDevice(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	device, err := h.svc.DecommissionDevice(r.Context(), id)
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

func (h *DeviceHandler) CheckinDevice(w http.ResponseWriter, r *http.Request) {
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

	checkinResult, err := h.svc.CheckinDevice(r.Context(), device)
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
