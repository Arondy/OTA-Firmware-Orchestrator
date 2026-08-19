package firmware_version_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers/firmware_version"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers/firmware_version/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func newHandler(t *testing.T) (*firmware_version.FirmwareVersionHandler, *mocks.MockFirmwareVersionService) {
	svc := mocks.NewMockFirmwareVersionService(t)
	return firmware_version.NewFirmwareVersionHandler(svc), svc
}

func req(method, body string) *http.Request {
	r := httptest.NewRequest(method, "/x", strings.NewReader(body))
	return r.WithContext(config.LoggerToContext(r.Context(), zap.NewNop().Sugar()))
}

func TestCreateFirmware_ValidRequest_Returns201(t *testing.T) {
	h, svc := newHandler(t)
	body, _ := json.Marshal(map[string]string{
		"device_model": "model-a", "fw_version": "1.0.0",
		"fw_checksum": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"binary_url":  "https://example.com/fw.bin",
	})
	svc.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.FirmwareVersion{ID: uuid.New()}, nil)

	w := httptest.NewRecorder()
	h.Create(w, req(http.MethodPost, string(body)))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateFirmware_ValidationFails_Returns400(t *testing.T) {
	h, _ := newHandler(t)
	body, _ := json.Marshal(map[string]string{
		"device_model": "a", "fw_version": "not-semver",
		"fw_checksum": "xyz", "binary_url": "not-a-url",
	})
	w := httptest.NewRecorder()
	h.Create(w, req(http.MethodPost, string(body)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateFirmware_AlreadyExists_Returns409(t *testing.T) {
	h, svc := newHandler(t)
	body, _ := json.Marshal(map[string]string{
		"device_model": "model-a", "fw_version": "1.0.0",
		"fw_checksum": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"binary_url":  "https://example.com/fw.bin",
	})
	svc.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.FirmwareVersion{}, domain.ErrFirmwareVersionAlreadyExists)

	w := httptest.NewRecorder()
	h.Create(w, req(http.MethodPost, string(body)))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCreateFirmware_ServiceFails_Returns500(t *testing.T) {
	h, svc := newHandler(t)
	body, _ := json.Marshal(map[string]string{
		"device_model": "model-a", "fw_version": "1.0.0",
		"fw_checksum": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"binary_url":  "https://example.com/fw.bin",
	})
	svc.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.FirmwareVersion{}, assertFwErr())

	w := httptest.NewRecorder()
	h.Create(w, req(http.MethodPost, string(body)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListFirmware_Success_Returns200(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().List(mock.Anything).Return([]domain.FirmwareVersion{{ID: uuid.New()}}, nil)

	w := httptest.NewRecorder()
	h.List(w, req(http.MethodGet, ""))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListFirmware_ServiceFails_Returns500(t *testing.T) {
	h, svc := newHandler(t)
	svc.EXPECT().List(mock.Anything).Return(nil, assertFwErr())

	w := httptest.NewRecorder()
	h.List(w, req(http.MethodGet, ""))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func assertFwErr() error {
	return errors.New("boom")
}
