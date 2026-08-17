package rollout_controller

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	rolloutv1 "github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/rollout/v1"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
)

func (c *Client) GetCampaignStats(ctx context.Context, id uuid.UUID) (domain.RolloutCampaignStats, error) {
	req := connect.NewRequest(&rolloutv1.GetCampaignStatsRequest{
		CampaignId: id.String(),
	})

	resp, err := c.campaignClient.GetCampaignStats(ctx, req)
	if err != nil {
		return domain.RolloutCampaignStats{}, err
	}

	return CampaignStatsToDomain(resp.Msg)
}

func CampaignStatsToDomain(status *rolloutv1.GetCampaignStatsResponse) (domain.RolloutCampaignStats, error) {
	idStr := status.GetActiveStageId()
	id, err := uuid.Parse(idStr)
	if err != nil {
		return domain.RolloutCampaignStats{}, fmt.Errorf("failed to parse UUID %s: %w", idStr, err)
	}

	return domain.RolloutCampaignStats{
		ActiveStageID: id,
		SuccessRate:   status.GetSuccessRate(),
		SampleSize:    int(status.GetSampleSize()),
	}, nil
}
