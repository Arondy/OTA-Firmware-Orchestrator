//go:build integration

package redis

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampaignCacheRepo_UpdateStageResults_ConcurrentSameEventID_IncrementsOnce(t *testing.T) {
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

	const goroutines = 64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _ = repo.UpdateStageResults(ctx, event)
		}()
	}
	close(start)
	wg.Wait()

	counterKey := fmt.Sprintf("campaign:%s:stage:%s:%s", campaignID, stageID, domain.UpdateAttemptsResultSuccess)
	got, err := testRDB.Get(ctx, counterKey).Int()
	require.NoError(t, err)
	assert.Equal(t, 1, got)
}

func TestCampaignCacheRepo_UpdateStageResults_ConcurrentDifferentEventIDs_IncrementsAll(t *testing.T) {
	resetRedis(t)
	repo := newCampaignCacheRepo(time.Hour)
	ctx := context.Background()
	campaignID, stageID := uuid.New(), uuid.New()

	const goroutines = 64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			event := domain.UpdateResultsEvent{
				EventID:    uuid.New(),
				CampaignID: campaignID,
				StageID:    stageID,
				Result:     domain.UpdateAttemptsResultSuccess,
			}
			_, _ = repo.UpdateStageResults(ctx, event)
		}()
	}
	close(start)
	wg.Wait()

	counterKey := fmt.Sprintf("campaign:%s:stage:%s:%s", campaignID, stageID, domain.UpdateAttemptsResultSuccess)
	got, err := testRDB.Get(ctx, counterKey).Int()
	require.NoError(t, err)
	assert.Equal(t, goroutines, got)
}
