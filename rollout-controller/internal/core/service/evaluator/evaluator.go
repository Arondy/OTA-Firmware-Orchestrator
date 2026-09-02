package evaluator

import (
	"context"
	"fmt"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CampaignRepo interface {
	GetCurrentStage(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
	ListRunningCampaigns(ctx context.Context) ([]uuid.UUID, error)
	GetCampaignStats(ctx context.Context, id uuid.UUID) (domain.CampaignStats, error)
	IncrStableCycles(ctx context.Context, id uuid.UUID) (int64, error)
	DeleteStableCycles(ctx context.Context, id uuid.UUID) error
	GetDecision(ctx context.Context, id uuid.UUID) (domain.DecisionType, error)
	SetDecision(ctx context.Context, id uuid.UUID, decision domain.DecisionType) error
	DeleteDecision(ctx context.Context, id uuid.UUID) error
}

type StageRepo interface {
	GetMinSampleSize(ctx context.Context, id uuid.UUID) (int, error)
	GetSuccessThreshold(ctx context.Context, id uuid.UUID) (float32, error)
}

type RolloutDecisionsProducer interface {
	Produce(event domain.DecisionEvent) error
}

type EvaluatorService struct {
	campaignRepo         CampaignRepo
	stageRepo            StageRepo
	decisionsProducer    RolloutDecisionsProducer
	frequency            time.Duration
	requiredStableCycles int64
	logger               *zap.SugaredLogger
}

func NewEvaluatorService(campaignRepo CampaignRepo, stageRepo StageRepo, decisionsProducer RolloutDecisionsProducer, config config.EvaluatorConfig, logger *zap.SugaredLogger) *EvaluatorService {
	return &EvaluatorService{
		campaignRepo:         campaignRepo,
		stageRepo:            stageRepo,
		decisionsProducer:    decisionsProducer,
		frequency:            config.Frequency,
		requiredStableCycles: config.RequiredStableCycles,
		logger:               logger,
	}
}

func (s *EvaluatorService) Run(ctx context.Context) error {
	ticker := time.Tick(s.frequency)

	for {
		campaigns, err := s.campaignRepo.ListRunningCampaigns(ctx)
		if err != nil {
			s.logger.Warnw("failed to list running campaigns", "error", err)
		}

		for _, campaignID := range campaigns {
			stageID, err := s.campaignRepo.GetCurrentStage(ctx, campaignID)
			if err != nil {
				s.logger.Warnw("failed to get current stage", "error", err)
				continue
			}

			decision, err := s.EvaluateDecision(ctx, campaignID)
			s.logger.Infof("campaign: %s - '%s'", campaignID, decision)

			if err == domain.ErrNotEnoughSamples {
				continue
			} else if err != nil {
				s.logger.Warnw("failed to evaluate decision", "error", err)
				continue
			}

			stableDecision, err := s.campaignRepo.GetDecision(ctx, campaignID)
			if err != nil {
				s.logger.Warnw("failed to get current decision", "error", err)
				continue
			}

			if decision != stableDecision {
				err = s.campaignRepo.SetDecision(ctx, campaignID, decision)
				if err != nil {
					s.logger.Warnw("failed to set current decision", "error", err)
					continue
				}

				err = s.campaignRepo.DeleteStableCycles(ctx, campaignID)
				if err != nil {
					s.logger.Warnw("failed to delete stable cycles", "error", err)
					continue
				}
			}

			cycles, err := s.campaignRepo.IncrStableCycles(ctx, campaignID)
			if err != nil {
				s.logger.Warnw("failed to increment stable cycles", "error", err)
				continue
			}

			if cycles >= s.requiredStableCycles {
				id, err := uuid.NewV7()
				if err != nil {
					return fmt.Errorf("failed to generate UUID: %w", err)
				}

				event := domain.DecisionEvent{
					DecisionID:      id,
					CampaignID:      campaignID,
					DecisionType:    decision,
					PreviousStageID: stageID,
					Timestamp:       time.Now(),
				}

				err = s.decisionsProducer.Produce(event)
				if err != nil {
					s.logger.Warnw("failed to produce decision", "error", err)
					continue
				}

				s.logger.Info("sent decision")

				err = s.campaignRepo.DeleteStableCycles(ctx, campaignID)
				if err != nil {
					s.logger.Warnw("failed to delete stable cycles", "error", err)
				}

				err = s.campaignRepo.DeleteDecision(ctx, campaignID)
				if err != nil {
					s.logger.Warnw("failed to delete current decision", "error", err)
				}
			}
		}

		select {
		case <-ticker:
			continue
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *EvaluatorService) EvaluateDecision(ctx context.Context, campaignID uuid.UUID) (domain.DecisionType, error) {
	stats, err := s.campaignRepo.GetCampaignStats(ctx, campaignID)
	if err != nil {
		return "", err
	}

	minSampleSize, err := s.stageRepo.GetMinSampleSize(ctx, stats.ActiveStageID)
	if err != nil {
		return "", err
	}

	if stats.SampleSize < minSampleSize {
		return "", domain.ErrNotEnoughSamples
	}

	successThreshold, err := s.stageRepo.GetSuccessThreshold(ctx, stats.ActiveStageID)
	if err != nil {
		return "", err
	}

	if stats.SuccessRate >= successThreshold {
		return domain.DecisionTypeAdvance, nil
	} else {
		return domain.DecisionTypeRollback, nil
	}
}
