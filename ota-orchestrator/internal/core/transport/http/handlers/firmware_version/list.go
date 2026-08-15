package firmware_version

import (
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
)

type ListFirmwareVersionsResponse struct {
	FirmwareVersions []FirmwareVersionResponse `json:"firmware_versions"`
}

func (h *FirmwareVersionHandler) List(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	firmwareVersions, err := h.svc.List(r.Context())
	if err != nil {
		logger.Errorw("failed to list firmware versions", "error", err)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	response := ListFirmwareVersionsResponse{
		FirmwareVersions: make([]FirmwareVersionResponse, len(firmwareVersions)),
	}

	for i, fwVersion := range firmwareVersions {
		response.FirmwareVersions[i] = FirmwareVersionFromDomain(fwVersion)
	}

	handlers.WriteJSON(w, logger, http.StatusOK, response)
}
