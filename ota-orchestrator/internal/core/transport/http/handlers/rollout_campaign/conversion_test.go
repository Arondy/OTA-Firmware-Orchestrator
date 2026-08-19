package rollout_campaign

import (
	"testing"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRolloutStageFromDomain_AllFieldsCopied(t *testing.T) {
	entered := time.Now()
	stage := domain.RolloutStage{
		ID:               uuid.New(),
		CampaignID:       uuid.New(),
		OrderIndex:       2,
		TargetPercent:    50,
		MinSampleSize:    10,
		SuccessThreshold: 0.9,
		Status:           domain.RolloutStagesStatusActive,
		EnteredAt:        &entered,
	}

	resp := RolloutStageFromDomain(stage)
	assert.Equal(t, stage.ID, resp.ID)
	assert.Equal(t, stage.CampaignID, resp.CampaignID)
	assert.Equal(t, stage.OrderIndex, resp.OrderIndex)
	assert.Equal(t, stage.TargetPercent, resp.TargetPercent)
	assert.Equal(t, stage.MinSampleSize, resp.MinSampleSize)
	assert.Equal(t, stage.SuccessThreshold, resp.SuccessThreshold)
	assert.Equal(t, stage.Status, resp.Status)
	require.NotNil(t, resp.EnteredAt)
	assert.Equal(t, entered.Unix(), resp.EnteredAt.Unix())
}

func TestRolloutCampaignListItemFromDomain_AllFieldsCopied(t *testing.T) {
	now := time.Now()
	started := now.Add(time.Minute)
	completed := now.Add(time.Hour)
	campaign := domain.RolloutCampaign{
		ID:                uuid.New(),
		FirmwareVersionID: uuid.New(),
		DeviceModel:       "model-z",
		Status:            domain.RolloutCampaignsStatusRunning,
		CreatedAt:         now,
		StartedAt:         &started,
		CompletedAt:       &completed,
	}

	resp := RolloutCampaignListItemFromDomain(campaign)
	assert.Equal(t, campaign.ID, resp.ID)
	assert.Equal(t, campaign.FirmwareVersionID, resp.FirmwareVersionID)
	assert.Equal(t, campaign.DeviceModel, resp.DeviceModel)
	assert.Equal(t, campaign.Status, resp.Status)
	assert.Equal(t, campaign.CreatedAt.Unix(), resp.CreatedAt.Unix())
	require.NotNil(t, resp.StartedAt)
	assert.Equal(t, started.Unix(), resp.StartedAt.Unix())
	require.NotNil(t, resp.CompletedAt)
	assert.Equal(t, completed.Unix(), resp.CompletedAt.Unix())
}

func TestRolloutCampaignFromDomain_WithStatsAndStages(t *testing.T) {
	now := time.Now()
	stage := domain.RolloutStage{
		ID:         uuid.New(),
		OrderIndex: 0,
		Status:     domain.RolloutStagesStatusActive,
	}
	campaign := domain.RolloutCampaign{
		ID:          uuid.New(),
		DeviceModel: "model-z",
		Status:      domain.RolloutCampaignsStatusRunning,
		CreatedAt:   now,
		RolloutStages: []domain.RolloutStage{
			stage,
		},
	}

	withStats := campaign
	withStats.Stats = &domain.RolloutCampaignStats{
		ActiveStageID: stage.ID,
		SuccessRate:   0.5,
		SampleSize:    2,
	}
	resp := RolloutCampaignFromDomain(withStats)
	require.Len(t, resp.RolloutStages, 1)
	assert.Equal(t, stage.ID, resp.RolloutStages[0].ID)
	require.NotNil(t, resp.Stats)
	assert.Equal(t, withStats.Stats.SampleSize, resp.Stats.SampleSize)

	withoutStats := campaign
	withoutStats.Stats = nil
	respNil := RolloutCampaignFromDomain(withoutStats)
	assert.Nil(t, respNil.Stats)
}
