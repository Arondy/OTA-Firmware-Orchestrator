package campaign_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/campaign"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/campaign/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type campaignMocks struct {
	campaignRepo *mocks.MockRolloutCampaignRepo
	firmwareRepo *mocks.MockFirmwareVersionRepo
	cache        *mocks.MockCampaignCacheRepo
	controller   *mocks.MockRolloutController
}

func newCampaignMocks(t *testing.T) *campaignMocks {
	return &campaignMocks{
		campaignRepo: mocks.NewMockRolloutCampaignRepo(t),
		firmwareRepo: mocks.NewMockFirmwareVersionRepo(t),
		cache:        mocks.NewMockCampaignCacheRepo(t),
		controller:   mocks.NewMockRolloutController(t),
	}
}

func (m *campaignMocks) service() *campaign.RolloutCampaignService {
	return campaign.NewService(m.campaignRepo, m.firmwareRepo, m.cache, m.controller)
}

func campaignWithStages(id uuid.UUID, status domain.RolloutCampaignsStatus, stages []domain.RolloutStage) domain.RolloutCampaign {
	return domain.RolloutCampaign{
		ID:            id,
		DeviceModel:   "model-a",
		Status:        status,
		RolloutStages: stages,
	}
}

func twoStages(campaignID uuid.UUID) []domain.RolloutStage {
	return []domain.RolloutStage{
		{ID: uuid.New(), CampaignID: campaignID, OrderIndex: 0, TargetPercent: 50, Status: domain.RolloutStagesStatusActive},
		{ID: uuid.New(), CampaignID: campaignID, OrderIndex: 1, TargetPercent: 100, Status: domain.RolloutStagesStatusPending},
	}
}

// --- Create ---

func TestCreate_FirmwareNotFound_ReturnsError(t *testing.T) {
	m := newCampaignMocks(t)
	fwID := uuid.New()

	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(domain.FirmwareVersion{}, domain.ErrFirmwareVersionNotFound)

	_, err := m.service().Create(context.Background(), domain.RolloutCampaign{FirmwareVersionID: fwID})
	assert.ErrorIs(t, err, domain.ErrFirmwareVersionNotFound)
}

func TestCreate_Success_SetsDeviceModelFromFirmware(t *testing.T) {
	m := newCampaignMocks(t)
	fwID := uuid.New()
	fw := domain.FirmwareVersion{ID: fwID, DeviceModel: "model-x"}

	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(fw, nil)
	m.campaignRepo.EXPECT().Create(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, c domain.RolloutCampaign) (domain.RolloutCampaign, error) {
		assert.Equal(t, "model-x", c.DeviceModel)
		c.ID = uuid.New()
		return c, nil
	})

	created, err := m.service().Create(context.Background(), domain.RolloutCampaign{FirmwareVersionID: fwID})
	require.NoError(t, err)
	assert.Equal(t, "model-x", created.DeviceModel)
}

// --- Start ---

func TestStart_CampaignNotFound_ReturnsError(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	_, err := m.service().Start(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestStart_WrongStatus_ReturnsErrRolloutCampaignWrongStatus(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)

	_, err := m.service().Start(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus)
}

func TestStart_RepoStartFails_ReturnsError(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusDraft, nil), nil)
	m.campaignRepo.EXPECT().Start(mock.Anything, id).Return(domain.RolloutCampaign{}, someCampaignErr())

	_, err := m.service().Start(context.Background(), id)
	require.Error(t, err)
}

func TestStart_NoStages_ReturnsError(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusDraft, nil), nil)
	m.campaignRepo.EXPECT().Start(mock.Anything, id).Return(domain.RolloutCampaign{ID: id, Status: domain.RolloutCampaignsStatusRunning}, nil)

	_, err := m.service().Start(context.Background(), id)
	require.Error(t, err)
}

func TestStart_Success_SetsCacheForFirstStage(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	started := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusDraft, nil), nil)
	m.campaignRepo.EXPECT().Start(mock.Anything, id).Return(started, nil)
	m.cache.EXPECT().SetCurrentStage(mock.Anything, id, stages[0].ID).Return(nil)
	m.cache.EXPECT().SetCurrentTargetPercent(mock.Anything, id, stages[0].TargetPercent).Return(nil)

	result, err := m.service().Start(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusRunning, result.Status)
}

func TestStart_CacheSetFails_StillReturnsCampaign(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	started := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusDraft, nil), nil)
	m.campaignRepo.EXPECT().Start(mock.Anything, id).Return(started, nil)
	m.cache.EXPECT().SetCurrentStage(mock.Anything, id, stages[0].ID).Return(someCampaignErr())
	m.cache.EXPECT().SetCurrentTargetPercent(mock.Anything, id, stages[0].TargetPercent).Return(someCampaignErr())

	result, err := m.service().Start(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusRunning, result.Status)
}

// --- Pause ---

func TestPause_CampaignNotFound_ReturnsError(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	_, err := m.service().Pause(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestPause_WrongStatus_ReturnsErrRolloutCampaignWrongStatus(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusDraft, nil), nil)

	_, err := m.service().Pause(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus)
}

func TestPause_Success(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().Pause(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusPaused, nil), nil)

	result, err := m.service().Pause(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusPaused, result.Status)
}

// --- Resume ---

func TestResume_CampaignNotFound_ReturnsError(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	_, err := m.service().Resume(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestResume_WrongStatus_ReturnsErrRolloutCampaignWrongStatus(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)

	_, err := m.service().Resume(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus)
}

func TestResume_RepoResumeFails_ReturnsError(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusPaused, nil), nil)
	m.campaignRepo.EXPECT().Resume(mock.Anything, id).Return(domain.RolloutCampaign{}, someCampaignErr())

	_, err := m.service().Resume(context.Background(), id)
	require.Error(t, err)
}

func TestResume_NoActiveStage_ReturnsCampaignWithoutCacheWrite(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := []domain.RolloutStage{{ID: uuid.New(), CampaignID: id, OrderIndex: 0, Status: domain.RolloutStagesStatusPassed}}
	resumed := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusPaused, nil), nil)
	m.campaignRepo.EXPECT().Resume(mock.Anything, id).Return(resumed, nil)

	result, err := m.service().Resume(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusRunning, result.Status)
}

func TestResume_Success_SetsCacheForActiveStage(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	resumed := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusPaused, nil), nil)
	m.campaignRepo.EXPECT().Resume(mock.Anything, id).Return(resumed, nil)
	m.cache.EXPECT().SetCurrentStage(mock.Anything, id, stages[0].ID).Return(nil)
	m.cache.EXPECT().SetCurrentTargetPercent(mock.Anything, id, stages[0].TargetPercent).Return(nil)

	result, err := m.service().Resume(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusRunning, result.Status)
}

func TestResume_CacheSetFails_StillReturnsCampaign(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	resumed := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusPaused, nil), nil)
	m.campaignRepo.EXPECT().Resume(mock.Anything, id).Return(resumed, nil)
	m.cache.EXPECT().SetCurrentStage(mock.Anything, id, stages[0].ID).Return(someCampaignErr())
	m.cache.EXPECT().SetCurrentTargetPercent(mock.Anything, id, stages[0].TargetPercent).Return(someCampaignErr())

	result, err := m.service().Resume(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusRunning, result.Status)
}

// --- AdvanceStage ---

func TestAdvanceStage_CampaignNotFound_ReturnsError(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	_, err := m.service().AdvanceStage(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestAdvanceStage_WrongStatus_ReturnsErrRolloutCampaignWrongStatus(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusPaused, nil), nil)

	_, err := m.service().AdvanceStage(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus)
}

func TestAdvanceStage_RepoAdvanceFails_ReturnsError(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().AdvanceStage(mock.Anything, id).Return(domain.RolloutCampaign{}, someCampaignErr())

	_, err := m.service().AdvanceStage(context.Background(), id)
	require.Error(t, err)
}

func TestAdvanceStage_Completed_DeletesCacheKeys(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	completed := campaignWithStages(id, domain.RolloutCampaignsStatusCompleted, nil)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().AdvanceStage(mock.Anything, id).Return(completed, nil)
	m.cache.EXPECT().DeleteCurrentStage(mock.Anything, id).Return(nil)
	m.cache.EXPECT().DeleteCurrentTargetPercent(mock.Anything, id).Return(nil)

	result, err := m.service().AdvanceStage(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusCompleted, result.Status)
}

func TestAdvanceStage_NextStageActive_SetsCacheForNewStage(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	advanced := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().AdvanceStage(mock.Anything, id).Return(advanced, nil)
	m.cache.EXPECT().SetCurrentStage(mock.Anything, id, stages[0].ID).Return(nil)
	m.cache.EXPECT().SetCurrentTargetPercent(mock.Anything, id, stages[0].TargetPercent).Return(nil)

	result, err := m.service().AdvanceStage(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusRunning, result.Status)
}

func TestAdvanceStage_NoActiveStageAfterAdvance_ReturnsCampaignWithoutCacheWrite(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := []domain.RolloutStage{{ID: uuid.New(), CampaignID: id, OrderIndex: 0, Status: domain.RolloutStagesStatusPassed}}
	advanced := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().AdvanceStage(mock.Anything, id).Return(advanced, nil)

	result, err := m.service().AdvanceStage(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusRunning, result.Status)
}

func TestAdvanceStage_CacheFails_StillReturnsCampaign(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	advanced := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().AdvanceStage(mock.Anything, id).Return(advanced, nil)
	m.cache.EXPECT().SetCurrentStage(mock.Anything, id, stages[0].ID).Return(someCampaignErr())
	m.cache.EXPECT().SetCurrentTargetPercent(mock.Anything, id, stages[0].TargetPercent).Return(someCampaignErr())

	result, err := m.service().AdvanceStage(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusRunning, result.Status)
}

// --- Get ---

func TestGet_CampaignNotFound_ReturnsError(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	_, err := m.service().Get(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestGet_ControllerFails_ReturnsCampaignWithoutStats(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.controller.EXPECT().GetCampaignStats(mock.Anything, id).Return(domain.RolloutCampaignStats{}, someCampaignErr())

	result, err := m.service().Get(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, result.Stats)
}

func TestGet_ControllerReturnsInvalidUUID_ReturnsCampaignWithoutStats(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.controller.EXPECT().GetCampaignStats(mock.Anything, id).Return(domain.RolloutCampaignStats{}, errors.New("invalid uuid"))

	result, err := m.service().Get(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, result.Stats)
}

func TestGet_Success_ReturnsCampaignWithStats(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	stats := domain.RolloutCampaignStats{ActiveStageID: uuid.New(), SuccessRate: 0.5, SampleSize: 2}
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.controller.EXPECT().GetCampaignStats(mock.Anything, id).Return(stats, nil)

	result, err := m.service().Get(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, result.Stats)
	assert.Equal(t, stats.SampleSize, result.Stats.SampleSize)
}

func TestList_DelegatesToRepo(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().List(mock.Anything).Return([]domain.RolloutCampaign{campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil)}, nil)

	result, err := m.service().List(context.Background())
	require.NoError(t, err)
	require.Len(t, result, 1)
}

// --- WarmUpCache ---

func TestWarmUpCache_ListRunningFails_ReturnsError(t *testing.T) {
	m := newCampaignMocks(t)
	m.campaignRepo.EXPECT().ListRunning(mock.Anything).Return(nil, someCampaignErr())

	err := m.service().WarmUpCache(context.Background(), zap.NewNop().Sugar())
	require.Error(t, err)
}

func TestWarmUpCache_FindActiveStagesFails_ReturnsError(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().ListRunning(mock.Anything).Return([]domain.RolloutCampaign{campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil)}, nil)
	m.campaignRepo.EXPECT().FindActiveStages(mock.Anything, mock.Anything).Return(nil, someCampaignErr())

	err := m.service().WarmUpCache(context.Background(), zap.NewNop().Sugar())
	require.Error(t, err)
}

func TestWarmUpCache_Success_SetsCacheForAllActiveStages(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	m.campaignRepo.EXPECT().ListRunning(mock.Anything).Return([]domain.RolloutCampaign{campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil)}, nil)
	m.campaignRepo.EXPECT().FindActiveStages(mock.Anything, mock.Anything).Return(stages, nil)
	m.cache.EXPECT().SetCurrentStage(mock.Anything, id, stages[0].ID).Return(nil)
	m.cache.EXPECT().SetCurrentTargetPercent(mock.Anything, id, stages[0].TargetPercent).Return(nil)
	m.cache.EXPECT().SetCurrentStage(mock.Anything, id, stages[1].ID).Return(nil)
	m.cache.EXPECT().SetCurrentTargetPercent(mock.Anything, id, stages[1].TargetPercent).Return(nil)

	err := m.service().WarmUpCache(context.Background(), zap.NewNop().Sugar())
	require.NoError(t, err)
}

func TestWarmUpCache_PartialCacheFailure_ContinuesProcessing(t *testing.T) {
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	m.campaignRepo.EXPECT().ListRunning(mock.Anything).Return([]domain.RolloutCampaign{campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil)}, nil)
	m.campaignRepo.EXPECT().FindActiveStages(mock.Anything, mock.Anything).Return(stages, nil)
	m.cache.EXPECT().SetCurrentStage(mock.Anything, id, stages[0].ID).Return(someCampaignErr())
	m.cache.EXPECT().SetCurrentTargetPercent(mock.Anything, id, stages[0].TargetPercent).Return(someCampaignErr())
	m.cache.EXPECT().SetCurrentStage(mock.Anything, id, stages[1].ID).Return(nil)
	m.cache.EXPECT().SetCurrentTargetPercent(mock.Anything, id, stages[1].TargetPercent).Return(nil)

	err := m.service().WarmUpCache(context.Background(), zap.NewNop().Sugar())
	require.NoError(t, err)
}

func TestWarmUpCache_NoRunningCampaigns_NoCacheWrites(t *testing.T) {
	m := newCampaignMocks(t)
	m.campaignRepo.EXPECT().ListRunning(mock.Anything).Return([]domain.RolloutCampaign{}, nil)
	m.campaignRepo.EXPECT().FindActiveStages(mock.Anything, mock.Anything).Return(nil, nil)

	err := m.service().WarmUpCache(context.Background(), zap.NewNop().Sugar())
	require.NoError(t, err)
}

func someCampaignErr() error {
	return errors.New("boom")
}
