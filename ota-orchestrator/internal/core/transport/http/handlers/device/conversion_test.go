package device

import (
	"testing"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceFromDomain_AllFieldsCopied(t *testing.T) {
	now := time.Now()
	lastSeen := now.Add(-time.Hour)

	withLastSeen := domain.Device{
		ID:             uuid.New(),
		DeviceModel:    "model-a",
		CurrentVersion: "1.0.0",
		Status:         domain.DeviceStatusActive,
		LastSeen:       &lastSeen,
		CreatedAt:      now,
	}
	resp := DeviceFromDomain(withLastSeen)
	assert.Equal(t, withLastSeen.ID, resp.ID)
	assert.Equal(t, withLastSeen.DeviceModel, resp.DeviceModel)
	assert.Equal(t, withLastSeen.CurrentVersion, resp.CurrentVersion)
	assert.Equal(t, withLastSeen.Status, resp.Status)
	require.NotNil(t, resp.LastSeen)
	assert.Equal(t, withLastSeen.LastSeen.Unix(), resp.LastSeen.Unix())
	assert.Equal(t, withLastSeen.CreatedAt.Unix(), resp.CreatedAt.Unix())

	withoutLastSeen := domain.Device{
		ID:             uuid.New(),
		DeviceModel:    "model-b",
		CurrentVersion: "2.0.0",
		Status:         domain.DeviceStatusDecommissioned,
		LastSeen:       nil,
		CreatedAt:      now,
	}
	respNil := DeviceFromDomain(withoutLastSeen)
	assert.Nil(t, respNil.LastSeen)
	assert.Equal(t, withoutLastSeen.Status, respNil.Status)
}

func TestCheckinResponseFromDomain_UpdateAvailableTrue(t *testing.T) {
	stageID := uuid.New()
	result := domain.CheckinResult{
		UpdateAvailable: true,
		StageID:         &stageID,
		BinaryUrl:       "https://example.com/fw.bin",
		FWChecksum:      "deadbeef",
	}

	resp := CheckinResponseFromDomain(result)
	assert.True(t, resp.UpdateAvailable)
	require.NotNil(t, resp.StageID)
	assert.Equal(t, stageID, *resp.StageID)
	assert.Equal(t, result.BinaryUrl, resp.BinaryUrl)
	assert.Equal(t, result.FWChecksum, resp.FWChecksum)
}

func TestCheckinResponseFromDomain_UpdateAvailableFalse(t *testing.T) {
	result := domain.CheckinResult{
		UpdateAvailable: false,
		StageID:         nil,
		BinaryUrl:       "",
		FWChecksum:      "",
	}

	resp := CheckinResponseFromDomain(result)
	assert.False(t, resp.UpdateAvailable)
	assert.Nil(t, resp.StageID)
	assert.Empty(t, resp.BinaryUrl)
	assert.Empty(t, resp.FWChecksum)
}

func TestReportResponseFromDomain_AllFieldsCopied(t *testing.T) {
	id := uuid.New()
	deviceID := uuid.New()
	campaignID := uuid.New()
	stageID := uuid.New()
	now := time.Now()

	attempt := domain.UpdateAttempt{
		ID:         id,
		DeviceID:   deviceID,
		CampaignID: campaignID,
		StageID:    stageID,
		Result:     domain.UpdateAttemptsResultSuccess,
		EventID:    uuid.New(),
		ReportedAt: now,
	}

	resp := ReportResponseFromDomain(attempt)
	assert.Equal(t, id, resp.ID)
	assert.Equal(t, deviceID, resp.DeviceID)
	assert.Equal(t, campaignID, resp.CampaignID)
	assert.Equal(t, stageID, resp.StageID)
	assert.Equal(t, domain.UpdateAttemptsResultSuccess, resp.Result)
	assert.Equal(t, now.Unix(), resp.ReportedAt.Unix())
}
