package handlers

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	rolloutv1 "github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/rollout/v1"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/google/uuid"
)

type CampaignStatsService interface {
	GetCampaignStats(ctx context.Context, id uuid.UUID) (domain.CampaignStats, error)
}

type CampaignStatsHandler struct {
	campaignStatsSvc CampaignStatsService
}

func NewCampaignStatsHandler(campaignStatsSvc CampaignStatsService) *CampaignStatsHandler {
	return &CampaignStatsHandler{
		campaignStatsSvc: campaignStatsSvc,
	}
}

func (h *CampaignStatsHandler) GetCampaignStats(ctx context.Context, req *connect.Request[rolloutv1.GetCampaignStatsRequest]) (*connect.Response[rolloutv1.GetCampaignStatsResponse], error) {
	campaignID := req.Msg.GetCampaignId()
	strID, err := uuid.Parse(campaignID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	status, err := h.campaignStatsSvc.GetCampaignStats(ctx, strID)
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
