package domain

import "github.com/google/uuid"

type RolloutCampaignStats struct {
	ActiveStageID uuid.UUID
	SuccessRate   float32
	SampleSize    int
}

type StageStats struct {
	MinSampleSize    int     `redis:"min_sample_size"`
	SuccessThreshold float32 `redis:"success_threshold"`
}
