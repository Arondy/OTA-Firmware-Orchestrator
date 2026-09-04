package evaluator

import (
	"context"
	"errors"
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
	GetStageStats(ctx context.Context, id uuid.UUID) (domain.StageStats, error)
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
			decision, err := s.evaluateDecision(ctx, campaignID)
			s.logger.Infof("campaign: %s - '%s'", campaignID, decision)

			if err != nil {
				if !errors.Is(err, domain.ErrNotEnoughSamples) {
					s.logger.Warnw("failed to evaluate decision", "error", err)
				}
				continue
			}

			err = s.updateDecision(ctx, campaignID, decision)
			if err != nil {
				s.logger.Warn(err)
				continue
			}

			err = s.sendDecisionIfStable(ctx, campaignID, decision)
			if err != nil {
				s.logger.Warn(err)
				continue
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

func (s *EvaluatorService) updateDecision(ctx context.Context, campaignID uuid.UUID, decision domain.DecisionType) error {
	stableDecision, err := s.campaignRepo.GetDecision(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("failed to get current decision: %w", err)
	}

	if decision != stableDecision {
		err = s.campaignRepo.SetDecision(ctx, campaignID, decision)
		if err != nil {
			return fmt.Errorf("failed to set current decision: %w", err)
		}

		err = s.campaignRepo.DeleteStableCycles(ctx, campaignID)
		if err != nil {
			return fmt.Errorf("failed to delete stable cycles: %w", err)
		}
	}

	return nil
}

func (s *EvaluatorService) sendDecisionIfStable(ctx context.Context, campaignID uuid.UUID, decision domain.DecisionType) error {
	cycles, err := s.campaignRepo.IncrStableCycles(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("failed to increment stable cycles: %w", err)
	}

	if cycles < s.requiredStableCycles {
		return nil
	}

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}

	stageID, err := s.campaignRepo.GetCurrentStage(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("failed to get current stage: %w", err)
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
		return fmt.Errorf("failed to produce decision: %w", err)
	}

	s.logger.Infow(fmt.Sprintf("sent decision %s", decision), "campaign_id", campaignID, "stage_id", stageID)

	err = s.campaignRepo.DeleteStableCycles(ctx, campaignID)
	if err != nil {
		s.logger.Warnw("failed to delete stable cycles", "error", err)
	}

	err = s.campaignRepo.DeleteDecision(ctx, campaignID)
	if err != nil {
		s.logger.Warnw("failed to delete current decision", "error", err)
	}

	return nil
}

func (s *EvaluatorService) evaluateDecision(ctx context.Context, campaignID uuid.UUID) (domain.DecisionType, error) {
	currentStats, err := s.campaignRepo.GetCampaignStats(ctx, campaignID)
	if err != nil {
		return "", err
	}

	stageStats, err := s.stageRepo.GetStageStats(ctx, currentStats.ActiveStageID)
	if err != nil {
		return "", err
	}

	if currentStats.SampleSize < stageStats.MinSampleSize {
		return "", domain.ErrNotEnoughSamples
	}

	if currentStats.SuccessRate >= stageStats.SuccessThreshold {
		return domain.DecisionTypeAdvance, nil
	} else {
		return domain.DecisionTypeRollback, nil
	}
}
