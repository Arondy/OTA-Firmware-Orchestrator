package device

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"slices"

	"github.com/google/uuid"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/Masterminds/semver/v3"
)

type DeviceRepo interface {
	List(ctx context.Context) ([]domain.Device, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Device, error)
	Create(ctx context.Context, device domain.Device) (domain.Device, error)
	UpdateCheckinInfo(ctx context.Context, id uuid.UUID, version string) (domain.Device, error)
	Decommission(ctx context.Context, id uuid.UUID) (domain.Device, error)
}

type FirmwareVersionRepo interface {
	Get(ctx context.Context, id uuid.UUID) (domain.FirmwareVersion, error)
}

type RolloutCampaignRepo interface {
	Get(ctx context.Context, id uuid.UUID) (domain.RolloutCampaign, error)
	FindRunning(ctx context.Context, deviceModel string) (domain.RolloutCampaign, error)
	FindActiveStage(ctx context.Context, campaignID uuid.UUID) (domain.RolloutStage, error)
}

type UpdateAttemptRepo interface {
	Create(ctx context.Context, updateAttempt domain.UpdateAttempt) (domain.UpdateAttempt, error)
}

type DeviceService struct {
	deviceRepo        DeviceRepo
	firmwareRepo      FirmwareVersionRepo
	campaignRepo      RolloutCampaignRepo
	updateAttemptRepo UpdateAttemptRepo
}

func NewService(deviceRepo DeviceRepo, firmwareRepo FirmwareVersionRepo, campaignRepo RolloutCampaignRepo, updateAttemptRepo UpdateAttemptRepo) *DeviceService {
	return &DeviceService{
		deviceRepo:        deviceRepo,
		firmwareRepo:      firmwareRepo,
		campaignRepo:      campaignRepo,
		updateAttemptRepo: updateAttemptRepo,
	}
}

func (s *DeviceService) List(ctx context.Context) ([]domain.Device, error) {
	return s.deviceRepo.List(ctx)
}

func (s *DeviceService) Create(ctx context.Context, device domain.Device) (domain.Device, error) {
	return s.deviceRepo.Create(ctx, device)
}

func (s *DeviceService) Decommission(ctx context.Context, id uuid.UUID) (domain.Device, error) {
	return s.deviceRepo.Decommission(ctx, id)
}

type CheckinResult struct {
	UpdateAvailable bool
	StageID         *uuid.UUID
	BinaryUrl       string
	FWChecksum      string
}

func (s *DeviceService) Checkin(ctx context.Context, checkinDevice domain.Device) (CheckinResult, error) {
	device, err := s.deviceRepo.Get(ctx, checkinDevice.ID)
	if err != nil {
		return CheckinResult{}, err
	}

	if device.Status == domain.DeviceStatusDecommissioned {
		return CheckinResult{UpdateAvailable: false}, nil
	}

	device, err = s.deviceRepo.UpdateCheckinInfo(ctx, checkinDevice.ID, checkinDevice.CurrentVersion)
	if err != nil {
		return CheckinResult{}, err
	}

	campaign, err := s.campaignRepo.FindRunning(ctx, device.DeviceModel)
	if errors.Is(err, domain.ErrRolloutCampaignNotFound) || errors.Is(err, domain.ErrRolloutCampaignWrongStatus) {
		return CheckinResult{UpdateAvailable: false}, nil
	} else if err != nil {
		return CheckinResult{}, err
	}

	fw, err := s.firmwareRepo.Get(ctx, campaign.FirmwareVersionID)
	if err != nil {
		return CheckinResult{}, err
	}

	isGreater, err := isGreaterSemver(device.CurrentVersion, fw.FWVersion)
	if err != nil {
		return CheckinResult{}, err
	}
	if isGreater {
		return CheckinResult{UpdateAvailable: false}, nil
	}

	stage, err := s.campaignRepo.FindActiveStage(ctx, campaign.ID)
	if err != nil {
		return CheckinResult{}, err
	}

	bucket := s.calculateBucket(device.ID, campaign.ID)
	if bucket > uint32(stage.TargetPercent) {
		return CheckinResult{UpdateAvailable: false}, nil
	}

	return CheckinResult{
		UpdateAvailable: true,
		StageID:         &stage.ID,
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

func (s *DeviceService) calculateBucket(deviceID, campaignID uuid.UUID) uint32 {
	hash := fnv.New32a()
	hash.Reset()
	hash.Write(deviceID[:])
	hash.Write(campaignID[:])
	return (hash.Sum32() % 100) + 1
}

func (s *DeviceService) Report(ctx context.Context, updateAttempt domain.UpdateAttempt) (domain.UpdateAttempt, error) {
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

	return s.updateAttemptRepo.Create(ctx, updateAttempt)
}
