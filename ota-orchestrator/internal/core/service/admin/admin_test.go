package admin_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/admin"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/admin/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newService(t *testing.T) (*admin.AdminService, *mocks.MockRolloutCampaignRepo, *mocks.MockCampaignCacheRepo, *mocks.MockRolloutController) {
	t.Helper()
	repo := mocks.NewMockRolloutCampaignRepo(t)
	cache := mocks.NewMockCampaignCacheRepo(t)
	controller := mocks.NewMockRolloutController(t)
	return admin.NewAdminService(repo, cache, controller), repo, cache, controller
}

func pausedCampaignWithActiveStage(id uuid.UUID) (domain.RolloutCampaign, domain.RolloutStage) {
	stageID := uuid.New()
	stage := domain.RolloutStage{
		ID:            stageID,
		CampaignID:    id,
		OrderIndex:    0,
		TargetPercent: 10,
		Status:        domain.RolloutStagesStatusActive,
	}
	campaign := domain.RolloutCampaign{
		ID:            id,
		Status:        domain.RolloutCampaignsStatusPaused,
		RolloutStages: []domain.RolloutStage{stage},
	}
	return campaign, stage
}

func TestForceRollback_CampaignNotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	svc, repo, _, _ := newService(t)
	id := uuid.New()
	repo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	err := svc.ForceRollback(context.Background(), id)
	assert.ErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
}

func TestForceRollback_WrongStatus_ReturnsWrongStatus(t *testing.T) {
	t.Parallel()
	statuses := []domain.RolloutCampaignsStatus{
		domain.RolloutCampaignsStatusDraft,
		domain.RolloutCampaignsStatusCompleted,
		domain.RolloutCampaignsStatusRolledBack,
	}
	for _, status := range statuses {
		svc, repo, _, _ := newService(t)
		id := uuid.New()
		repo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{ID: id, Status: status}, nil)

		err := svc.ForceRollback(context.Background(), id)
		assert.ErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus, "status %s", status)
	}
}

func TestForceRollback_ControllerFails_ReturnsError(t *testing.T) {
	t.Parallel()
	svc, repo, _, controller := newService(t)
	id := uuid.New()
	repo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{ID: id, Status: domain.RolloutCampaignsStatusRunning}, nil)
	controller.EXPECT().ForceRollback(mock.Anything, id).Return(errors.New("boom"))

	err := svc.ForceRollback(context.Background(), id)
	assert.Error(t, err)
}

func TestForceRollback_Running_DelegatesWithoutCache(t *testing.T) {
	t.Parallel()
	svc, repo, _, controller := newService(t)
	id := uuid.New()
	repo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{ID: id, Status: domain.RolloutCampaignsStatusRunning}, nil)
	controller.EXPECT().ForceRollback(mock.Anything, id).Return(nil)

	err := svc.ForceRollback(context.Background(), id)
	assert.NoError(t, err)
}

func TestForceRollback_Paused_RestoresCacheAndDelegates(t *testing.T) {
	t.Parallel()
	svc, repo, cache, controller := newService(t)
	id := uuid.New()
	campaign, stage := pausedCampaignWithActiveStage(id)
	repo.EXPECT().Get(mock.Anything, id).Return(campaign, nil)
	cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CampaignCheckinData{
		StageID:       stage.ID,
		TargetPercent: stage.TargetPercent,
	}).Return(nil)
	controller.EXPECT().ForceRollback(mock.Anything, id).Return(nil)

	err := svc.ForceRollback(context.Background(), id)
	assert.NoError(t, err)
}

func TestForceRollback_Paused_NoActiveStage_ReturnsError(t *testing.T) {
	t.Parallel()
	svc, repo, _, _ := newService(t)
	id := uuid.New()
	repo.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{ID: id, Status: domain.RolloutCampaignsStatusPaused}, nil)

	err := svc.ForceRollback(context.Background(), id)
	assert.Error(t, err)
}

func TestForceRollback_Paused_CacheFails_ReturnsError(t *testing.T) {
	t.Parallel()
	svc, repo, cache, _ := newService(t)
	id := uuid.New()
	campaign, stage := pausedCampaignWithActiveStage(id)
	repo.EXPECT().Get(mock.Anything, id).Return(campaign, nil)
	cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CampaignCheckinData{
		StageID:       stage.ID,
		TargetPercent: stage.TargetPercent,
	}).Return(errors.New("boom"))

	err := svc.ForceRollback(context.Background(), id)
	assert.Error(t, err)
}

func TestForceRollback_Paused_ControllerFails_ReturnsError(t *testing.T) {
	t.Parallel()
	svc, repo, cache, controller := newService(t)
	id := uuid.New()
	campaign, stage := pausedCampaignWithActiveStage(id)
	repo.EXPECT().Get(mock.Anything, id).Return(campaign, nil)
	cache.EXPECT().SetCheckinData(mock.Anything, id, domain.CampaignCheckinData{
		StageID:       stage.ID,
		TargetPercent: stage.TargetPercent,
	}).Return(nil)
	controller.EXPECT().ForceRollback(mock.Anything, id).Return(errors.New("boom"))

	err := svc.ForceRollback(context.Background(), id)
	assert.Error(t, err)
}
