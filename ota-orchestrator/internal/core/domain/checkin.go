package domain

import "github.com/google/uuid"

type CheckinResult struct {
	UpdateAvailable bool
	StageID         *uuid.UUID
	BinaryUrl       string
	FWChecksum      string
}

type CheckinData struct {
	StageID       uuid.UUID
	TargetPercent int
}
