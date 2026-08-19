package update_test

import (
	"context"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newReportMocks(t *testing.T) *checkinMocks {
	return newCheckinMocks(t)
}

func reportCampaignWithStage(campaignID, stageID uuid.UUID) domain.RolloutCampaign {
	return domain.RolloutCampaign{
		ID:          campaignID,
		DeviceModel: "model-a",
		RolloutStages: []domain.RolloutStage{
			{ID: stageID, CampaignID: campaignID, Status: domain.RolloutStagesStatusActive},
		},
	}
}

func TestReport_CampaignNotFound_ReturnsError(t *testing.T) {
	m := newReportMocks(t)
	campaignID := uuid.New()

	m.campaignRepo.EXPECT().Get(mock.Anything, campaignID).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	_, err := m.service().Report(context.Background(), domain.UpdateAttempt{CampaignID: campaignID})
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestReport_StageNotInCampaign_ReturnsErrRolloutStageNotFoundInCampaign(t *testing.T) {
	m := newReportMocks(t)
	campaignID := uuid.New()
	otherStage := uuid.New()

	m.campaignRepo.EXPECT().Get(mock.Anything, campaignID).Return(reportCampaignWithStage(campaignID, otherStage), nil)

	_, err := m.service().Report(context.Background(), domain.UpdateAttempt{
		CampaignID: campaignID,
		StageID:    uuid.New(),
	})
	assert.ErrorIs(t, err, domain.ErrRolloutStageNotFoundInCampaign)
}

func TestReport_DeviceNotFound_ReturnsErrDeviceNotFound(t *testing.T) {
	m := newReportMocks(t)
	campaignID := uuid.New()
	stageID := uuid.New()
	deviceID := uuid.New()

	m.campaignRepo.EXPECT().Get(mock.Anything, campaignID).Return(reportCampaignWithStage(campaignID, stageID), nil)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(domain.Device{}, domain.ErrDeviceNotFound)

	_, err := m.service().Report(context.Background(), domain.UpdateAttempt{
		CampaignID: campaignID,
		StageID:    stageID,
		DeviceID:   deviceID,
	})
	assert.ErrorIs(t, err, domain.ErrDeviceNotFound)
}

func TestReport_DeviceModelMismatch_ReturnsErrWrongDeviceModel(t *testing.T) {
	m := newReportMocks(t)
	campaignID := uuid.New()
	stageID := uuid.New()
	deviceID := uuid.New()

	m.campaignRepo.EXPECT().Get(mock.Anything, campaignID).Return(reportCampaignWithStage(campaignID, stageID), nil)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(domain.Device{ID: deviceID, DeviceModel: "other-model"}, nil)

	_, err := m.service().Report(context.Background(), domain.UpdateAttempt{
		CampaignID: campaignID,
		StageID:    stageID,
		DeviceID:   deviceID,
	})
	assert.ErrorIs(t, err, domain.ErrWrongDeviceModel)
}

func TestReport_CreateAttemptFails_ReturnsError(t *testing.T) {
	m := newReportMocks(t)
	campaignID := uuid.New()
	stageID := uuid.New()
	deviceID := uuid.New()

	m.campaignRepo.EXPECT().Get(mock.Anything, campaignID).Return(reportCampaignWithStage(campaignID, stageID), nil)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(domain.Device{ID: deviceID, DeviceModel: "model-a"}, nil)
	m.updateAttemptRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.UpdateAttempt{}, someErr())

	_, err := m.service().Report(context.Background(), domain.UpdateAttempt{
		CampaignID: campaignID,
		StageID:    stageID,
		DeviceID:   deviceID,
		Result:     domain.UpdateAttemptsResultSuccess,
	})
	require.Error(t, err)
}

func TestReport_ProduceFails_ReturnsErrUpdateResultNotProduced(t *testing.T) {
	m := newReportMocks(t)
	campaignID := uuid.New()
	stageID := uuid.New()
	deviceID := uuid.New()

	m.campaignRepo.EXPECT().Get(mock.Anything, campaignID).Return(reportCampaignWithStage(campaignID, stageID), nil)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(domain.Device{ID: deviceID, DeviceModel: "model-a"}, nil)
	m.updateAttemptRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.UpdateAttempt{}, nil)
	m.updateResultsProd.EXPECT().Produce(mock.Anything).Return(someErr())

	_, err := m.service().Report(context.Background(), domain.UpdateAttempt{
		CampaignID: campaignID,
		StageID:    stageID,
		DeviceID:   deviceID,
		Result:     domain.UpdateAttemptsResultSuccess,
	})
	assert.ErrorIs(t, err, domain.ErrUpdateResultNotProduced)
}

func TestReport_Success_GeneratesEventIDAndProduces(t *testing.T) {
	m := newReportMocks(t)
	campaignID := uuid.New()
	stageID := uuid.New()
	deviceID := uuid.New()

	var createdEventID uuid.UUID

	m.campaignRepo.EXPECT().Get(mock.Anything, campaignID).Return(reportCampaignWithStage(campaignID, stageID), nil)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(domain.Device{ID: deviceID, DeviceModel: "model-a"}, nil)
	m.updateAttemptRepo.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, attempt domain.UpdateAttempt) (domain.UpdateAttempt, error) {
		require.NotEqual(t, uuid.Nil, attempt.EventID)
		createdEventID = attempt.EventID
		return attempt, nil
	})
	m.updateResultsProd.EXPECT().Produce(mock.Anything).Run(func(event domain.UpdateResultsEvent) {
		assert.Equal(t, createdEventID, event.EventID)
		assert.Equal(t, campaignID, event.CampaignID)
		assert.Equal(t, stageID, event.StageID)
		assert.Equal(t, deviceID, event.DeviceID)
	}).Return(nil)

	_, err := m.service().Report(context.Background(), domain.UpdateAttempt{
		CampaignID: campaignID,
		StageID:    stageID,
		DeviceID:   deviceID,
		Result:     domain.UpdateAttemptsResultSuccess,
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, createdEventID)
}

func TestReport_Success_AttemptHasEventIDBeforeCreate(t *testing.T) {
	m := newReportMocks(t)
	campaignID := uuid.New()
	stageID := uuid.New()
	deviceID := uuid.New()

	m.campaignRepo.EXPECT().Get(mock.Anything, campaignID).Return(reportCampaignWithStage(campaignID, stageID), nil)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(domain.Device{ID: deviceID, DeviceModel: "model-a"}, nil)
	m.updateAttemptRepo.EXPECT().Create(mock.Anything, mock.Anything).Run(func(_ context.Context, attempt domain.UpdateAttempt) {
		require.NotEqual(t, uuid.Nil, attempt.EventID, "EventID must be set before Create")
	}).Return(domain.UpdateAttempt{}, nil)
	m.updateResultsProd.EXPECT().Produce(mock.Anything).Return(nil)

	_, err := m.service().Report(context.Background(), domain.UpdateAttempt{
		CampaignID: campaignID,
		StageID:    stageID,
		DeviceID:   deviceID,
		Result:     domain.UpdateAttemptsResultSuccess,
	})
	require.NoError(t, err)
}
