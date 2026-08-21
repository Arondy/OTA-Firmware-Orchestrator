package update_test

import (
	"context"
	"errors"
	"hash/fnv"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/update"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/service/update/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func computeBucket(deviceID, campaignID uuid.UUID) uint32 {
	h := fnv.New32a()
	h.Reset()
	h.Write(deviceID[:])
	h.Write(campaignID[:])
	return (h.Sum32() % 100) + 1
}

func someErr() error {
	return errors.New("boom")
}

type checkinMocks struct {
	deviceRepo        *mocks.MockDeviceRepo
	firmwareRepo      *mocks.MockFirmwareVersionRepo
	campaignRepo      *mocks.MockRolloutCampaignRepo
	updateAttemptRepo *mocks.MockUpdateAttemptRepo
	deviceCacheRepo   *mocks.MockDeviceCacheRepo
	campaignCacheRepo *mocks.MockCampaignCacheRepo
	checkinProducer   *mocks.MockCheckinProducer
	updateResultsProd *mocks.MockUpdateResultsProducer
}

func newCheckinMocks(t *testing.T) *checkinMocks {
	return &checkinMocks{
		deviceRepo:        mocks.NewMockDeviceRepo(t),
		firmwareRepo:      mocks.NewMockFirmwareVersionRepo(t),
		campaignRepo:      mocks.NewMockRolloutCampaignRepo(t),
		updateAttemptRepo: mocks.NewMockUpdateAttemptRepo(t),
		deviceCacheRepo:   mocks.NewMockDeviceCacheRepo(t),
		campaignCacheRepo: mocks.NewMockCampaignCacheRepo(t),
		checkinProducer:   mocks.NewMockCheckinProducer(t),
		updateResultsProd: mocks.NewMockUpdateResultsProducer(t),
	}
}

func (m *checkinMocks) service() *update.UpdateService {
	return update.NewService(
		m.deviceRepo,
		m.firmwareRepo,
		m.campaignRepo,
		m.updateAttemptRepo,
		m.deviceCacheRepo,
		m.campaignCacheRepo,
		m.checkinProducer,
		m.updateResultsProd,
	)
}

func activeDevice(id uuid.UUID) domain.Device {
	return domain.Device{
		ID:             id,
		DeviceModel:    "model-a",
		CurrentVersion: "1.0.0",
		Status:         domain.DeviceStatusActive,
	}
}

func runningCampaign(id uuid.UUID, fwID uuid.UUID) domain.RolloutCampaign {
	return domain.RolloutCampaign{
		ID:                id,
		FirmwareVersionID: fwID,
		DeviceModel:       "model-a",
		Status:            domain.RolloutCampaignsStatusRunning,
	}
}

func newerFirmware(id uuid.UUID) domain.FirmwareVersion {
	return domain.FirmwareVersion{
		ID:          id,
		DeviceModel: "model-a",
		FWVersion:   "2.0.0",
		FWChecksum:  "checksum-abc",
		BinaryUrl:   "https://example.com/fw.bin",
	}
}

func expectCacheWrites(m *checkinMocks) {
	m.deviceCacheRepo.EXPECT().SetCurrentVersion(mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m.deviceCacheRepo.EXPECT().SetLastSeen(mock.Anything, mock.Anything, mock.Anything).Return(nil)
}

func TestCheckin_DeviceNotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()

	dev := activeDevice(deviceID)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(domain.Device{}, domain.ErrDeviceNotFound)

	_, err := m.service().Checkin(context.Background(), dev)
	assert.ErrorIs(t, err, domain.ErrDeviceNotFound)
}

func TestCheckin_DeviceDecommissioned_ReturnsNoUpdate(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	dev := activeDevice(deviceID)
	dev.Status = domain.DeviceStatusDecommissioned

	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)

	result, err := m.service().Checkin(context.Background(), dev)
	require.NoError(t, err)
	assert.False(t, result.UpdateAvailable)
}

func TestCheckin_NoRunningCampaign_ReturnsNoUpdate(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	dev := activeDevice(deviceID)

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	result, err := m.service().Checkin(context.Background(), dev)
	require.NoError(t, err)
	assert.False(t, result.UpdateAvailable)
}

func TestCheckin_CampaignWrongStatus_ReturnsNoUpdate(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	dev := activeDevice(deviceID)

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignWrongStatus)

	result, err := m.service().Checkin(context.Background(), dev)
	require.NoError(t, err)
	assert.False(t, result.UpdateAvailable)
}

func TestCheckin_CacheSetFails_StillContinues(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	campaignID := uuid.New()
	fwID := uuid.New()
	dev := activeDevice(deviceID)
	campaign := runningCampaign(campaignID, fwID)
	fw := newerFirmware(fwID)
	stageID := uuid.UUID{1}

	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.deviceCacheRepo.EXPECT().SetCurrentVersion(mock.Anything, mock.Anything, mock.Anything).Return(someErr())
	m.deviceCacheRepo.EXPECT().SetLastSeen(mock.Anything, mock.Anything, mock.Anything).Return(someErr())
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(campaign, nil)
	m.checkinProducer.EXPECT().Produce(mock.Anything).Return()
	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(fw, nil)
	m.campaignCacheRepo.EXPECT().GetCurrentStage(mock.Anything, campaignID).Return(stageID, nil)
	m.campaignCacheRepo.EXPECT().GetCurrentTargetPercent(mock.Anything, campaignID).Return(100, nil)

	result, err := m.service().Checkin(context.Background(), dev)
	require.NoError(t, err)
	assert.True(t, result.UpdateAvailable)
}

func TestCheckin_ProducesEvent_WhenRunningCampaignExists(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	campaignID := uuid.New()
	fwID := uuid.New()
	dev := activeDevice(deviceID)
	campaign := runningCampaign(campaignID, fwID)
	fw := newerFirmware(fwID)
	stageID := uuid.UUID{1}

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(campaign, nil)
	m.checkinProducer.EXPECT().Produce(mock.Anything).Run(func(event domain.CheckinEvent) {
		assert.Equal(t, deviceID, event.DeviceID)
		assert.Equal(t, campaignID, event.CampaignID)
		assert.Equal(t, dev.DeviceModel, event.DeviceModel)
		assert.Equal(t, dev.CurrentVersion, event.CurrentVersion)
	}).Return()
	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(fw, nil)
	m.campaignCacheRepo.EXPECT().GetCurrentStage(mock.Anything, campaignID).Return(stageID, nil)
	m.campaignCacheRepo.EXPECT().GetCurrentTargetPercent(mock.Anything, campaignID).Return(100, nil)

	result, err := m.service().Checkin(context.Background(), dev)
	require.NoError(t, err)
	assert.True(t, result.UpdateAvailable)
}

func TestCheckin_FirmwareNotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	campaignID := uuid.New()
	fwID := uuid.New()
	dev := activeDevice(deviceID)
	campaign := runningCampaign(campaignID, fwID)

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(campaign, nil)
	m.checkinProducer.EXPECT().Produce(mock.Anything).Return()
	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(domain.FirmwareVersion{}, domain.ErrFirmwareVersionNotFound)

	_, err := m.service().Checkin(context.Background(), dev)
	assert.ErrorIs(t, err, domain.ErrFirmwareVersionNotFound)
}

func TestCheckin_VersionEqualOrGreater_ReturnsNoUpdate(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	campaignID := uuid.New()
	fwID := uuid.New()
	dev := activeDevice(deviceID)
	dev.CurrentVersion = "2.0.0"
	campaign := runningCampaign(campaignID, fwID)
	fw := newerFirmware(fwID)

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(campaign, nil)
	m.checkinProducer.EXPECT().Produce(mock.Anything).Return()
	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(fw, nil)

	result, err := m.service().Checkin(context.Background(), dev)
	require.NoError(t, err)
	assert.False(t, result.UpdateAvailable)
}

func TestCheckin_CacheGetStageFails_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	campaignID := uuid.New()
	fwID := uuid.New()
	dev := activeDevice(deviceID)
	campaign := runningCampaign(campaignID, fwID)
	fw := newerFirmware(fwID)

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(campaign, nil)
	m.checkinProducer.EXPECT().Produce(mock.Anything).Return()
	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(fw, nil)
	m.campaignCacheRepo.EXPECT().GetCurrentStage(mock.Anything, campaignID).Return(uuid.UUID{}, someErr())

	_, err := m.service().Checkin(context.Background(), dev)
	require.Error(t, err)
}

func TestCheckin_CacheGetTargetPercentFails_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	campaignID := uuid.New()
	fwID := uuid.New()
	dev := activeDevice(deviceID)
	campaign := runningCampaign(campaignID, fwID)
	fw := newerFirmware(fwID)
	stageID := uuid.UUID{1}

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(campaign, nil)
	m.checkinProducer.EXPECT().Produce(mock.Anything).Return()
	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(fw, nil)
	m.campaignCacheRepo.EXPECT().GetCurrentStage(mock.Anything, campaignID).Return(stageID, nil)
	m.campaignCacheRepo.EXPECT().GetCurrentTargetPercent(mock.Anything, campaignID).Return(0, someErr())

	_, err := m.service().Checkin(context.Background(), dev)
	require.Error(t, err)
}

func TestCheckin_BucketAboveTargetPercent_ReturnsNoUpdate(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	campaignID := uuid.New()
	fwID := uuid.New()
	dev := activeDevice(deviceID)
	campaign := runningCampaign(campaignID, fwID)
	fw := newerFirmware(fwID)
	stageID := uuid.UUID{1}

	bucket := int(computeBucket(deviceID, campaignID))

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(campaign, nil)
	m.checkinProducer.EXPECT().Produce(mock.Anything).Return()
	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(fw, nil)
	m.campaignCacheRepo.EXPECT().GetCurrentStage(mock.Anything, campaignID).Return(stageID, nil)
	target := bucket - 1
	if target < 0 {
		target = 0
	}
	m.campaignCacheRepo.EXPECT().GetCurrentTargetPercent(mock.Anything, campaignID).Return(target, nil)

	result, err := m.service().Checkin(context.Background(), dev)
	require.NoError(t, err)
	assert.False(t, result.UpdateAvailable)
}

func TestCheckin_BucketWithinTargetPercent_ReturnsUpdate(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	campaignID := uuid.New()
	fwID := uuid.New()
	dev := activeDevice(deviceID)
	campaign := runningCampaign(campaignID, fwID)
	fw := newerFirmware(fwID)
	stageID := uuid.UUID{1}

	bucket := int(computeBucket(deviceID, campaignID))

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(campaign, nil)
	m.checkinProducer.EXPECT().Produce(mock.Anything).Return()
	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(fw, nil)
	m.campaignCacheRepo.EXPECT().GetCurrentStage(mock.Anything, campaignID).Return(stageID, nil)
	m.campaignCacheRepo.EXPECT().GetCurrentTargetPercent(mock.Anything, campaignID).Return(bucket, nil)

	result, err := m.service().Checkin(context.Background(), dev)
	require.NoError(t, err)
	assert.True(t, result.UpdateAvailable)
	require.NotNil(t, result.StageID)
	assert.Equal(t, stageID, *result.StageID)
	assert.Equal(t, fw.BinaryUrl, result.BinaryUrl)
	assert.Equal(t, fw.FWChecksum, result.FWChecksum)
}

func TestCheckin_TargetPercent100_AllDevicesGetUpdate(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	campaignID := uuid.New()
	fwID := uuid.New()
	dev := activeDevice(deviceID)
	campaign := runningCampaign(campaignID, fwID)
	fw := newerFirmware(fwID)
	stageID := uuid.UUID{1}

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(campaign, nil)
	m.checkinProducer.EXPECT().Produce(mock.Anything).Return()
	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(fw, nil)
	m.campaignCacheRepo.EXPECT().GetCurrentStage(mock.Anything, campaignID).Return(stageID, nil)
	m.campaignCacheRepo.EXPECT().GetCurrentTargetPercent(mock.Anything, campaignID).Return(100, nil)

	result, err := m.service().Checkin(context.Background(), dev)
	require.NoError(t, err)
	assert.True(t, result.UpdateAvailable)
}

func TestCheckin_FindRunningGenericError_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	dev := activeDevice(deviceID)

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(domain.RolloutCampaign{}, someErr())

	_, err := m.service().Checkin(context.Background(), dev)
	require.Error(t, err)
	assert.ErrorContains(t, err, "boom")
	assert.NotErrorIs(t, err, domain.ErrRolloutCampaignNotFound)
	assert.NotErrorIs(t, err, domain.ErrRolloutCampaignWrongStatus)
}

func TestCheckin_InvalidFirmwareVersion_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	campaignID := uuid.New()
	fwID := uuid.New()
	dev := activeDevice(deviceID)
	campaign := runningCampaign(campaignID, fwID)
	fw := newerFirmware(fwID)
	fw.FWVersion = "not-a-version"

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(campaign, nil)
	m.checkinProducer.EXPECT().Produce(mock.Anything).Return()
	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(fw, nil)

	_, err := m.service().Checkin(context.Background(), dev)
	require.Error(t, err)
	assert.ErrorContains(t, err, "parsing semver 'not-a-version'")
}

func TestCheckin_EmptyCurrentVersion_ReturnsError(t *testing.T) {
	t.Parallel()
	m := newCheckinMocks(t)
	deviceID := uuid.New()
	campaignID := uuid.New()
	fwID := uuid.New()
	dev := activeDevice(deviceID)
	dev.CurrentVersion = ""
	campaign := runningCampaign(campaignID, fwID)
	fw := newerFirmware(fwID)

	expectCacheWrites(m)
	m.deviceRepo.EXPECT().Get(mock.Anything, deviceID).Return(dev, nil)
	m.campaignRepo.EXPECT().FindRunning(mock.Anything, dev.DeviceModel).Return(campaign, nil)
	m.checkinProducer.EXPECT().Produce(mock.Anything).Return()
	m.firmwareRepo.EXPECT().Get(mock.Anything, fwID).Return(fw, nil)

	_, err := m.service().Checkin(context.Background(), dev)
	require.Error(t, err)
	assert.ErrorContains(t, err, "parsing semver ''")
}
