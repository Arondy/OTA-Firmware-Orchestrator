package domain

import (
	"time"

	"github.com/google/uuid"
)

type CheckinEvent struct {
	DeviceID       uuid.UUID `json:"device_id"`
	DeviceModel    string    `json:"device_model"`
	CurrentVersion string    `json:"current_version"`
	CampaignID     uuid.UUID `json:"campaign_id"`
	Timestamp      time.Time `json:"timestamp"`
}

type UpdateResultsEvent struct {
	EventID    uuid.UUID            `json:"event_id"`
	DeviceID   uuid.UUID            `json:"device_id"`
	CampaignID uuid.UUID            `json:"campaign_id"`
	StageID    uuid.UUID            `json:"stage_id"`
	Result     UpdateAttemptsResult `json:"result"`
	Timestamp  time.Time            `json:"timestamp"`
}

func UpdateResultsEventFromAttempt(attempt UpdateAttempt) UpdateResultsEvent {
	return UpdateResultsEvent{
		EventID:    attempt.EventID,
		DeviceID:   attempt.DeviceID,
		CampaignID: attempt.CampaignID,
		StageID:    attempt.StageID,
		Result:     attempt.Result,
		Timestamp:  attempt.ReportedAt,
	}
}
