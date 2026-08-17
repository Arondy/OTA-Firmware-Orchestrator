package core

import (
	"context"
	"net/http"

	rollout_controller "github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/clients/rollout-controller"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/repository/kafka"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/repository/postgres"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/repository/redis"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/campaign"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/device"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/firmware"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/update"
	core_http "github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http"
	devicehandler "github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers/device"
	firmwarehandler "github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers/firmware_version"
	healthhandler "github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers/health"
	campaignhandler "github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers/rollout_campaign"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/middleware"
	"go.uber.org/zap"
)

func Run(ctx context.Context, config *config.Config, logger *zap.SugaredLogger) error {
	db, err := postgres.NewDB(ctx, config.DB, logger)
	if err != nil {
		return err
	}
	defer db.Close()

	rdb, err := redis.NewRedisClient(ctx, config.Cache, logger)
	if err != nil {
		return err
	}
	defer rdb.Close()

	checkinsProducer, err := kafka.NewCheckinsProducer(logger, config.Broker)
	if err != nil {
		return err
	}
	defer checkinsProducer.Close()

	updateResultsProducer, err := kafka.NewUpdateResultsProducer(logger, config.Broker)
	if err != nil {
		return err
	}
	defer updateResultsProducer.Close()

	rolloutControllerClient, err := rollout_controller.NewClient(config.RolloutController, logger)
	if err != nil {
		return err
	}
	defer rolloutControllerClient.Close()

	deviceRepo := postgres.NewDeviceRepo(db)
	firmwareVersionRepo := postgres.NewFirmwareVersionRepo(db)
	rolloutCampaignRepo := postgres.NewRolloutCampaignRepo(db)
	updateAttemptRepo := postgres.NewUpdateAttemptRepo(db)
	deviceCacheRepo := redis.NewDeviceCacheRepo(rdb, config.Cache)
	campaignCacheRepo := redis.NewCampaignCacheRepo(rdb)

	deviceSvc := device.NewService(deviceRepo, deviceCacheRepo)
	updateSvc := update.NewService(deviceRepo, firmwareVersionRepo, rolloutCampaignRepo, updateAttemptRepo, deviceCacheRepo, campaignCacheRepo, checkinsProducer, updateResultsProducer)
	firmwareVersionSvc := firmware.NewService(firmwareVersionRepo)
	rolloutCampaignSvc := campaign.NewService(rolloutCampaignRepo, firmwareVersionRepo, campaignCacheRepo, rolloutControllerClient)

	err = rolloutCampaignSvc.WarmUpCache(ctx, logger)
	if err != nil {
		logger.Errorw("errors during cache warmup, last one:", "error", err)
	}

	healthAPI := healthhandler.NewHealthHandler()
	deviceAPI := devicehandler.NewDeviceHandler(deviceSvc, updateSvc)
	firmwareVersionAPI := firmwarehandler.NewFirmwareVersionHandler(firmwareVersionSvc)
	rolloutCampaignAPI := campaignhandler.NewRolloutCampaignHandler(rolloutCampaignSvc)

	var router http.Handler = core_http.NewRouter(healthAPI, deviceAPI, firmwareVersionAPI, rolloutCampaignAPI)
	router = middleware.WrapInMiddleware(router, logger)
	server := core_http.NewServer(router, config.HTTPServer, logger.Named("Server"))

	if err := server.Run(ctx); err != nil {
		return err
	}
	return nil
}
