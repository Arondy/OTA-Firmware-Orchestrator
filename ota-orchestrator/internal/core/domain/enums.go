package domain

type DeviceStatus string

const (
	DeviceStatusActive         DeviceStatus = "active"
	DeviceStatusDecommissioned DeviceStatus = "decommissioned"
)

func (s DeviceStatus) IsValid() bool {
	switch s {
	case DeviceStatusActive, DeviceStatusDecommissioned:
		return true
	}
	return false
}

type RolloutCampaignsStatus string

const (
	RolloutCampaignsStatusDraft      RolloutCampaignsStatus = "draft"
	RolloutCampaignsStatusRunning    RolloutCampaignsStatus = "running"
	RolloutCampaignsStatusPaused     RolloutCampaignsStatus = "paused"
	RolloutCampaignsStatusCompleted  RolloutCampaignsStatus = "completed"
	RolloutCampaignsStatusRolledBack RolloutCampaignsStatus = "rolled_back"
)

func (s RolloutCampaignsStatus) IsValid() bool {
	switch s {
	case RolloutCampaignsStatusDraft, RolloutCampaignsStatusRunning,
		RolloutCampaignsStatusPaused, RolloutCampaignsStatusCompleted,
		RolloutCampaignsStatusRolledBack:
		return true
	}
	return false
}

type RolloutStagesStatus string

const (
	RolloutStagesStatusPending RolloutStagesStatus = "pending"
	RolloutStagesStatusActive  RolloutStagesStatus = "active"
	RolloutStagesStatusPassed  RolloutStagesStatus = "passed"
	RolloutStagesStatusFailed  RolloutStagesStatus = "failed"
)

func (s RolloutStagesStatus) IsValid() bool {
	switch s {
	case RolloutStagesStatusPending, RolloutStagesStatusActive,
		RolloutStagesStatusPassed, RolloutStagesStatusFailed:
		return true
	}
	return false
}

type UpdateAttemptsResult string

const (
	UpdateAttemptsResultSuccess UpdateAttemptsResult = "success"
	UpdateAttemptsResultFailure UpdateAttemptsResult = "failure"
	UpdateAttemptsResultTimeout UpdateAttemptsResult = "timeout"
)

func (s UpdateAttemptsResult) IsValid() bool {
	switch s {
	case UpdateAttemptsResultSuccess, UpdateAttemptsResultFailure,
		UpdateAttemptsResultTimeout:
		return true
	}
	return false
}
