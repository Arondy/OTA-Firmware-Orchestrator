package domain

import (
	"time"

	"github.com/google/uuid"
)

type UpdateResultsEvent struct {
	EventID    uuid.UUID            `json:"event_id"`
	DeviceID   uuid.UUID            `json:"device_id"`
	CampaignID uuid.UUID            `json:"campaign_id"`
	StageID    uuid.UUID            `json:"stage_id"`
	Result     UpdateAttemptsResult `json:"result"`
	Timestamp  time.Time            `json:"timestamp"`
}

type DecisionEvent struct {
	DecisionID      uuid.UUID    `json:"decision_id"`
	CampaignID      uuid.UUID    `json:"campaign_id"`
	DecisionType    DecisionType `json:"decision_type"`
	PreviousStageID uuid.UUID    `json:"previous_stage_id"`
	Timestamp       time.Time    `json:"timestamp"`
}
