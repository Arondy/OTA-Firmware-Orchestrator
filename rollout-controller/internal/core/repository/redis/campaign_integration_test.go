//go:build integration

package redis

import (
	"context"
	"testing"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCampaignCacheRepo(ttl time.Duration) *CampaignCacheRepo {
	return NewCampaignCacheRepo(testRDB, config.CacheConfig{
		CampaignEventIDSeenTTL: ttl,
	})
}

func updateEvent(campaignID, stageID uuid.UUID, result domain.UpdateAttemptsResult) domain.UpdateResultsEvent {
	return domain.UpdateResultsEvent{
		EventID:    uuid.New(),
		CampaignID: campaignID,
		StageID:    stageID,
		Result:     result,
	}
}

func TestCampaignCacheRepo_UpdateStageResults_FirstEvent_IncrementsCounter(t *testing.T) {
	resetRedis(t)
	repo := newCampaignCacheRepo(time.Hour)
	ctx := context.Background()
	campaignID, stageID := uuid.New(), uuid.New()

	got, err := repo.UpdateStageResults(ctx, updateEvent(campaignID, stageID, domain.UpdateAttemptsResultSuccess))
	require.NoError(t, err)
	assert.Equal(t, 1, got)
}

func TestCampaignCacheRepo_UpdateStageResults_DuplicateEvent_ReturnsZero(t *testing.T) {
	resetRedis(t)
	repo := newCampaignCacheRepo(time.Hour)
	ctx := context.Background()
	campaignID, stageID := uuid.New(), uuid.New()

	event := domain.UpdateResultsEvent{
		EventID:    uuid.New(),
		CampaignID: campaignID,
		StageID:    stageID,
		Result:     domain.UpdateAttemptsResultSuccess,
	}

	first, err := repo.UpdateStageResults(ctx, event)
	require.NoError(t, err)
	assert.Equal(t, 1, first)

	second, err := repo.UpdateStageResults(ctx, event)
	require.NoError(t, err)
	assert.Equal(t, 0, second)
}

func TestCampaignCacheRepo_UpdateStageResults_DifferentEvents_IncrementIndependently(t *testing.T) {
	resetRedis(t)
	repo := newCampaignCacheRepo(time.Hour)
	ctx := context.Background()
	campaignID, stageID := uuid.New(), uuid.New()

	e1 := updateEvent(campaignID, stageID, domain.UpdateAttemptsResultSuccess)
	e2 := updateEvent(campaignID, stageID, domain.UpdateAttemptsResultSuccess)

	got1, err := repo.UpdateStageResults(ctx, e1)
	require.NoError(t, err)
	assert.Equal(t, 1, got1)

	got2, err := repo.UpdateStageResults(ctx, e2)
	require.NoError(t, err)
	assert.Equal(t, 2, got2)
}

func TestCampaignCacheRepo_UpdateStageResults_DifferentResults_WriteToDifferentKeys(t *testing.T) {
	resetRedis(t)
	repo := newCampaignCacheRepo(time.Hour)
	ctx := context.Background()
	campaignID, stageID := uuid.New(), uuid.New()

	successKey := "campaign:" + campaignID.String() + ":stage:" + stageID.String() + ":" + string(domain.UpdateAttemptsResultSuccess)
	failureKey := "campaign:" + campaignID.String() + ":stage:" + stageID.String() + ":" + string(domain.UpdateAttemptsResultFailure)

	_, err := repo.UpdateStageResults(ctx, updateEvent(campaignID, stageID, domain.UpdateAttemptsResultSuccess))
	require.NoError(t, err)
	_, err = repo.UpdateStageResults(ctx, updateEvent(campaignID, stageID, domain.UpdateAttemptsResultFailure))
	require.NoError(t, err)

	success, err := testRDB.Get(ctx, successKey).Int()
	require.NoError(t, err)
	assert.Equal(t, 1, success)

	failure, err := testRDB.Get(ctx, failureKey).Int()
	require.NoError(t, err)
	assert.Equal(t, 1, failure)
}

func TestCampaignCacheRepo_UpdateStageResults_SeenKeyExpires(t *testing.T) {
	resetRedis(t)
	repo := newCampaignCacheRepo(1 * time.Second)
	ctx := context.Background()
	campaignID, stageID := uuid.New(), uuid.New()

	event := domain.UpdateResultsEvent{
		EventID:    uuid.New(),
		CampaignID: campaignID,
		StageID:    stageID,
		Result:     domain.UpdateAttemptsResultSuccess,
	}

	first, err := repo.UpdateStageResults(ctx, event)
	require.NoError(t, err)
	assert.Equal(t, 1, first)

	require.Eventually(t, func() bool {
		_, err := repo.UpdateStageResults(ctx, event)
		return err == nil
	}, 5*time.Second, 100*time.Millisecond)

	// after the seen-key TTL passes, the same event id is allowed to increment again
	require.Eventually(t, func() bool {
		got, err := repo.UpdateStageResults(ctx, event)
		return err == nil && got == 2
	}, 5*time.Second, 100*time.Millisecond)
}

func TestCampaignCacheRepo_GetCampaignStats_NoCurrentStage_ReturnsErrCurrentStageNotFound(t *testing.T) {
	resetRedis(t)
	repo := newCampaignCacheRepo(time.Hour)
	ctx := context.Background()
	id := uuid.New()

	_, err := repo.GetCampaignStats(ctx, id)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrCurrentStageNotFound)
}

func TestCampaignCacheRepo_GetCampaignStats_EmptyCounters_ReturnsZeroStats(t *testing.T) {
	resetRedis(t)
	repo := newCampaignCacheRepo(time.Hour)
	ctx := context.Background()
	campaignID, stageID := uuid.New(), uuid.New()
	setCurrentStage(t, campaignID, stageID)

	stats, err := repo.GetCampaignStats(ctx, campaignID)
	require.NoError(t, err)
	assert.Equal(t, stageID, stats.ActiveStageID)
	assert.Equal(t, float32(0), stats.SuccessRate)
	assert.Equal(t, 0, stats.SampleSize)
}

func TestCampaignCacheRepo_GetCampaignStats_MixedCounters_CalculatesSuccessRate(t *testing.T) {
	resetRedis(t)
	repo := newCampaignCacheRepo(time.Hour)
	ctx := context.Background()
	campaignID, stageID := uuid.New(), uuid.New()
	setCurrentStage(t, campaignID, stageID)

	for i := 0; i < 3; i++ {
		_, err := repo.UpdateStageResults(ctx, updateEvent(campaignID, stageID, domain.UpdateAttemptsResultSuccess))
		require.NoError(t, err)
	}
	_, err := repo.UpdateStageResults(ctx, updateEvent(campaignID, stageID, domain.UpdateAttemptsResultFailure))
	require.NoError(t, err)
	_, err = repo.UpdateStageResults(ctx, updateEvent(campaignID, stageID, domain.UpdateAttemptsResultTimeout))
	require.NoError(t, err)

	stats, err := repo.GetCampaignStats(ctx, campaignID)
	require.NoError(t, err)
	assert.Equal(t, stageID, stats.ActiveStageID)
	assert.Equal(t, 5, stats.SampleSize)
	assert.InDelta(t, float64(0.6), float64(stats.SuccessRate), 0.0001)
}

func TestCampaignCacheRepo_GetCampaignStats_OnlyFailure_ReturnsZeroSuccessRate(t *testing.T) {
	resetRedis(t)
	repo := newCampaignCacheRepo(time.Hour)
	ctx := context.Background()
	campaignID, stageID := uuid.New(), uuid.New()
	setCurrentStage(t, campaignID, stageID)

	_, err := repo.UpdateStageResults(ctx, updateEvent(campaignID, stageID, domain.UpdateAttemptsResultFailure))
	require.NoError(t, err)

	stats, err := repo.GetCampaignStats(ctx, campaignID)
	require.NoError(t, err)
	assert.Equal(t, stageID, stats.ActiveStageID)
	assert.Equal(t, 1, stats.SampleSize)
	assert.Equal(t, float32(0), stats.SuccessRate)
}
