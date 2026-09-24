package handlers

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	rolloutv1 "github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/rollout/v1"
	"github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/rollout/v1/rolloutv1connect"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/google/uuid"
)

type CampaignStatsService interface {
	GetCampaignStats(ctx context.Context, id uuid.UUID) (domain.CampaignStats, error)
}

type DecisionsService interface {
	SendDecision(ctx context.Context, campaignID uuid.UUID, decision domain.DecisionType) error
}

type CampaignHandler struct {
	rolloutv1connect.UnimplementedCampaignServiceHandler

	statsSvc     CampaignStatsService
	decisionsSvc DecisionsService
}

func NewCampaignHandler(statsSvc CampaignStatsService, decisionsSvc DecisionsService) *CampaignHandler {
	return &CampaignHandler{
		statsSvc:     statsSvc,
		decisionsSvc: decisionsSvc,
	}
}

func (h *CampaignHandler) GetCampaignStats(ctx context.Context, req *connect.Request[rolloutv1.GetCampaignStatsRequest]) (*connect.Response[rolloutv1.GetCampaignStatsResponse], error) {
	campaignIDStr := req.Msg.GetCampaignId()
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	status, err := h.statsSvc.GetCampaignStats(ctx, campaignID)
	if errors.Is(err, domain.ErrCurrentStageNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	} else if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := GetCampaignStatsResponseFromDomain(status)
	return connect.NewResponse(resp), nil
}

func GetCampaignStatsResponseFromDomain(status domain.CampaignStats) *rolloutv1.GetCampaignStatsResponse {
	return &rolloutv1.GetCampaignStatsResponse{
		ActiveStageId: status.ActiveStageID.String(),
		SuccessRate:   status.SuccessRate,
		SampleSize:    int32(status.SampleSize),
	}
}

func (h *CampaignHandler) ForceRollback(ctx context.Context, req *connect.Request[rolloutv1.ForceRollbackRequest]) (*connect.Response[rolloutv1.ForceRollbackResponse], error) {
	campaignIDStr := req.Msg.GetCampaignId()
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = h.decisionsSvc.SendDecision(ctx, campaignID, domain.DecisionTypeRollback)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &rolloutv1.ForceRollbackResponse{}
	return connect.NewResponse(resp), nil
}
