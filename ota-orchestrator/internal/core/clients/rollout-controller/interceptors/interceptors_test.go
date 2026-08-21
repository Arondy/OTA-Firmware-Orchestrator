package interceptors_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	healthv1 "github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/health/v1"
	"github.com/Arondy/OTA-Firmware-Orchestrator/api/gen/health/v1/healthv1connect"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/clients/rollout-controller/interceptors"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func capturingNext(header *string) connect.UnaryFunc {
	return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		*header = req.Header().Get(config.RequestIDHeader)
		return connect.NewResponse(&healthv1.CheckHealthResponse{Status: "OK"}), nil
	})
}

func TestRequestID_SetsOutgoingHeaderFromContext(t *testing.T) {
	t.Parallel()
	var got string

	ctx := context.WithValue(context.Background(), config.CtxKeyRequestID{}, "req-42")
	_, err := interceptors.RequestID().WrapUnary(capturingNext(&got))(ctx, connect.NewRequest(&healthv1.CheckHealthRequest{}))

	require.NoError(t, err)
	assert.Equal(t, "req-42", got)
}

func TestRequestID_NoIDInContext_LeavesHeaderEmpty(t *testing.T) {
	t.Parallel()
	var got string

	_, err := interceptors.RequestID().WrapUnary(capturingNext(&got))(context.Background(), connect.NewRequest(&healthv1.CheckHealthRequest{}))

	require.NoError(t, err)
	assert.Empty(t, got)
}

type capturingHealth struct {
	healthv1connect.UnimplementedHealthServiceHandler
	onCheck func(*connect.Request[healthv1.CheckHealthRequest])
}

func (h *capturingHealth) CheckHealth(_ context.Context, req *connect.Request[healthv1.CheckHealthRequest]) (*connect.Response[healthv1.CheckHealthResponse], error) {
	h.onCheck(req)
	return connect.NewResponse(&healthv1.CheckHealthResponse{Status: "OK"}), nil
}

func TestNewInterceptorsOption_PropagatesRequestIDOverWire(t *testing.T) {
	t.Parallel()
	var serverSeen string

	mux := http.NewServeMux()
	mux.Handle(healthv1connect.NewHealthServiceHandler(&capturingHealth{
		onCheck: func(req *connect.Request[healthv1.CheckHealthRequest]) {
			serverSeen = req.Header().Get(config.RequestIDHeader)
		},
	}))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := healthv1connect.NewHealthServiceClient(srv.Client(), srv.URL, interceptors.NewInterceptorsOption())
	ctx := context.WithValue(context.Background(), config.CtxKeyRequestID{}, "wire-id-7")
	_, err := client.CheckHealth(ctx, connect.NewRequest(&healthv1.CheckHealthRequest{}))

	require.NoError(t, err)
	assert.Equal(t, "wire-id-7", serverSeen)
}
