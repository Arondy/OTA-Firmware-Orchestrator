//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newFirmware(t *testing.T, model string) domain.FirmwareVersion {
	t.Helper()
	repo := NewFirmwareVersionRepo(testDB)
	fw, err := repo.Create(context.Background(), domain.FirmwareVersion{
		DeviceModel: model,
		FWVersion:   "1.0.0",
		FWChecksum:  "checksum",
		BinaryUrl:   "http://example/fw.bin",
	})
	require.NoError(t, err)
	return fw
}

func createCampaign(t *testing.T, repo *RolloutCampaignRepo, fwID uuid.UUID, model string, stages int) domain.RolloutCampaign {
	t.Helper()
	stagesDef := make([]domain.RolloutStage, stages)
	for i := range stagesDef {
		stagesDef[i] = domain.RolloutStage{
			OrderIndex:       i,
			TargetPercent:    10 * (i + 1),
			MinSampleSize:    5,
			SuccessThreshold: 0.9,
		}
	}
	c, err := repo.Create(context.Background(), domain.RolloutCampaign{
		FirmwareVersionID: fwID,
		DeviceModel:       model,
		RolloutStages:     stagesDef,
	})
	require.NoError(t, err)
	return c
}

func TestCampaignCreateWithStages(t *testing.T) {
	resetDB(t)
	fw := newFirmware(t, "model-a")
	repo := NewRolloutCampaignRepo(testDB)

	created, err := repo.Create(context.Background(), domain.RolloutCampaign{
		FirmwareVersionID: fw.ID,
		DeviceModel:       "model-a",
		RolloutStages: []domain.RolloutStage{
			{OrderIndex: 0, TargetPercent: 10, MinSampleSize: 5, SuccessThreshold: 0.9},
			{OrderIndex: 1, TargetPercent: 50, MinSampleSize: 10, SuccessThreshold: 0.95},
		},
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, domain.RolloutCampaignsStatusDraft, created.Status)
	require.Len(t, created.RolloutStages, 2)
	for _, s := range created.RolloutStages {
		require.Equal(t, domain.RolloutStagesStatusPending, s.Status)
	}

	got, err := repo.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, got.RolloutStages, 2)
}

func TestCampaignCreateInvalidStage(t *testing.T) {
	resetDB(t)
	fw := newFirmware(t, "model-a")
	repo := NewRolloutCampaignRepo(testDB)

	// target_percent must be between 1 and 100
	_, err := repo.Create(context.Background(), domain.RolloutCampaign{
		FirmwareVersionID: fw.ID,
		DeviceModel:       "model-a",
		RolloutStages: []domain.RolloutStage{
			{OrderIndex: 0, TargetPercent: 0, MinSampleSize: 5, SuccessThreshold: 0.9},
		},
	})
	require.Error(t, err)
}

func TestCampaignStart(t *testing.T) {
	resetDB(t)
	fw := newFirmware(t, "model-a")
	repo := NewRolloutCampaignRepo(testDB)
	c := createCampaign(t, repo, fw.ID, "model-a", 2)

	started, err := repo.Start(context.Background(), c.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RolloutCampaignsStatusRunning, started.Status)
	require.NotNil(t, started.StartedAt)
	require.Len(t, started.RolloutStages, 2)
	require.Equal(t, domain.RolloutStagesStatusActive, started.RolloutStages[0].Status)
	require.Equal(t, domain.RolloutStagesStatusPending, started.RolloutStages[1].Status)
}

func TestCampaignStartWrongStatus(t *testing.T) {
	resetDB(t)
	fw := newFirmware(t, "model-a")
	repo := NewRolloutCampaignRepo(testDB)
	c := createCampaign(t, repo, fw.ID, "model-a", 2)

	_, err := repo.Start(context.Background(), c.ID)
	require.NoError(t, err)

	_, err = repo.Start(context.Background(), c.ID)
	require.ErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus)
}

func TestCampaignStartNotFound(t *testing.T) {
	resetDB(t)
	repo := NewRolloutCampaignRepo(testDB)

	_, err := repo.Start(context.Background(), uuid.New())
	require.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestCampaignPauseResume(t *testing.T) {
	resetDB(t)
	fw := newFirmware(t, "model-a")
	repo := NewRolloutCampaignRepo(testDB)
	c := createCampaign(t, repo, fw.ID, "model-a", 2)

	_, err := repo.Start(context.Background(), c.ID)
	require.NoError(t, err)

	paused, err := repo.Pause(context.Background(), c.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RolloutCampaignsStatusPaused, paused.Status)

	resumed, err := repo.Resume(context.Background(), c.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RolloutCampaignsStatusRunning, resumed.Status)

	_, err = repo.Resume(context.Background(), c.ID)
	require.ErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus)
}

func TestCampaignPauseNotFound(t *testing.T) {
	resetDB(t)
	repo := NewRolloutCampaignRepo(testDB)

	_, err := repo.Pause(context.Background(), uuid.New())
	require.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestCampaignAdvanceStages(t *testing.T) {
	resetDB(t)
	fw := newFirmware(t, "model-a")
	repo := NewRolloutCampaignRepo(testDB)
	c := createCampaign(t, repo, fw.ID, "model-a", 2)

	_, err := repo.Start(context.Background(), c.ID)
	require.NoError(t, err)

	adv, err := repo.AdvanceStage(context.Background(), c.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RolloutCampaignsStatusRunning, adv.Status)
	require.Len(t, adv.RolloutStages, 2)
	require.Equal(t, domain.RolloutStagesStatusPassed, adv.RolloutStages[0].Status)
	require.Equal(t, domain.RolloutStagesStatusActive, adv.RolloutStages[1].Status)

	adv, err = repo.AdvanceStage(context.Background(), c.ID)
	require.NoError(t, err)
	require.Equal(t, domain.RolloutCampaignsStatusCompleted, adv.Status)
	require.NotNil(t, adv.CompletedAt)

	_, err = repo.AdvanceStage(context.Background(), c.ID)
	require.ErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus)
}

func TestCampaignAdvanceNoActiveStage(t *testing.T) {
	resetDB(t)
	fw := newFirmware(t, "model-a")
	repo := NewRolloutCampaignRepo(testDB)
	c := createCampaign(t, repo, fw.ID, "model-a", 1)

	// draft campaign has no active stage
	_, err := repo.AdvanceStage(context.Background(), c.ID)
	require.ErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus)
}

func TestCampaignOnlyOneRunningPerModel(t *testing.T) {
	resetDB(t)
	fw := newFirmware(t, "model-a")
	repo := NewRolloutCampaignRepo(testDB)
	c1 := createCampaign(t, repo, fw.ID, "model-a", 1)
	c2 := createCampaign(t, repo, fw.ID, "model-a", 1)

	_, err := repo.Start(context.Background(), c1.ID)
	require.NoError(t, err)

	_, err = repo.Start(context.Background(), c2.ID)
	require.ErrorIs(t, err, domain.ErrCampaignAlreadyRunning)
}

func TestCampaignFindRunning(t *testing.T) {
	resetDB(t)
	fw := newFirmware(t, "model-a")
	repo := NewRolloutCampaignRepo(testDB)
	c := createCampaign(t, repo, fw.ID, "model-a", 1)

	_, err := repo.Start(context.Background(), c.ID)
	require.NoError(t, err)

	running, err := repo.FindRunning(context.Background(), "model-a")
	require.NoError(t, err)
	require.Equal(t, c.ID, running.ID)

	list, err := repo.ListRunning(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, c.ID, list[0].ID)

	_, err = repo.FindRunning(context.Background(), "model-x")
	require.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestCampaignFindActiveStages(t *testing.T) {
	resetDB(t)
	fw := newFirmware(t, "model-a")
	repo := NewRolloutCampaignRepo(testDB)
	c := createCampaign(t, repo, fw.ID, "model-a", 2)

	_, err := repo.Start(context.Background(), c.ID)
	require.NoError(t, err)

	active, err := repo.FindActiveStages(context.Background(), []uuid.UUID{c.ID})
	require.NoError(t, err)
	require.Len(t, active, 1)
	require.Equal(t, domain.RolloutStagesStatusActive, active[0].Status)
	require.Equal(t, c.ID, active[0].CampaignID)
}
