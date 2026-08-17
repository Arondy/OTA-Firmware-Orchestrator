package handlers

import (
	"context"

	"connectrpc.com/connect"
	healthv1 "github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/health/v1"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) CheckHealth(ctx context.Context, req *connect.Request[healthv1.CheckHealthRequest]) (*connect.Response[healthv1.CheckHealthResponse], error) {
	return connect.NewResponse(&healthv1.CheckHealthResponse{Status: "OK"}), nil
}
