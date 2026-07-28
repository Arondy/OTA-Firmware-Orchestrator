package domain

import "errors"

var ErrDeviceNotFound = errors.New("device not found")
var ErrFirmwareVersionNotFound = errors.New("firmware version not found")
var ErrRolloutCampaignNotFound = errors.New("rollout campaign not found")

var ErrCampaignAlreadyRunning = errors.New("another campaign for this model is already running")
