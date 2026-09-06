package handlers

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	rolloutv1 "github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/rollout/v1"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestForceRollback_InvalidUUID_ReturnsInvalidArgument(t *testing.T) {
	t.Parallel()
	h, _, _ := newCampaignHandler(t)

	req := connect.NewRequest(&rolloutv1.ForceRollbackRequest{CampaignId: "not-a-uuid"})
	_, err := h.ForceRollback(context.Background(), req)
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestForceRollback_SendDecisionFails_ReturnsInternal(t *testing.T) {
	t.Parallel()
	h, _, decisionsSvc := newCampaignHandler(t)
	id := uuid.New()
	decisionsSvc.EXPECT().SendDecision(mock.Anything, id, domain.DecisionTypeRollback).Return(errors.New("boom"))

	req := connect.NewRequest(&rolloutv1.ForceRollbackRequest{CampaignId: id.String()})
	_, err := h.ForceRollback(context.Background(), req)
	require.Error(t, err)
	assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestForceRollback_Success_PublishesRollback(t *testing.T) {
	t.Parallel()
	h, _, decisionsSvc := newCampaignHandler(t)
	id := uuid.New()
	decisionsSvc.EXPECT().SendDecision(mock.Anything, id, domain.DecisionTypeRollback).Return(nil)

	req := connect.NewRequest(&rolloutv1.ForceRollbackRequest{CampaignId: id.String()})
	resp, err := h.ForceRollback(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
}
