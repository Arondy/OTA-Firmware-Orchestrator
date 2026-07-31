package domain

import "errors"

var ErrDeviceNotFound = errors.New("device not found")
var ErrFirmwareVersionNotFound = errors.New("firmware version not found")
var ErrRolloutCampaignNotFound = errors.New("rollout campaign not found")

var ErrFirmwareVersionAlreadyExists = errors.New("this firmware version for this model already exists")
var ErrCampaignAlreadyRunning = errors.New("another campaign for this model is already running")
var ErrRolloutStageAlreadyExists = errors.New("rollout stage for this rollout campaign with such index already exists")
