package core_http

import "net/http"

type HealthAPI interface {
	CheckHealth(w http.ResponseWriter, r *http.Request)
}

type DeviceAPI interface {
	ListDevices(w http.ResponseWriter, r *http.Request)
	CreateDevice(w http.ResponseWriter, r *http.Request)
	DecommissionDevice(w http.ResponseWriter, r *http.Request)
	CheckinDevice(w http.ResponseWriter, r *http.Request)
}

type FirmwareVersionAPI interface {
	ListFirmwareVersions(w http.ResponseWriter, r *http.Request)
	CreateFirmwareVersion(w http.ResponseWriter, r *http.Request)
}

type RolloutCampaignAPI interface {
	ListRolloutCampaigns(w http.ResponseWriter, r *http.Request)
	GetRolloutCampaign(w http.ResponseWriter, r *http.Request)
	CreateRolloutCampaign(w http.ResponseWriter, r *http.Request)
	StartRolloutCampaign(w http.ResponseWriter, r *http.Request)
	PauseRolloutCampaign(w http.ResponseWriter, r *http.Request)
	ResumeRolloutCampaign(w http.ResponseWriter, r *http.Request)
}
