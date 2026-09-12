package firmware_version

import (
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
)

type ListFirmwareVersionsResponse struct {
	FirmwareVersions []FirmwareVersionResponse `json:"firmware_versions"`
}

type ListFirmwareVersionsParams struct {
	DeviceModel string `form:"device_model" validate:"omitempty,min=2,max=64"`
	handlers.PaginationParams
}

func (p ListFirmwareVersionsParams) ToFilters() string {
	return p.DeviceModel
}

func (h *FirmwareVersionHandler) List(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	params := ListFirmwareVersionsParams{}
	if !handlers.DecodeQueryParams(w, r, logger, &params) || !handlers.ValidateRequest(w, logger, params) {
		return
	}

	firmwareVersions, err := h.svc.List(r.Context(), params.ToFilters(), params.PaginationParams.ToDomain())
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
