package device

import (
	"errors"
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
	"github.com/google/uuid"
)

type CheckinDeviceResponse struct {
	UpdateAvailable bool       `json:"update_available"`
	StageID         *uuid.UUID `json:"stage_id,omitempty"`
	BinaryUrl       string     `json:"binary_url,omitempty"`
	FWChecksum      string     `json:"fw_checksum,omitempty"`
}

func CheckinResponseFromDomain(c domain.CheckinResult) CheckinDeviceResponse {
	return CheckinDeviceResponse{
		UpdateAvailable: c.UpdateAvailable,
		StageID:         c.StageID,
		BinaryUrl:       c.BinaryUrl,
		FWChecksum:      c.FWChecksum,
	}
}

type CheckinDeviceRequest struct {
	CurrentVersion string `json:"current_version" validate:"required,max=64,semver"`
}

func (r CheckinDeviceRequest) ToDomainWithID(id uuid.UUID) domain.Device {
	return domain.Device{
		ID:             id,
		CurrentVersion: r.CurrentVersion,
	}
}

func (h *DeviceHandler) Checkin(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	id, ok := handlers.ParseUUIDFromPath(w, r, logger)
	if !ok {
		return
	}

	logger = logger.With("id", id)

	checkinReq := CheckinDeviceRequest{}

	if !handlers.DecodeJSONBody(w, r, logger, &checkinReq) {
		return
	}

	if !handlers.ValidateRequest(w, logger, checkinReq) {
		return
	}

	device := checkinReq.ToDomainWithID(id)

	checkinResult, err := h.updateSvc.Checkin(r.Context(), device)
	if errors.Is(err, domain.ErrDeviceNotFound) {
		logger.Warnw("nonexistent id was received", "error", err)
		handlers.WriteError(w, logger, http.StatusNotFound, domain.ErrDeviceNotFound.Error())
		return
	} else if err != nil {
		logger.Errorw("failed to checkin device", "error", err)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	response := CheckinResponseFromDomain(checkinResult)
	handlers.WriteJSON(w, logger, http.StatusOK, response)
}
