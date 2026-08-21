package health

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestHealthHandler_HTTP_ReturnsOK(t *testing.T) {
	t.Parallel()
	h := NewHealthHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r = r.WithContext(config.LoggerToContext(r.Context(), zap.NewNop().Sugar()))

	h.CheckHealth(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "OK")
}
