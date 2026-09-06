package core

import (
	"context"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/repository/kafka"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/repository/redis"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/service/evaluator"
	update_results "github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/service/results"
	core_connect "github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/transport/connect"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/transport/connect/handlers"
	core_kafka "github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/transport/kafka"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, cfg *config.Config, logger *zap.SugaredLogger) error {
	rdb, err := redis.NewRedisClient(ctx, cfg.Cache, logger)
	if err != nil {
		return err
	}
	defer rdb.Close()

	rolloutDecisionsBrokerConfig := cfg.Broker
	rolloutDecisionsBrokerConfig.Topic = config.RolloutDecisionTopic
	rolloutDecisionsProducer, err := kafka.NewRolloutDecisionsProducer(rolloutDecisionsBrokerConfig, logger)
	if err != nil {
		return err
	}
	defer rolloutDecisionsProducer.Close()

	campaignCacheRepo := redis.NewCampaignCacheRepo(rdb, cfg.Cache)
	stageCacheRepo := redis.NewStageCacheRepo(rdb)

	campaignStatsSvc := update_results.NewCampaignStatsService(campaignCacheRepo)
	evaluatorSvc := evaluator.NewEvaluatorService(campaignCacheRepo, stageCacheRepo, rolloutDecisionsProducer, cfg.Evaluator, logger)

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return evaluatorSvc.Run(egCtx)
	})

	updateResultsBrokerConfig := cfg.Broker
	updateResultsBrokerConfig.Topic = config.UpdateResultsTopic
	updateResultsConsumer, err := core_kafka.NewUpdateResultsConsumer(campaignStatsSvc, updateResultsBrokerConfig, logger)
	if err != nil {
		return err
	}
	defer updateResultsConsumer.Close()

	eg.Go(func() error {
		return updateResultsConsumer.Run(egCtx)
	})

	healthHandler := handlers.NewHealthHandler()
	campaignHandler := handlers.NewCampaignHandler(campaignStatsSvc, evaluatorSvc)

	router := core_connect.NewRouter(healthHandler, campaignHandler, logger)
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
