package handlers

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	rolloutv1 "github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/rollout/v1"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/transport/connect/handlers/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newCampaignHandler(t *testing.T) (*CampaignHandler, *mocks.MockCampaignStatsService, *mocks.MockDecisionsService) {
	t.Helper()
	statsSvc := mocks.NewMockCampaignStatsService(t)
	decisionsSvc := mocks.NewMockDecisionsService(t)
	return NewCampaignHandler(statsSvc, decisionsSvc), statsSvc, decisionsSvc
}

func TestGetCampaignStats_InvalidUUID_ReturnsInvalidArgument(t *testing.T) {
	t.Parallel()
	h, _, _ := newCampaignHandler(t)

	req := connect.NewRequest(&rolloutv1.GetCampaignStatsRequest{CampaignId: "not-a-uuid"})
	_, err := h.GetCampaignStats(context.Background(), req)
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestGetCampaignStats_CurrentStageNotFound_ReturnsNotFound(t *testing.T) {
	t.Parallel()
	h, svc, _ := newCampaignHandler(t)
	id := uuid.New()
	svc.EXPECT().GetCampaignStats(mock.Anything, id).Return(domain.CampaignStats{}, domain.ErrCurrentStageNotFound)

	req := connect.NewRequest(&rolloutv1.GetCampaignStatsRequest{CampaignId: id.String()})
	_, err := h.GetCampaignStats(context.Background(), req)
	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestGetCampaignStats_InternalError_ReturnsInternal(t *testing.T) {
	t.Parallel()
	h, svc, _ := newCampaignHandler(t)
	id := uuid.New()
	svc.EXPECT().GetCampaignStats(mock.Anything, id).Return(domain.CampaignStats{}, assertStatsErr())

	req := connect.NewRequest(&rolloutv1.GetCampaignStatsRequest{CampaignId: id.String()})
	_, err := h.GetCampaignStats(context.Background(), req)
	require.Error(t, err)
	assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestGetCampaignStats_Success_ReturnsStats(t *testing.T) {
	t.Parallel()
	h, svc, _ := newCampaignHandler(t)
	id := uuid.New()
	stats := domain.CampaignStats{ActiveStageID: uuid.New(), SuccessRate: 0.5, SampleSize: 2}
	svc.EXPECT().GetCampaignStats(mock.Anything, id).Return(stats, nil)

	req := connect.NewRequest(&rolloutv1.GetCampaignStatsRequest{CampaignId: id.String()})
	resp, err := h.GetCampaignStats(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, stats.SampleSize, int(resp.Msg.SampleSize))
}

func assertStatsErr() error {
	return errors.New("boom")
}
