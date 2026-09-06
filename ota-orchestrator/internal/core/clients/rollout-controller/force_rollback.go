package rollout_controller

import (
	"context"

	"connectrpc.com/connect"
	rolloutv1 "github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/rollout/v1"
	"github.com/google/uuid"
)

func (c *Client) ForceRollback(ctx context.Context, campaignID uuid.UUID) error {
	req := connect.NewRequest(&rolloutv1.ForceRollbackRequest{
		CampaignId: campaignID.String(),
	})

	_, err := c.campaignClient.ForceRollback(ctx, req)
	return err
}
