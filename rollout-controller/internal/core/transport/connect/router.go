package core_connect

import (
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/rollout/v1/rolloutv1connect"
)

func NewRouter(campaignStatsHandler rolloutv1connect.CampaignServiceHandler) *http.ServeMux {
	mux := http.NewServeMux()

	path, handler := rolloutv1connect.NewCampaignServiceHandler(campaignStatsHandler)
	mux.Handle(path, handler)

	return mux
}
