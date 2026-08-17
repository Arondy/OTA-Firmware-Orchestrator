package domain

import "github.com/google/uuid"

type RolloutCampaignStats struct {
	ActiveStageID uuid.UUID
	SuccessRate   float32
	SampleSize    int
}
