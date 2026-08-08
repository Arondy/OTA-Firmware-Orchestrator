package core_http

import "net/http"

type HealthAPI interface {
	CheckHealth(w http.ResponseWriter, r *http.Request)
}

type DeviceAPI interface {
	List(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	Decommission(w http.ResponseWriter, r *http.Request)
	Checkin(w http.ResponseWriter, r *http.Request)
	Report(w http.ResponseWriter, r *http.Request)
}

type FirmwareVersionAPI interface {
	List(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
}

type RolloutCampaignAPI interface {
	List(w http.ResponseWriter, r *http.Request)
	Get(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	Start(w http.ResponseWriter, r *http.Request)
	Pause(w http.ResponseWriter, r *http.Request)
	Resume(w http.ResponseWriter, r *http.Request)
	AdvanceStage(w http.ResponseWriter, r *http.Request)
}
