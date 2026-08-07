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

	mux.HandleFunc("GET /api/v1/devices", device.List)
	mux.HandleFunc("POST /api/v1/devices", device.Create)
	mux.HandleFunc("POST /api/v1/devices/{id}/decommission", device.Decommission)
	mux.HandleFunc("POST /api/v1/devices/{id}/checkin", device.Checkin)
	mux.HandleFunc("POST /api/v1/devices/{id}/report", device.Report)

	mux.HandleFunc("GET /api/v1/firmware", firmware.List)
	mux.HandleFunc("POST /api/v1/firmware", firmware.Create)

	mux.HandleFunc("GET /api/v1/campaigns", campaign.List)
	mux.HandleFunc("POST /api/v1/campaigns", campaign.Create)
	mux.HandleFunc("GET /api/v1/campaigns/{id}", campaign.Get)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/start", campaign.Start)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/pause", campaign.Pause)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/resume", campaign.Resume)
}
