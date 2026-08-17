package core_connect

import (
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/health/v1/healthv1connect"
	"github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/rollout/v1/rolloutv1connect"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/transport/connect/interceptors"
	"go.uber.org/zap"
)

func NewRouter(healthHandler healthv1connect.HealthServiceHandler, campaignStatsHandler rolloutv1connect.CampaignServiceHandler, logger *zap.SugaredLogger) *http.ServeMux {
	mux := http.NewServeMux()
	interceptors := interceptors.NewInterceptorsOption(logger)

	path, handler := healthv1connect.NewHealthServiceHandler(healthHandler, interceptors)
	mux.Handle(path, handler)

	path, handler = rolloutv1connect.NewCampaignServiceHandler(campaignStatsHandler, interceptors)
	mux.Handle(path, handler)

	return mux
}
