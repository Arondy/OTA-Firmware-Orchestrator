//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUpdateAttemptCreate(t *testing.T) {
	resetDB(t)

	dev, err := NewDeviceRepo(testDB).Create(context.Background(), domain.Device{
		DeviceModel:    "model-a",
		CurrentVersion: "1.0.0",
	})
	require.NoError(t, err)

	fw := newFirmware(t, "model-a")
	campaignRepo := NewRolloutCampaignRepo(testDB)
	c := createCampaign(t, campaignRepo, fw.ID, "model-a", 1)
	started, err := campaignRepo.Start(context.Background(), c.ID)
	require.NoError(t, err)
	stageID := started.RolloutStages[0].ID

	eventID := uuid.New()
	created, err := NewUpdateAttemptRepo(testDB).Create(context.Background(), domain.UpdateAttempt{
		DeviceID:   dev.ID,
		CampaignID: c.ID,
		StageID:    stageID,
		Result:     domain.UpdateAttemptsResultSuccess,
		EventID:    eventID,
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, eventID, created.EventID)
	require.Equal(t, domain.UpdateAttemptsResultSuccess, created.Result)
	require.NotZero(t, created.ReportedAt)
}

func TestUpdateAttemptDuplicateEventID(t *testing.T) {
	resetDB(t)

	dev, err := NewDeviceRepo(testDB).Create(context.Background(), domain.Device{
		DeviceModel:    "model-a",
		CurrentVersion: "1.0.0",
	})
	require.NoError(t, err)

	fw := newFirmware(t, "model-a")
	campaignRepo := NewRolloutCampaignRepo(testDB)
	c := createCampaign(t, campaignRepo, fw.ID, "model-a", 1)
	started, err := campaignRepo.Start(context.Background(), c.ID)
	require.NoError(t, err)
	stageID := started.RolloutStages[0].ID

	attemptRepo := NewUpdateAttemptRepo(testDB)
	eventID := uuid.New()
	first, err := attemptRepo.Create(context.Background(), domain.UpdateAttempt{
		DeviceID:   dev.ID,
		CampaignID: c.ID,
		StageID:    stageID,
		Result:     domain.UpdateAttemptsResultSuccess,
		EventID:    eventID,
	})
	require.NoError(t, err)

	_, err = attemptRepo.Create(context.Background(), domain.UpdateAttempt{
		DeviceID:   dev.ID,
		CampaignID: c.ID,
		StageID:    stageID,
		Result:     domain.UpdateAttemptsResultFailure,
		EventID:    first.EventID,
	})
	require.Error(t, err)
}
