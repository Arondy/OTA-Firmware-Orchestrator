package results_test

import (
	"context"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/service/results"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/service/results/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateStageResults_DelegatesToRepo(t *testing.T) {
	repo := mocks.NewMockCampaignRepo(t)
	event := domain.UpdateResultsEvent{
		EventID:    uuid.New(),
		CampaignID: uuid.New(),
		StageID:    uuid.New(),
		Result:     domain.UpdateAttemptsResultSuccess,
	}

	repo.EXPECT().UpdateStageResults(mock.Anything, event).Return(1, nil)

	count, err := results.NewCampaignStatsService(repo).UpdateStageResults(context.Background(), event)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestGetCampaignStats_DelegatesToRepo(t *testing.T) {
	repo := mocks.NewMockCampaignRepo(t)
	id := uuid.New()
	stats := domain.CampaignStats{ActiveStageID: uuid.New(), SuccessRate: 0.5, SampleSize: 2}

	repo.EXPECT().GetCampaignStats(mock.Anything, id).Return(stats, nil)

	result, err := results.NewCampaignStatsService(repo).GetCampaignStats(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, stats.SampleSize, result.SampleSize)
}
