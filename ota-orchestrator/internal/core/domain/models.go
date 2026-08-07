package domain

import (
	"time"

	"github.com/google/uuid"
)

type Device struct {
	ID             uuid.UUID
	DeviceModel    string
	CurrentVersion string
	Status         DeviceStatus
	LastSeen       *time.Time
	CreatedAt      time.Time
}

type FirmwareVersion struct {
	ID          uuid.UUID
	DeviceModel string
	FWVersion   string
	FWChecksum  string
	BinaryUrl   string
	CreatedAt   time.Time
}

type RolloutStage struct {
	ID               uuid.UUID
	CampaignID       uuid.UUID
	OrderIndex       int
	TargetPercent    int
	MinSampleSize    int
	SuccessThreshold float32
	Status           RolloutStagesStatus
	EnteredAt        *time.Time
}

type RolloutCampaign struct {
	ID                uuid.UUID
	FirmwareVersionID uuid.UUID
	DeviceModel       string
	Status            RolloutCampaignsStatus
	RolloutStages     []RolloutStage
	CreatedAt         time.Time
	StartedAt         *time.Time
	CompletedAt       *time.Time
}

type UpdateAttempt struct {
	ID         uuid.UUID
	DeviceID   uuid.UUID
	CampaignID uuid.UUID
	StageID    uuid.UUID
	Result     UpdateAttemptsResult
	ReportedAt time.Time
}
