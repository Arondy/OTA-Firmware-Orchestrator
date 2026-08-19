package firmware_version

import (
	"testing"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestFirmwareVersionFromDomain_AllFieldsCopied(t *testing.T) {
	now := time.Now()
	fw := domain.FirmwareVersion{
		ID:          uuid.New(),
		DeviceModel: "model-x",
		FWVersion:   "1.2.3",
		FWChecksum:  "abcdef",
		BinaryUrl:   "https://example.com/fw.bin",
		CreatedAt:   now,
	}

	resp := FirmwareVersionFromDomain(fw)
	assert.Equal(t, fw.ID, resp.ID)
	assert.Equal(t, fw.DeviceModel, resp.DeviceModel)
	assert.Equal(t, fw.FWVersion, resp.FWVersion)
	assert.Equal(t, fw.FWChecksum, resp.FWChecksum)
	assert.Equal(t, fw.BinaryUrl, resp.BinaryUrl)
	assert.Equal(t, fw.CreatedAt.Unix(), resp.CreatedAt.Unix())
}
