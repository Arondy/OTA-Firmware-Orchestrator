package firmware_version

import (
	"errors"
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
)

type CreateFirmwareVersionRequest struct {
	DeviceModel string `json:"device_model" validate:"required,min=2,max=64"`
	FWVersion   string `json:"fw_version" validate:"required,max=64,semver"`
	FWChecksum  string `json:"fw_checksum" validate:"required,len=64,hexadecimal"`
	BinaryUrl   string `json:"binary_url" validate:"required,url"`
}

func (r CreateFirmwareVersionRequest) ToDomain() domain.FirmwareVersion {
	return domain.FirmwareVersion{
		DeviceModel: r.DeviceModel,
		FWVersion:   r.FWVersion,
		FWChecksum:  r.FWChecksum,
		BinaryUrl:   r.BinaryUrl,
	}
}

func (h *FirmwareVersionHandler) Create(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())
	firmwareVersionReq := CreateFirmwareVersionRequest{}

	if !handlers.DecodeJSONBody(w, r, logger, &firmwareVersionReq) {
		return
	}

	if !handlers.ValidateRequest(w, logger, firmwareVersionReq) {
		return
	}

	firmwareVersion := firmwareVersionReq.ToDomain()
	createdFirmwareVersion, err := h.svc.Create(r.Context(), firmwareVersion)
	if errors.Is(err, domain.ErrFirmwareVersionAlreadyExists) {
		logger.Warnw("such firmware version already exists", "error", err)
		handlers.WriteError(w, logger, http.StatusConflict, domain.ErrFirmwareVersionAlreadyExists.Error())
		return
	} else if err != nil {
		logger.Errorw("failed to create firmware version", "error", err)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	response := FirmwareVersionFromDomain(createdFirmwareVersion)
	handlers.WriteJSON(w, logger, http.StatusCreated, response)
}
