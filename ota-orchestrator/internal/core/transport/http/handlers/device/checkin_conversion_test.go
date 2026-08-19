package device

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCheckinRequestToDomainWithID(t *testing.T) {
	id := uuid.New()
	req := CheckinDeviceRequest{CurrentVersion: "1.2.3"}
	device := req.ToDomainWithID(id)
	assert.Equal(t, id, device.ID)
	assert.Equal(t, "1.2.3", device.CurrentVersion)
}
