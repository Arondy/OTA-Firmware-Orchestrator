package core

import (
	"context"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/repository/redis"
	update_results "github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/service/results"
	core_connect "github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/transport/connect"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/transport/connect/handlers"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/transport/kafka"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, cfg *config.Config, logger *zap.SugaredLogger) error {
	cfg.Broker.Topic = config.UpdateResultsTopic

	rdb, err := redis.NewRedisClient(ctx, cfg.Cache, logger)
	if err != nil {
		return err
	}
	defer rdb.Close()

	campaignCacheRepo := redis.NewCampaignCacheRepo(rdb, cfg.Cache)

	campaignStatsSvc := update_results.NewCampaignStatsService(campaignCacheRepo)

	updateResultsConsumer, err := kafka.NewUpdateResultsConsumer(campaignStatsSvc, cfg.Broker, logger)
	if err != nil {
		return err
	}
	defer updateResultsConsumer.Close()

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return updateResultsConsumer.Run(egCtx)
	})

	healthHandler := handlers.NewHealthHandler()
	campaignStatsHandler := handlers.NewCampaignStatsHandler(campaignStatsSvc)

	router := core_connect.NewRouter(healthHandler, campaignStatsHandler, logger)
	server := core_connect.NewServer(router, cfg.Server, logger.Named("Server"))

	eg.Go(func() error {
		return server.Run(egCtx)
	})

	<-egCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- eg.Wait() }()

	select {
	case err := <-done:
		return err
	case <-shutdownCtx.Done():
		logger.Warn("shutdown timeout, forcing exit")
		return shutdownCtx.Err()
	}
}
