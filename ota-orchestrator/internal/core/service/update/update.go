package update

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Masterminds/semver/v3"
)

type DeviceRepo interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Device, error)
}

type FirmwareVersionRepo interface {
	Get(ctx context.Context, id uuid.UUID) (domain.FirmwareVersion, error)
}

type RolloutCampaignRepo interface {
	Get(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	FindRunning(ctx context.Context, deviceModel string) (domain.RolloutCampaign, error)
}

type UpdateAttemptRepo interface {
	Create(ctx context.Context, updateAttempt domain.UpdateAttempt) (domain.UpdateAttempt, error)
}

type DeviceCacheRepo interface {
	SetCurrentVersion(ctx context.Context, id uuid.UUID, currentVersion string) error
	SetLastSeen(ctx context.Context, id uuid.UUID, lastSeen time.Time) error
}

type CampaignCacheRepo interface {
	GetCurrentStage(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
	GetCurrentTargetPercent(ctx context.Context, id uuid.UUID) (int, error)
}

type CheckinProducer interface {
	Produce(event domain.CheckinEvent)
}

type UpdateResultsProducer interface {
	Produce(event domain.UpdateResultsEvent) error
}

type UpdateService struct {
	deviceRepo            DeviceRepo
	firmwareRepo          FirmwareVersionRepo
	campaignRepo          RolloutCampaignRepo
	updateAttemptRepo     UpdateAttemptRepo
	deviceCacheRepo       DeviceCacheRepo
	campaignCacheRepo     CampaignCacheRepo
	checkinProducer       CheckinProducer
	updateResultsProducer UpdateResultsProducer
}

func NewService(deviceRepo DeviceRepo, firmwareRepo FirmwareVersionRepo, campaignRepo RolloutCampaignRepo, updateAttemptRepo UpdateAttemptRepo, deviceCacheRepo DeviceCacheRepo, campaignCacheRepo CampaignCacheRepo, checkinProducer CheckinProducer, updateResultsProducer UpdateResultsProducer) *UpdateService {
	return &UpdateService{
		deviceRepo:            deviceRepo,
		firmwareRepo:          firmwareRepo,
		campaignRepo:          campaignRepo,
		updateAttemptRepo:     updateAttemptRepo,
		deviceCacheRepo:       deviceCacheRepo,
		campaignCacheRepo:     campaignCacheRepo,
		checkinProducer:       checkinProducer,
		updateResultsProducer: updateResultsProducer,
	}
}

func (s *UpdateService) Checkin(ctx context.Context, checkinDevice domain.Device) (domain.CheckinResult, error) {
	device, err := s.deviceRepo.Get(ctx, checkinDevice.ID)
	if err != nil {
		return domain.CheckinResult{}, err
	}

	if device.Status == domain.DeviceStatusDecommissioned {
		return domain.CheckinResult{UpdateAvailable: false}, nil
	}

	logger := config.LoggerFromContext(ctx).With("device_id", checkinDevice.ID)

	err = s.deviceCacheRepo.SetCurrentVersion(ctx, checkinDevice.ID, checkinDevice.CurrentVersion)
	if err != nil {
		logger.Warnw("failed to set device current version", "error", err)
	}

	err = s.deviceCacheRepo.SetLastSeen(ctx, checkinDevice.ID, time.Now())
	if err != nil {
		logger.Warnw("failed to set device last seen", "error", err)
	}

	campaign, err := s.campaignRepo.FindRunning(ctx, device.DeviceModel)
	if errors.Is(err, domain.ErrRolloutCampaignNotFound) || errors.Is(err, domain.ErrRolloutCampaignWrongStatus) {
		return domain.CheckinResult{UpdateAvailable: false}, nil
	} else if err != nil {
		return domain.CheckinResult{}, err
	}

	s.checkinProducer.Produce(domain.CheckinEvent{
		DeviceID:       checkinDevice.ID,
		DeviceModel:    device.DeviceModel,
		CurrentVersion: checkinDevice.CurrentVersion,
		CampaignID:     campaign.ID,
		Timestamp:      time.Now(),
	})

	fw, err := s.firmwareRepo.Get(ctx, campaign.FirmwareVersionID)
	if err != nil {
		return domain.CheckinResult{}, err
	}

	isGreater, err := isGreaterSemver(checkinDevice.CurrentVersion, fw.FWVersion)
	if err != nil {
		return domain.CheckinResult{}, err
	}
	if isGreater {
		return domain.CheckinResult{UpdateAvailable: false}, nil
	}

	stageID, err := s.campaignCacheRepo.GetCurrentStage(ctx, campaign.ID)
	if err != nil {
		return domain.CheckinResult{}, err
	}

	targetPercent, err := s.campaignCacheRepo.GetCurrentTargetPercent(ctx, campaign.ID)
	if err != nil {
		return domain.CheckinResult{}, err
	}

	bucket := s.calculateBucket(device.ID, campaign.ID)
	if bucket > uint32(targetPercent) {
		return domain.CheckinResult{UpdateAvailable: false}, nil
	}

	return domain.CheckinResult{
		UpdateAvailable: true,
		StageID:         &stageID,
		BinaryUrl:       fw.BinaryUrl,
		FWChecksum:      fw.FWChecksum,
	}, nil
}

func isGreaterSemver(s1, s2 string) (bool, error) {
	semver1, err := semver.NewVersion(s1)
	if err != nil {
		return false, fmt.Errorf("parsing semver '%s' error: %w", s1, err)
	}

	semver2, err := semver.NewVersion(s2)
	if err != nil {
		return false, fmt.Errorf("parsing semver '%s' error: %w", s2, err)
	}

	return semver1.GreaterThanEqual(semver2), nil
}

func (s *UpdateService) calculateBucket(deviceID, campaignID uuid.UUID) uint32 {
	hash := fnv.New32a()
	hash.Reset()
	hash.Write(deviceID[:])
	hash.Write(campaignID[:])
	return (hash.Sum32() % 100) + 1
}

func (s *UpdateService) Report(ctx context.Context, updateAttempt domain.UpdateAttempt) (domain.UpdateAttempt, error) {
	campaign, err := s.campaignRepo.Get(ctx, updateAttempt.CampaignID)
	if err != nil {
		return domain.UpdateAttempt{}, err
	}

	hasStage := slices.ContainsFunc(campaign.RolloutStages, func(stage domain.RolloutStage) bool {
		return stage.ID == updateAttempt.StageID
	})
	if !hasStage {
		return domain.UpdateAttempt{}, domain.ErrRolloutStageNotFoundInCampaign
	}

	device, err := s.deviceRepo.Get(ctx, updateAttempt.DeviceID)
	if err != nil {
		return domain.UpdateAttempt{}, err
	}

	if device.DeviceModel != campaign.DeviceModel {
		return domain.UpdateAttempt{}, domain.ErrWrongDeviceModel
	}

	eventID, err := uuid.NewV7()
	if err != nil {
		return domain.UpdateAttempt{}, fmt.Errorf("failed to generate event_id: %w", err)
	}
	updateAttempt.EventID = eventID

	attempt, err := s.updateAttemptRepo.Create(ctx, updateAttempt)
	if err != nil {
		return domain.UpdateAttempt{}, err
	}

	updateResults := domain.UpdateResultsEventFromAttempt(attempt)
	err = s.updateResultsProducer.Produce(updateResults)
	if err != nil {
		return domain.UpdateAttempt{}, fmt.Errorf("%w: %w", domain.ErrUpdateResultNotProduced, err)
	}

	return attempt, err
}
