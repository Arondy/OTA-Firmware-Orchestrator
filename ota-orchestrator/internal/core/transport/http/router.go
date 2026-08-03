package core_http

import (
	"net/http"
)

func NewRouter(health HealthAPI, device DeviceAPI, firmware FirmwareVersionAPI, campaign RolloutCampaignAPI) *http.ServeMux {
	mux := http.NewServeMux()
	registerRoutes(mux, health, device, firmware, campaign)
	return mux
}

func registerRoutes(mux *http.ServeMux, health HealthAPI, device DeviceAPI, firmware FirmwareVersionAPI, campaign RolloutCampaignAPI) {
	mux.HandleFunc("GET /healthz", health.CheckHealth)

	mux.HandleFunc("GET /api/v1/devices", device.ListDevices)
	mux.HandleFunc("POST /api/v1/devices", device.CreateDevice)
	mux.HandleFunc("POST /api/v1/devices/{id}/decommission", device.DecommissionDevice)

	mux.HandleFunc("GET /api/v1/firmware", firmware.ListFirmwareVersions)
	mux.HandleFunc("POST /api/v1/firmware", firmware.CreateFirmwareVersion)

	mux.HandleFunc("GET /api/v1/campaigns", campaign.ListRolloutCampaigns)
	mux.HandleFunc("POST /api/v1/campaigns", campaign.CreateRolloutCampaign)
	mux.HandleFunc("GET /api/v1/campaigns/{id}", campaign.GetRolloutCampaign)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/start", campaign.StartRolloutCampaign)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/pause", campaign.PauseRolloutCampaign)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/resume", campaign.ResumeRolloutCampaign)
}
