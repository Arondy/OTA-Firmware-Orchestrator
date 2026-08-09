package core

import (
	"context"
	"net/http"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/repository/postgres"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/repository/redis"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/service/campaign"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/service/device"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/service/firmware"
	core_http "github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/transport/http"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/transport/http/handlers"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/transport/http/middleware"
	"go.uber.org/zap"
)

func Run(ctx context.Context, config *config.Config, logger *zap.SugaredLogger) error {
	db, err := postgres.NewDB(ctx, config.DB)
	if err != nil {
		return err
	}
	defer db.Close()

	rdb, err := redis.NewRedisClient(ctx, config.Cache)
	if err != nil {
		return err
	}
	defer rdb.Close()

	deviceRepo := postgres.NewDeviceRepo(db)
	firmwareVersionRepo := postgres.NewFirmwareVersionRepo(db)
	rolloutCampaignRepo := postgres.NewRolloutCampaignRepo(db)
	updateAttemptRepo := postgres.NewUpdateAttemptRepo(db)
	deviceCacheRepo := redis.NewDeviceCacheRepo(rdb, config.Cache)
	campaignCacheRepo := redis.NewCampaignCacheRepo(rdb)

	deviceSvc := device.NewService(deviceRepo, firmwareVersionRepo, rolloutCampaignRepo, updateAttemptRepo, deviceCacheRepo, campaignCacheRepo)
	firmwareVersionSvc := firmware.NewService(firmwareVersionRepo)
	rolloutCampaignSvc := campaign.NewService(rolloutCampaignRepo, firmwareVersionRepo, campaignCacheRepo)

	err = rolloutCampaignSvc.WarmUpCache(ctx, logger)
	if err != nil {
		logger.Errorw("errors during cache warmup, last one:", "error", err)
	}

	healthAPI := handlers.NewHealthHandler()
	deviceAPI := handlers.NewDeviceHandler(deviceSvc)
	firmwareVersionAPI := handlers.NewFirmwareVersionHandler(firmwareVersionSvc)
	rolloutCampaignAPI := handlers.NewRolloutCampaignHandler(rolloutCampaignSvc)

	var router http.Handler = core_http.NewRouter(healthAPI, deviceAPI, firmwareVersionAPI, rolloutCampaignAPI)
	router = middleware.WrapInMiddleware(router, logger)
	server := core_http.NewServer(router, &config.HTTPServer, logger.Named("Server"))

	if err := server.Run(ctx); err != nil {
		return err
	}
	return nil
}
