package handlers

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	healthv1 "github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/health/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler_Connect_ReturnsOK(t *testing.T) {
	h := NewHealthHandler()
	req := connect.NewRequest(&healthv1.CheckHealthRequest{})

	resp, err := h.CheckHealth(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "OK", resp.Msg.Status)
}
