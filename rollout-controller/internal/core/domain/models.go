package domain

import (
	"github.com/google/uuid"
)

type CampaignStats struct {
	ActiveStageID uuid.UUID
	SuccessRate   float32
	SampleSize    int
}
