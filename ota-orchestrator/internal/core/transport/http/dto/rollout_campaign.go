package dto

import (
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/google/uuid"
)

// Stage не самостоятелен, только в составе Campaign
type RolloutStageRequest struct {
	OrderIndex       int     `json:"order_index" validate:"min=0"`
	TargetPercent    int     `json:"target_percent" validate:"required,min=1,max=100"`
	MinSampleSize    int     `json:"min_sample_size" validate:"required,min=1"`
	SuccessThreshold float32 `json:"success_threshold" validate:"required,gt=0,max=1"`
}

func (r RolloutStageRequest) ToDomain() domain.RolloutStage {
	return domain.RolloutStage{
		OrderIndex:       r.OrderIndex,
		TargetPercent:    r.TargetPercent,
		MinSampleSize:    r.MinSampleSize,
		SuccessThreshold: r.SuccessThreshold,
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

type RolloutCampaignResponse struct {
	RolloutCampaignListItemResponse
	RolloutStages []RolloutStageResponse `json:"rollout_stages"`
}

func RolloutCampaignFromDomain(rc domain.RolloutCampaign) RolloutCampaignResponse {
	rolloutStages := make([]RolloutStageResponse, len(rc.RolloutStages))
	for i, stage := range rc.RolloutStages {
		rolloutStages[i] = RolloutStageFromDomain(stage)
	}

	return RolloutCampaignResponse{
		RolloutCampaignListItemResponse: RolloutCampaignListItemFromDomain(rc),
		RolloutStages:                   rolloutStages,
	}
}

type ListRolloutCampaignsResponse struct {
	RolloutCampaigns []RolloutCampaignListItemResponse `json:"rollout_campaigns"`
}

type CreateRolloutCampaignRequest struct {
	FirmwareVersionID uuid.UUID             `json:"firmware_version_id" validate:"required"`
	RolloutStages     []RolloutStageRequest `json:"rollout_stages" validate:"required,max=20,rollout_stages,dive"`
}

func (r CreateRolloutCampaignRequest) ToDomain() domain.RolloutCampaign {
	rolloutStages := make([]domain.RolloutStage, len(r.RolloutStages))
	for i, stage := range r.RolloutStages {
		rolloutStages[i] = stage.ToDomain()
	}

	return domain.RolloutCampaign{
		FirmwareVersionID: r.FirmwareVersionID,
		RolloutStages:     rolloutStages,
	}
}
