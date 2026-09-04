package domain

import (
	"time"

	"github.com/google/uuid"
)

type CheckinResult struct {
	UpdateAvailable bool
	StageID         *uuid.UUID
	BinaryUrl       string
	FWChecksum      string
}

type CampaignCheckinData struct {
	StageID       uuid.UUID `redis:"stage_id"`
	TargetPercent int       `redis:"target_percent"`
}

type DeviceCheckinData struct {
	CurrentVersion string    `redis:"current_version"`
	LastSeen       time.Time `redis:"last_seen"`
}
