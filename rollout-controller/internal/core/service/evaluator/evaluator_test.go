package evaluator

import (
	"context"
	"errors"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---- mocks ----

type mockCampaignRepo struct{ mock.Mock }

func (m *mockCampaignRepo) GetCurrentStage(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(uuid.UUID), args.Error(1)
}
func (m *mockCampaignRepo) ListRunningCampaigns(ctx context.Context) ([]uuid.UUID, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}
func (m *mockCampaignRepo) GetCampaignStats(ctx context.Context, id uuid.UUID) (domain.CampaignStats, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.CampaignStats), args.Error(1)
}
func (m *mockCampaignRepo) IncrStableCycles(ctx context.Context, id uuid.UUID) (int64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockCampaignRepo) DeleteStableCycles(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockCampaignRepo) GetDecision(ctx context.Context, id uuid.UUID) (domain.DecisionType, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.DecisionType), args.Error(1)
}
func (m *mockCampaignRepo) SetDecision(ctx context.Context, id uuid.UUID, decision domain.DecisionType) error {
	args := m.Called(ctx, id, decision)
	return args.Error(0)
}
func (m *mockCampaignRepo) DeleteDecision(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockStageRepo struct{ mock.Mock }

func (m *mockStageRepo) GetStageStats(ctx context.Context, id uuid.UUID) (domain.StageStats, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.StageStats), args.Error(1)
}

type mockProducer struct{ mock.Mock }

func (m *mockProducer) Produce(event domain.DecisionEvent) error {
	args := m.Called(event)
	return args.Error(0)
}

func newTestService(campaignRepo CampaignRepo, stageRepo StageRepo, producer RolloutDecisionsProducer) *EvaluatorService {
	return NewEvaluatorService(
		campaignRepo,
		stageRepo,
		producer,
		config.EvaluatorConfig{Frequency: 1000000000, RequiredStableCycles: 3},
		zap.NewNop().Sugar(),
	)
}

func TestEvaluateDecision_Table(t *testing.T) {
	t.Parallel()

	campaignID := uuid.New()
	stageID := uuid.New()

	tests := []struct {
		name         string
		stats        domain.CampaignStats
		statsErr     error
		stageStats   domain.StageStats
		stageErr     error
		wantDecision domain.DecisionType
		wantErr      error
	}{
		{
			name:       "not_enough_samples_sample_lt_min",
			stats:      domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 1.0, SampleSize: 2},
			stageStats: domain.StageStats{MinSampleSize: 5, SuccessThreshold: 0.5},
			wantErr:    domain.ErrNotEnoughSamples,
		},
		{
			name:       "not_enough_samples_zero_sample",
			stats:      domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 0, SampleSize: 0},
			stageStats: domain.StageStats{MinSampleSize: 1, SuccessThreshold: 0.5},
			wantErr:    domain.ErrNotEnoughSamples,
		},
		{
			name:         "advance_when_success_rate_equals_threshold",
			stats:        domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 0.5, SampleSize: 10},
			stageStats:   domain.StageStats{MinSampleSize: 10, SuccessThreshold: 0.5},
			wantDecision: domain.DecisionTypeAdvance,
		},
		{
			name:         "advance_when_success_rate_above_threshold",
			stats:        domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 0.8, SampleSize: 100},
			stageStats:   domain.StageStats{MinSampleSize: 10, SuccessThreshold: 0.5},
			wantDecision: domain.DecisionTypeAdvance,
		},
		{
			name:         "advance_when_success_rate_1_threshold_1",
			stats:        domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 1.0, SampleSize: 50},
			stageStats:   domain.StageStats{MinSampleSize: 50, SuccessThreshold: 1.0},
			wantDecision: domain.DecisionTypeAdvance,
		},
		{
			name:         "advance_when_threshold_0_always_advance",
			stats:        domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 0, SampleSize: 5},
			stageStats:   domain.StageStats{MinSampleSize: 5, SuccessThreshold: 0},
			wantDecision: domain.DecisionTypeAdvance,
		},
		{
			name:         "rollback_when_success_rate_below_threshold",
			stats:        domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 0.49, SampleSize: 10},
			stageStats:   domain.StageStats{MinSampleSize: 5, SuccessThreshold: 0.5},
			wantDecision: domain.DecisionTypeRollback,
		},
		{
			name:         "rollback_when_success_rate_0_threshold_high",
			stats:        domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 0, SampleSize: 10},
			stageStats:   domain.StageStats{MinSampleSize: 5, SuccessThreshold: 0.9},
			wantDecision: domain.DecisionTypeRollback,
		},
		{
			name:         "rollback_when_just_below_threshold",
			stats:        domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 0.799, SampleSize: 20},
			stageStats:   domain.StageStats{MinSampleSize: 10, SuccessThreshold: 0.8},
			wantDecision: domain.DecisionTypeRollback,
		},
		{
			name:         "sample_exactly_min_advance",
			stats:        domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 0.6, SampleSize: 10},
			stageStats:   domain.StageStats{MinSampleSize: 10, SuccessThreshold: 0.5},
			wantDecision: domain.DecisionTypeAdvance,
		},
		{
			name:         "sample_exactly_min_rollback",
			stats:        domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 0.4, SampleSize: 10},
			stageStats:   domain.StageStats{MinSampleSize: 10, SuccessThreshold: 0.5},
			wantDecision: domain.DecisionTypeRollback,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			campaignRepo := new(mockCampaignRepo)
			stageRepo := new(mockStageRepo)
			producer := new(mockProducer)

			campaignRepo.On("GetCampaignStats", mock.Anything, campaignID).Return(tc.stats, tc.statsErr)
			if tc.statsErr == nil {
				stageRepo.On("GetStageStats", mock.Anything, stageID).Return(tc.stageStats, tc.stageErr)
			}

			svc := newTestService(campaignRepo, stageRepo, producer)
			decision, err := svc.EvaluateDecision(context.Background(), campaignID)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Empty(t, decision)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantDecision, decision)
			}
		})
	}
}

func TestEvaluateDecision_Errors(t *testing.T) {
	t.Parallel()
	campaignID := uuid.New()
	stageID := uuid.New()

	t.Run("GetCampaignStats_error_propagated", func(t *testing.T) {
		t.Parallel()
		campaignRepo := new(mockCampaignRepo)
		stageRepo := new(mockStageRepo)
		producer := new(mockProducer)

		wantErr := errors.New("redis unavailable")
		campaignRepo.On("GetCampaignStats", mock.Anything, campaignID).Return(domain.CampaignStats{}, wantErr)

		svc := newTestService(campaignRepo, stageRepo, producer)
		_, err := svc.EvaluateDecision(context.Background(), campaignID)
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("GetStageStats_error_propagated", func(t *testing.T) {
		t.Parallel()
		campaignRepo := new(mockCampaignRepo)
		stageRepo := new(mockStageRepo)
		producer := new(mockProducer)

		stats := domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 0.9, SampleSize: 10}
		wantErr := errors.New("stage not found")
		campaignRepo.On("GetCampaignStats", mock.Anything, campaignID).Return(stats, nil)
		stageRepo.On("GetStageStats", mock.Anything, stageID).Return(domain.StageStats{}, wantErr)

		svc := newTestService(campaignRepo, stageRepo, producer)
		_, err := svc.EvaluateDecision(context.Background(), campaignID)
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("not_enough_samples_returns_error", func(t *testing.T) {
		t.Parallel()
		campaignRepo := new(mockCampaignRepo)
		stageRepo := new(mockStageRepo)
		producer := new(mockProducer)

		stats := domain.CampaignStats{ActiveStageID: stageID, SuccessRate: 0.9, SampleSize: 2}
		campaignRepo.On("GetCampaignStats", mock.Anything, campaignID).Return(stats, nil)
		stageRepo.On("GetStageStats", mock.Anything, stageID).Return(domain.StageStats{MinSampleSize: 10, SuccessThreshold: 0.5}, nil)

		svc := newTestService(campaignRepo, stageRepo, producer)
		_, err := svc.EvaluateDecision(context.Background(), campaignID)
		require.ErrorIs(t, err, domain.ErrNotEnoughSamples)
	})
}
