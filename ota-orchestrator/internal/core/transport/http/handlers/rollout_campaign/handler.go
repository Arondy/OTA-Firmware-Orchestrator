package rollout_campaign

import (
	"context"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
)

type RolloutCampaignService interface {
	List(ctx context.Context) ([]domain.RolloutCampaign, error)
	Get(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Create(ctx context.Context, campaign domain.RolloutCampaign) (domain.RolloutCampaign, error)
	Start(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Pause(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	Resume(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	AdvanceStage(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
}

type RolloutCampaignHandler struct {
	svc RolloutCampaignService
}

func NewRolloutCampaignHandler(svc RolloutCampaignService) *RolloutCampaignHandler {
	return &RolloutCampaignHandler{
		svc: svc,
	}
}

type RolloutCampaignListItemResponse struct {
	ID                uuid.UUID                     `json:"id"`
	FirmwareVersionID uuid.UUID                     `json:"firmware_version_id"`
	DeviceModel       string                        `json:"device_model"`
	Status            domain.RolloutCampaignsStatus `json:"status"`
	CreatedAt         time.Time                     `json:"created_at"`
	StartedAt         *time.Time                    `json:"started_at,omitempty"`
	CompletedAt       *time.Time                    `json:"completed_at,omitempty"`
}

func RolloutCampaignListItemFromDomain(rc domain.RolloutCampaign) RolloutCampaignListItemResponse {
	return RolloutCampaignListItemResponse{
		ID:                rc.ID,
		FirmwareVersionID: rc.FirmwareVersionID,
		DeviceModel:       rc.DeviceModel,
		Status:            rc.Status,
		CreatedAt:         rc.CreatedAt,
		StartedAt:         rc.StartedAt,
		CompletedAt:       rc.CompletedAt,
	}
}

type RolloutStageResponse struct {
	ID               uuid.UUID                  `json:"id"`
	CampaignID       uuid.UUID                  `json:"campaign_id"`
	OrderIndex       int                        `json:"order_index"`
	TargetPercent    int                        `json:"target_percent"`
	MinSampleSize    int                        `json:"min_sample_size"`
	SuccessThreshold float32                    `json:"success_threshold"`
	Status           domain.RolloutStagesStatus `json:"status"`
	EnteredAt        *time.Time                 `json:"entered_at,omitempty"`
}

func RolloutStageFromDomain(rs domain.RolloutStage) RolloutStageResponse {
	return RolloutStageResponse{
		ID:               rs.ID,
		CampaignID:       rs.CampaignID,
		OrderIndex:       rs.OrderIndex,
		TargetPercent:    rs.TargetPercent,
		MinSampleSize:    rs.MinSampleSize,
		SuccessThreshold: rs.SuccessThreshold,
		Status:           rs.Status,
		EnteredAt:        rs.EnteredAt,
	}
}

type RolloutCampaignStats struct {
	ActiveStageID uuid.UUID `json:"active_stage_id"`
	SuccessRate   float32   `json:"success_rate"`
	SampleSize    int       `json:"sample_size"`
}

func RolloutCampaignStatsFromDomain(s *domain.RolloutCampaignStats) *RolloutCampaignStats {
	if s == nil {
		return nil
	}
	return &RolloutCampaignStats{
		ActiveStageID: s.ActiveStageID,
		SuccessRate:   s.SuccessRate,
		SampleSize:    s.SampleSize,
	}
}

type RolloutCampaignResponse struct {
	RolloutCampaignListItemResponse
	RolloutStages []RolloutStageResponse `json:"rollout_stages"`
	Stats         *RolloutCampaignStats  `json:"stats,omitempty"`
}

func RolloutCampaignFromDomain(rc domain.RolloutCampaign) RolloutCampaignResponse {
	rolloutStages := make([]RolloutStageResponse, len(rc.RolloutStages))
	for i, stage := range rc.RolloutStages {
		rolloutStages[i] = RolloutStageFromDomain(stage)
	}

	return RolloutCampaignResponse{
		RolloutCampaignListItemResponse: RolloutCampaignListItemFromDomain(rc),
		RolloutStages:                   rolloutStages,
		Stats:                           RolloutCampaignStatsFromDomain(rc.Stats),
	}
}
