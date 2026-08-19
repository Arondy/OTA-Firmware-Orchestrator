package device

import (
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestReportRequestToDomainWithDeviceID(t *testing.T) {
	deviceID := uuid.New()
	campaignID := uuid.New()
	stageID := uuid.New()
	req := ReportDeviceRequest{
		CampaignID: campaignID,
		StageID:    stageID,
		Result:     domain.UpdateAttemptsResultFailure,
	}
	attempt := req.ToDomainWithDeviceID(deviceID)
	assert.Equal(t, deviceID, attempt.DeviceID)
	assert.Equal(t, campaignID, attempt.CampaignID)
	assert.Equal(t, stageID, attempt.StageID)
	assert.Equal(t, domain.UpdateAttemptsResultFailure, attempt.Result)
}
