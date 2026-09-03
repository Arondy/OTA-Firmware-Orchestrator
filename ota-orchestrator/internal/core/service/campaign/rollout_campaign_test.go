package campaign_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

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
	decisionRepo *mocks.MockAppliedDecisionRepo
	cache        *mocks.MockCampaignCacheRepo
	stageCache   *mocks.MockStageCacheRepo
	controller   *mocks.MockRolloutController
}

type stubTxManager struct{}

func (s *stubTxManager) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func newCampaignMocks(t *testing.T) *campaignMocks {
	return &campaignMocks{
		campaignRepo: mocks.NewMockRolloutCampaignRepo(t),
		firmwareRepo: mocks.NewMockFirmwareVersionRepo(t),
		decisionRepo: mocks.NewMockAppliedDecisionRepo(t),
		cache:        mocks.NewMockCampaignCacheRepo(t),
		stageCache:   mocks.NewMockStageCacheRepo(t),
		controller:   mocks.NewMockRolloutController(t),
	}
}

func (m *campaignMocks) service() *campaign.RolloutCampaignService {
	return campaign.NewService(m.campaignRepo, m.firmwareRepo, m.decisionRepo, &stubTxManager{}, m.cache, m.stageCache, m.controller)
}

func (m *campaignMocks) serviceWithTx(tx campaign.TxManager) *campaign.RolloutCampaignService {
	return campaign.NewService(m.campaignRepo, m.firmwareRepo, m.decisionRepo, tx, m.cache, m.stageCache, m.controller)
}

func advanceDecision(campaignID, prevStageID uuid.UUID) domain.DecisionEvent {
	return domain.DecisionEvent{
		DecisionID:      uuid.New(),
		CampaignID:      campaignID,
		DecisionType:    domain.DecisionTypeAdvance,
		PreviousStageID: prevStageID,
		Timestamp:       time.Now(),
	}
}

func rollbackDecision(campaignID, prevStageID uuid.UUID) domain.DecisionEvent {
	return domain.DecisionEvent{
		DecisionID:      uuid.New(),
		CampaignID:      campaignID,
		DecisionType:    domain.DecisionTypeRollback,
		PreviousStageID: prevStageID,
		Timestamp:       time.Now(),
	}
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
		{ID: uuid.New(), CampaignID: campaignID, OrderIndex: 0, TargetPercent: 50, MinSampleSize: 10, SuccessThreshold: 0.9, Status: domain.RolloutStagesStatusActive},
		{ID: uuid.New(), CampaignID: campaignID, OrderIndex: 1, TargetPercent: 100, MinSampleSize: 20, SuccessThreshold: 0.95, Status: domain.RolloutStagesStatusPending},
	}
}

// --- Create ---

func TestCreate_FirmwareNotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	fwID := uuid.New()

	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(domain.FirmwareVersion{}, domain.ErrFirmwareVersionNotFound)

	_, err := m.service().Create(context.Background(), domain.RolloutCampaign{FirmwareVersionID: fwID})
	assert.ErrorIs(t, err, domain.ErrFirmwareVersionNotFound)
}

func TestCreate_Success_SetsDeviceModelFromFirmware(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	_, err := m.service().Start(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestStart_WrongStatus_ReturnsErrRolloutCampaignWrongStatus(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)

	_, err := m.service().Start(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus)
}

func TestStart_RepoStartFails_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusDraft, nil), nil)
	m.campaignRepo.EXPECT().Start(mock.Anything, id).Return(domain.RolloutCampaign{}, someCampaignErr())

	_, err := m.service().Start(context.Background(), id)
	require.Error(t, err)
}

func TestStart_NoStages_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusDraft, nil), nil)
	m.campaignRepo.EXPECT().Start(mock.Anything, id).Return(domain.RolloutCampaign{ID: id, Status: domain.RolloutCampaignsStatusRunning}, nil)

	_, err := m.service().Start(context.Background(), id)
	require.Error(t, err)
}

func TestStart_Success_SetsCacheForFirstStage(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	started := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusDraft, nil), nil)
	m.campaignRepo.EXPECT().Start(mock.Anything, id).Return(started, nil)
	m.cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CheckinData{StageID: stages[0].ID, TargetPercent: stages[0].TargetPercent}).Return(nil)
	m.cache.EXPECT().AddRunningCampaigns(mock.Anything, id).Return(nil)
	m.stageCache.EXPECT().SetStageStats(mock.Anything, stages[0].ID, domain.StageStats{MinSampleSize: stages[0].MinSampleSize, SuccessThreshold: stages[0].SuccessThreshold}).Return(nil)

	result, err := m.service().Start(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusRunning, result.Status)
}

func TestStart_CacheSetFails_StillReturnsCampaign(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	started := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusDraft, nil), nil)
	m.campaignRepo.EXPECT().Start(mock.Anything, id).Return(started, nil)
	m.cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CheckinData{StageID: stages[0].ID, TargetPercent: stages[0].TargetPercent}).Return(someCampaignErr())
	m.cache.EXPECT().AddRunningCampaigns(mock.Anything, id).Return(someCampaignErr())
	m.stageCache.EXPECT().SetStageStats(mock.Anything, stages[0].ID, domain.StageStats{MinSampleSize: stages[0].MinSampleSize, SuccessThreshold: stages[0].SuccessThreshold}).Return(someCampaignErr())

	result, err := m.service().Start(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusRunning, result.Status)
}

// --- Pause ---

func TestPause_CampaignNotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	_, err := m.service().Pause(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestPause_WrongStatus_ReturnsErrRolloutCampaignWrongStatus(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusDraft, nil), nil)

	_, err := m.service().Pause(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus)
}

func TestPause_Success(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().Pause(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusPaused, nil), nil)
	m.cache.EXPECT().RemoveRunningCampaigns(mock.Anything, id).Return(nil)

	result, err := m.service().Pause(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusPaused, result.Status)
}

// --- Resume ---

func TestResume_CampaignNotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	_, err := m.service().Resume(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestResume_WrongStatus_ReturnsErrRolloutCampaignWrongStatus(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)

	_, err := m.service().Resume(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus)
}

func TestResume_RepoResumeFails_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusPaused, nil), nil)
	m.campaignRepo.EXPECT().Resume(mock.Anything, id).Return(domain.RolloutCampaign{}, someCampaignErr())

	_, err := m.service().Resume(context.Background(), id)
	require.Error(t, err)
}

func TestResume_NoActiveStage_ReturnsCampaignWithoutCacheWrite(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	resumed := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusPaused, nil), nil)
	m.campaignRepo.EXPECT().Resume(mock.Anything, id).Return(resumed, nil)
	m.cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CheckinData{StageID: stages[0].ID, TargetPercent: stages[0].TargetPercent}).Return(nil)
	m.cache.EXPECT().AddRunningCampaigns(mock.Anything, id).Return(nil)
	m.stageCache.EXPECT().SetStageStats(mock.Anything, stages[0].ID, domain.StageStats{MinSampleSize: stages[0].MinSampleSize, SuccessThreshold: stages[0].SuccessThreshold}).Return(nil)

	result, err := m.service().Resume(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusRunning, result.Status)
}

func TestResume_CacheSetFails_StillReturnsCampaign(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	resumed := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)

	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusPaused, nil), nil)
	m.campaignRepo.EXPECT().Resume(mock.Anything, id).Return(resumed, nil)
	m.cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CheckinData{StageID: stages[0].ID, TargetPercent: stages[0].TargetPercent}).Return(someCampaignErr())
	m.cache.EXPECT().AddRunningCampaigns(mock.Anything, id).Return(someCampaignErr())
	m.stageCache.EXPECT().SetStageStats(mock.Anything, stages[0].ID, domain.StageStats{MinSampleSize: stages[0].MinSampleSize, SuccessThreshold: stages[0].SuccessThreshold}).Return(someCampaignErr())

	result, err := m.service().Resume(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, domain.RolloutCampaignsStatusRunning, result.Status)
}

// --- ApplyDecision (advance_stage) ---

func TestApplyDecision_Advance_CampaignNotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	decision := advanceDecision(id, uuid.New())

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	err := m.service().ApplyDecision(context.Background(), decision)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestApplyDecision_Advance_WrongStatus_NoOpReturnsNil(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	prevStageID := uuid.New()
	decision := advanceDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusPaused, nil), nil)

	err := m.service().ApplyDecision(context.Background(), decision)
	require.NoError(t, err)
}

func TestApplyDecision_Advance_RepoAdvanceFails_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	prevStageID := uuid.New()
	decision := advanceDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().AdvanceStage(mock.Anything, id, prevStageID).Return(domain.RolloutCampaign{}, someCampaignErr())

	err := m.service().ApplyDecision(context.Background(), decision)
	require.Error(t, err)
}

func TestApplyDecision_Advance_Completed_DeletesCacheKeys(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	prevStageID := uuid.New()
	completed := campaignWithStages(id, domain.RolloutCampaignsStatusCompleted, nil)
	decision := advanceDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().AdvanceStage(mock.Anything, id, prevStageID).Return(completed, nil)
	m.cache.EXPECT().RemoveRunningCampaigns(mock.Anything, id).Return(nil)
	m.cache.EXPECT().DeleteCheckinData(mock.Anything, id).Return(nil)
	m.stageCache.EXPECT().DeleteStageStats(mock.Anything, prevStageID).Return(nil)

	err := m.service().ApplyDecision(context.Background(), decision)
	require.NoError(t, err)
}

func TestApplyDecision_Advance_NextStageActive_SetsCacheForNewStage(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	prevStageID := stages[0].ID
	stages[0].Status = domain.RolloutStagesStatusPassed
	stages[1].Status = domain.RolloutStagesStatusActive
	advanced := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)
	decision := advanceDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().AdvanceStage(mock.Anything, id, prevStageID).Return(advanced, nil)
	m.stageCache.EXPECT().DeleteStageStats(mock.Anything, prevStageID).Return(nil)
	m.cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CheckinData{StageID: stages[1].ID, TargetPercent: stages[1].TargetPercent}).Return(nil)
	m.cache.EXPECT().AddRunningCampaigns(mock.Anything, id).Return(nil)
	m.stageCache.EXPECT().SetStageStats(mock.Anything, stages[1].ID, domain.StageStats{MinSampleSize: stages[1].MinSampleSize, SuccessThreshold: stages[1].SuccessThreshold}).Return(nil)

	err := m.service().ApplyDecision(context.Background(), decision)
	require.NoError(t, err)
}

func TestApplyDecision_Advance_NoActiveStageAfterAdvance_ReturnsNil(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	prevStageID := uuid.New()
	stages := []domain.RolloutStage{{ID: uuid.New(), CampaignID: id, OrderIndex: 0, Status: domain.RolloutStagesStatusPassed}}
	advanced := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)
	decision := advanceDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().AdvanceStage(mock.Anything, id, prevStageID).Return(advanced, nil)

	err := m.service().ApplyDecision(context.Background(), decision)
	require.NoError(t, err)
}

func TestApplyDecision_Advance_CacheFails_StillReturnsNil(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	prevStageID := stages[0].ID
	advanced := campaignWithStages(id, domain.RolloutCampaignsStatusRunning, stages)
	decision := advanceDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().AdvanceStage(mock.Anything, id, prevStageID).Return(advanced, nil)
	m.stageCache.EXPECT().DeleteStageStats(mock.Anything, prevStageID).Return(someCampaignErr())
	m.cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CheckinData{StageID: stages[0].ID, TargetPercent: stages[0].TargetPercent}).Return(someCampaignErr())
	m.cache.EXPECT().AddRunningCampaigns(mock.Anything, id).Return(someCampaignErr())
	m.stageCache.EXPECT().SetStageStats(mock.Anything, stages[0].ID, domain.StageStats{MinSampleSize: stages[0].MinSampleSize, SuccessThreshold: stages[0].SuccessThreshold}).Return(someCampaignErr())

	err := m.service().ApplyDecision(context.Background(), decision)
	require.NoError(t, err)
}

func TestApplyDecision_DuplicateDecision_NoOp(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	decision := advanceDecision(uuid.New(), uuid.New())
	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{DecisionID: decision.DecisionID}, nil)

	err := m.service().ApplyDecision(context.Background(), decision)
	require.NoError(t, err)
}

func TestApplyDecision_GetFails_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	decision := advanceDecision(uuid.New(), uuid.New())
	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, someCampaignErr())

	err := m.service().ApplyDecision(context.Background(), decision)
	require.Error(t, err)
	assert.ErrorContains(t, err, "boom")
}

func TestApplyDecision_CreateFails_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	decision := advanceDecision(uuid.New(), uuid.New())
	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, someCampaignErr())

	err := m.service().ApplyDecision(context.Background(), decision)
	require.Error(t, err)
	assert.ErrorContains(t, err, "boom")
}

func TestApplyDecision_Advance_StaleStageWrongStatus_RecordsDecisionNoCache(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	prevStageID := uuid.New()
	decision := advanceDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().AdvanceStage(mock.Anything, id, prevStageID).Return(domain.RolloutCampaign{}, fmt.Errorf("%w: can't advance from passed stage", domain.ErrRolloutStageWrongStatus))

	err := m.service().ApplyDecision(context.Background(), decision)
	require.NoError(t, err)
}

func TestApplyDecision_Advance_StaleCampaignWrongStatus_RecordsDecision(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	prevStageID := uuid.New()
	decision := advanceDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusCompleted, nil), nil)

	err := m.service().ApplyDecision(context.Background(), decision)
	require.NoError(t, err)
}

// --- Rollback ---

func TestApplyDecision_Rollback_Success_DeletesCache(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	prevStageID := uuid.New()
	rolled := campaignWithStages(id, domain.RolloutCampaignsStatusRolledBack, nil)
	decision := rollbackDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().Rollback(mock.Anything, id, prevStageID).Return(rolled, nil)
	m.cache.EXPECT().RemoveRunningCampaigns(mock.Anything, id).Return(nil)
	m.cache.EXPECT().DeleteCheckinData(mock.Anything, id).Return(nil)
	m.stageCache.EXPECT().DeleteStageStats(mock.Anything, prevStageID).Return(nil)

	err := m.service().ApplyDecision(context.Background(), decision)
	require.NoError(t, err)
}

func TestApplyDecision_Rollback_StaleWrongStatus_RecordsDecisionNoCache(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	prevStageID := uuid.New()
	decision := rollbackDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().Rollback(mock.Anything, id, prevStageID).Return(domain.RolloutCampaign{}, fmt.Errorf("%w: can't rollback from passed stage", domain.ErrRolloutStageWrongStatus))

	err := m.service().ApplyDecision(context.Background(), decision)
	require.NoError(t, err)
}

func TestApplyDecision_Rollback_RepoFails_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	prevStageID := uuid.New()
	decision := rollbackDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().Rollback(mock.Anything, id, prevStageID).Return(domain.RolloutCampaign{}, someCampaignErr())

	err := m.service().ApplyDecision(context.Background(), decision)
	require.Error(t, err)
}

func TestApplyDecision_Rollback_CampaignNotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	decision := rollbackDecision(id, uuid.New())

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	err := m.service().ApplyDecision(context.Background(), decision)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestApplyDecision_Rollback_WrongCampaignStatus_NoOp(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	prevStageID := uuid.New()
	decision := rollbackDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusPaused, nil), nil)

	err := m.service().ApplyDecision(context.Background(), decision)
	require.NoError(t, err)
}

func TestApplyDecision_Rollback_CacheFails_StillReturnsNil(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	prevStageID := uuid.New()
	rolled := campaignWithStages(id, domain.RolloutCampaignsStatusRolledBack, nil)
	decision := rollbackDecision(id, prevStageID)

	m.decisionRepo.EXPECT().Get(mock.Anything, decision.DecisionID).Return(domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound)
	m.decisionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.AppliedDecision{}, nil)
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.campaignRepo.EXPECT().Rollback(mock.Anything, id, prevStageID).Return(rolled, nil)
	m.cache.EXPECT().RemoveRunningCampaigns(mock.Anything, id).Return(someCampaignErr())
	m.cache.EXPECT().DeleteCheckinData(mock.Anything, id).Return(someCampaignErr())
	m.stageCache.EXPECT().DeleteStageStats(mock.Anything, prevStageID).Return(someCampaignErr())

	err := m.service().ApplyDecision(context.Background(), decision)
	require.NoError(t, err)
}

// --- Get ---

func TestGet_CampaignNotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	_, err := m.service().Get(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestGet_ControllerFails_ReturnsCampaignWithoutStats(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().Get(mock.Anything, id).Return(campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil), nil)
	m.controller.EXPECT().GetCampaignStats(mock.Anything, id).Return(domain.RolloutCampaignStats{}, someCampaignErr())

	result, err := m.service().Get(context.Background(), id)
	require.NoError(t, err)
	assert.Nil(t, result.Stats)
}

func TestGet_Success_ReturnsCampaignWithStats(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().List(mock.Anything).Return([]domain.RolloutCampaign{campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil)}, nil)

	result, err := m.service().List(context.Background())
	require.NoError(t, err)
	require.Len(t, result, 1)
}

// --- WarmUpCache ---

func TestWarmUpCache_ListRunningFails_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	m.campaignRepo.EXPECT().ListRunning(mock.Anything).Return(nil, someCampaignErr())

	err := m.service().WarmUpCache(context.Background(), zap.NewNop().Sugar())
	require.Error(t, err)
}

func TestWarmUpCache_FindActiveStagesFails_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	m.campaignRepo.EXPECT().ListRunning(mock.Anything).Return([]domain.RolloutCampaign{campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil)}, nil)
	m.campaignRepo.EXPECT().FindActiveStages(mock.Anything, mock.Anything).Return(nil, someCampaignErr())

	err := m.service().WarmUpCache(context.Background(), zap.NewNop().Sugar())
	require.Error(t, err)
}

func TestWarmUpCache_Success_SetsCacheForAllActiveStages(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	m.campaignRepo.EXPECT().ListRunning(mock.Anything).Return([]domain.RolloutCampaign{campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil)}, nil)
	m.campaignRepo.EXPECT().FindActiveStages(mock.Anything, mock.Anything).Return(stages, nil)
	m.cache.EXPECT().DeleteAllRunningCampaigns(mock.Anything).Return(nil)
	m.cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CheckinData{StageID: stages[0].ID, TargetPercent: stages[0].TargetPercent}).Return(nil)
	m.cache.EXPECT().AddRunningCampaigns(mock.Anything, id).Return(nil)
	m.stageCache.EXPECT().SetStageStats(mock.Anything, stages[0].ID, domain.StageStats{MinSampleSize: stages[0].MinSampleSize, SuccessThreshold: stages[0].SuccessThreshold}).Return(nil)
	m.cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CheckinData{StageID: stages[1].ID, TargetPercent: stages[1].TargetPercent}).Return(nil)
	m.cache.EXPECT().AddRunningCampaigns(mock.Anything, id).Return(nil)
	m.stageCache.EXPECT().SetStageStats(mock.Anything, stages[1].ID, domain.StageStats{MinSampleSize: stages[1].MinSampleSize, SuccessThreshold: stages[1].SuccessThreshold}).Return(nil)

	err := m.service().WarmUpCache(context.Background(), zap.NewNop().Sugar())
	require.NoError(t, err)
}

func TestWarmUpCache_PartialCacheFailure_ContinuesProcessing(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	id := uuid.New()
	stages := twoStages(id)
	m.campaignRepo.EXPECT().ListRunning(mock.Anything).Return([]domain.RolloutCampaign{campaignWithStages(id, domain.RolloutCampaignsStatusRunning, nil)}, nil)
	m.campaignRepo.EXPECT().FindActiveStages(mock.Anything, mock.Anything).Return(stages, nil)
	m.cache.EXPECT().DeleteAllRunningCampaigns(mock.Anything).Return(nil)
	m.cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CheckinData{StageID: stages[0].ID, TargetPercent: stages[0].TargetPercent}).Return(someCampaignErr())
	m.cache.EXPECT().AddRunningCampaigns(mock.Anything, id).Return(someCampaignErr())
	m.stageCache.EXPECT().SetStageStats(mock.Anything, stages[0].ID, domain.StageStats{MinSampleSize: stages[0].MinSampleSize, SuccessThreshold: stages[0].SuccessThreshold}).Return(someCampaignErr())
	m.cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CheckinData{StageID: stages[1].ID, TargetPercent: stages[1].TargetPercent}).Return(nil)
	m.cache.EXPECT().AddRunningCampaigns(mock.Anything, id).Return(nil)
	m.stageCache.EXPECT().SetStageStats(mock.Anything, stages[1].ID, domain.StageStats{MinSampleSize: stages[1].MinSampleSize, SuccessThreshold: stages[1].SuccessThreshold}).Return(nil)

	err := m.service().WarmUpCache(context.Background(), zap.NewNop().Sugar())
	require.Error(t, err)
	assert.ErrorContains(t, err, "boom")
}

func TestWarmUpCache_NoRunningCampaigns_NoCacheWrites(t *testing.T) {
	t.Parallel()
	m := newCampaignMocks(t)
	m.campaignRepo.EXPECT().ListRunning(mock.Anything).Return([]domain.RolloutCampaign{}, nil)
	m.campaignRepo.EXPECT().FindActiveStages(mock.Anything, mock.Anything).Return(nil, nil)
	m.cache.EXPECT().DeleteAllRunningCampaigns(mock.Anything).Return(nil)

	err := m.service().WarmUpCache(context.Background(), zap.NewNop().Sugar())
	require.NoError(t, err)
}

func someCampaignErr() error {
	return errors.New("boom")
}
