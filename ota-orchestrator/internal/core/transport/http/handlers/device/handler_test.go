package device_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers/device"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers/device/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func ctxWithLogger() context.Context {
	return config.LoggerToContext(context.Background(), zap.NewNop().Sugar())
}

func newDeviceHandler(t *testing.T) (*device.DeviceHandler, *mocks.MockDeviceService, *mocks.MockUpdateService) {
	svc := mocks.NewMockDeviceService(t)
	upd := mocks.NewMockUpdateService(t)
	return device.NewDeviceHandler(svc, upd), svc, upd
}

func newReq(method, body string) *http.Request {
	req := httptest.NewRequest(method, "/x", strings.NewReader(body))
	return req.WithContext(ctxWithLogger())
}

func withID(req *http.Request, id uuid.UUID) *http.Request {
	req.SetPathValue("id", id.String())
	return req
}

// --- Create ---

func TestCreateDevice_ValidRequest_Returns201(t *testing.T) {
	t.Parallel()
	h, svc, _ := newDeviceHandler(t)
	body, _ := json.Marshal(map[string]string{"device_model": "model-a", "current_version": "1.0.0"})
	svc.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.Device{ID: uuid.New()}, nil)

	w := httptest.NewRecorder()
	h.Create(w, newReq(http.MethodPost, string(body)))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateDevice_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()
	h, _, _ := newDeviceHandler(t)
	w := httptest.NewRecorder()
	h.Create(w, newReq(http.MethodPost, "{not json"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateDevice_UnknownField_Returns400(t *testing.T) {
	t.Parallel()
	h, _, _ := newDeviceHandler(t)
	body, _ := json.Marshal(map[string]string{"device_model": "model-a", "current_version": "1.0.0", "unknown": "x"})
	w := httptest.NewRecorder()
	h.Create(w, newReq(http.MethodPost, string(body)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateDevice_EmptyBody_Returns400(t *testing.T) {
	t.Parallel()
	h, _, _ := newDeviceHandler(t)
	w := httptest.NewRecorder()
	h.Create(w, newReq(http.MethodPost, ""))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateDevice_BodyTooLarge_Returns413(t *testing.T) {
	t.Parallel()
	h, _, _ := newDeviceHandler(t)
	huge := `{"device_model":"` + strings.Repeat("x", 1<<20) + `","current_version":"1.0.0"}`
	w := httptest.NewRecorder()
	h.Create(w, newReq(http.MethodPost, huge))
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestCreateDevice_ValidationFails_Returns400(t *testing.T) {
	t.Parallel()
	h, _, _ := newDeviceHandler(t)
	body, _ := json.Marshal(map[string]string{"device_model": "a", "current_version": "not-semver"})
	w := httptest.NewRecorder()
	h.Create(w, newReq(http.MethodPost, string(body)))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateDevice_ServiceFails_Returns500(t *testing.T) {
	t.Parallel()
	h, svc, _ := newDeviceHandler(t)
	body, _ := json.Marshal(map[string]string{"device_model": "model-a", "current_version": "1.0.0"})
	svc.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.Device{}, assertErr())

	w := httptest.NewRecorder()
	h.Create(w, newReq(http.MethodPost, string(body)))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- List ---

func TestListDevices_Success_Returns200(t *testing.T) {
	t.Parallel()
	h, svc, _ := newDeviceHandler(t)
	svc.EXPECT().List(mock.Anything, domain.DeviceFilters{}).Return([]domain.Device{{ID: uuid.New()}}, nil)

	w := httptest.NewRecorder()
	h.List(w, newReq(http.MethodGet, ""))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListDevices_WithFilters_ForwardsFilters(t *testing.T) {
	t.Parallel()
	h, svc, _ := newDeviceHandler(t)
	svc.EXPECT().List(mock.Anything, domain.DeviceFilters{DeviceModel: "model-a", Status: domain.DeviceStatusActive}).Return([]domain.Device{{ID: uuid.New()}}, nil)

	w := httptest.NewRecorder()
	h.List(w, listReq("?device_model=model-a&status=active"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListDevices_InvalidDeviceModel_Returns400(t *testing.T) {
	t.Parallel()
	h, _, _ := newDeviceHandler(t)

	w := httptest.NewRecorder()
	h.List(w, listReq("?device_model=a"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListDevices_InvalidStatus_Returns400(t *testing.T) {
	t.Parallel()
	h, _, _ := newDeviceHandler(t)

	w := httptest.NewRecorder()
	h.List(w, listReq("?status=broken"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListDevices_ServiceFails_Returns500(t *testing.T) {
	t.Parallel()
	h, svc, _ := newDeviceHandler(t)
	svc.EXPECT().List(mock.Anything, domain.DeviceFilters{}).Return(nil, assertErr())

	w := httptest.NewRecorder()
	h.List(w, newReq(http.MethodGet, ""))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func listReq(query string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/x"+query, nil)
	return req.WithContext(ctxWithLogger())
}

// --- Decommission ---

func TestDecommission_InvalidUUID_Returns400(t *testing.T) {
	t.Parallel()
	h, _, _ := newDeviceHandler(t)
	req := newReq(http.MethodPost, "")
	req.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()
	h.Decommission(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDecommission_DeviceNotFound_Returns404(t *testing.T) {
	t.Parallel()
	h, svc, _ := newDeviceHandler(t)
	id := uuid.New()
	svc.EXPECT().Decommission(mock.Anything, id).Return(domain.Device{}, domain.ErrDeviceNotFound)

	w := httptest.NewRecorder()
	h.Decommission(w, withID(newReq(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDecommission_ServiceFails_Returns500(t *testing.T) {
	t.Parallel()
	h, svc, _ := newDeviceHandler(t)
	id := uuid.New()
	svc.EXPECT().Decommission(mock.Anything, id).Return(domain.Device{}, assertErr())

	w := httptest.NewRecorder()
	h.Decommission(w, withID(newReq(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDecommission_Success_Returns200(t *testing.T) {
	t.Parallel()
	h, svc, _ := newDeviceHandler(t)
	id := uuid.New()
	svc.EXPECT().Decommission(mock.Anything, id).Return(domain.Device{ID: id, Status: domain.DeviceStatusDecommissioned}, nil)

	w := httptest.NewRecorder()
	h.Decommission(w, withID(newReq(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Checkin ---

func TestCheckin_InvalidUUID_Returns400(t *testing.T) {
	t.Parallel()
	h, _, _ := newDeviceHandler(t)
	req := newReq(http.MethodPost, `{"current_version":"1.0.0"}`)
	req.SetPathValue("id", "bad")
	w := httptest.NewRecorder()
	h.Checkin(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCheckin_InvalidBody_Returns400(t *testing.T) {
	t.Parallel()
	h, _, _ := newDeviceHandler(t)
	id := uuid.New()
	w := httptest.NewRecorder()
	h.Checkin(w, withID(newReq(http.MethodPost, "{bad"), id))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCheckin_DeviceNotFound_Returns404(t *testing.T) {
	t.Parallel()
	h, _, upd := newDeviceHandler(t)
	id := uuid.New()
	upd.EXPECT().Checkin(mock.Anything, mock.Anything).Return(domain.CheckinResult{}, domain.ErrDeviceNotFound)

	w := httptest.NewRecorder()
	h.Checkin(w, withID(newReq(http.MethodPost, `{"current_version":"1.0.0"}`), id))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCheckin_ServiceFails_Returns500(t *testing.T) {
	t.Parallel()
	h, _, upd := newDeviceHandler(t)
	id := uuid.New()
	upd.EXPECT().Checkin(mock.Anything, mock.Anything).Return(domain.CheckinResult{}, assertErr())

	w := httptest.NewRecorder()
	h.Checkin(w, withID(newReq(http.MethodPost, `{"current_version":"1.0.0"}`), id))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCheckin_Success_Returns200(t *testing.T) {
	t.Parallel()
	h, _, upd := newDeviceHandler(t)
	id := uuid.New()
	stageID := uuid.New()
	upd.EXPECT().Checkin(mock.Anything, mock.Anything).Return(domain.CheckinResult{
		UpdateAvailable: true, StageID: &stageID, BinaryUrl: "u", FWChecksum: "c",
	}, nil)

	w := httptest.NewRecorder()
	h.Checkin(w, withID(newReq(http.MethodPost, `{"current_version":"1.0.0"}`), id))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "update_available")
}

func TestCheckin_NoUpdate_Returns200WithFalse(t *testing.T) {
	t.Parallel()
	h, _, upd := newDeviceHandler(t)
	id := uuid.New()
	upd.EXPECT().Checkin(mock.Anything, mock.Anything).Return(domain.CheckinResult{UpdateAvailable: false}, nil)

	w := httptest.NewRecorder()
	h.Checkin(w, withID(newReq(http.MethodPost, `{"current_version":"1.0.0"}`), id))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"update_available":false`)
}

// --- Report ---

func TestReport_InvalidUUID_Returns400(t *testing.T) {
	t.Parallel()
	h, _, _ := newDeviceHandler(t)
	req := newReq(http.MethodPost, `{"campaign_id":"x","stage_id":"y","result":"success"}`)
	req.SetPathValue("id", "bad")
	w := httptest.NewRecorder()
	h.Report(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReport_InvalidBody_Returns400(t *testing.T) {
	t.Parallel()
	h, _, _ := newDeviceHandler(t)
	id := uuid.New()
	w := httptest.NewRecorder()
	h.Report(w, withID(newReq(http.MethodPost, "{bad"), id))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReport_DeviceNotFound_Returns404(t *testing.T) {
	t.Parallel()
	h, _, upd := newDeviceHandler(t)
	id := uuid.New()
	body, _ := json.Marshal(map[string]string{"campaign_id": uuid.New().String(), "stage_id": uuid.New().String(), "result": "success"})
	upd.EXPECT().Report(mock.Anything, mock.Anything).Return(domain.UpdateAttempt{}, domain.ErrDeviceNotFound)

	w := httptest.NewRecorder()
	h.Report(w, withID(newReq(http.MethodPost, string(body)), id))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestReport_CampaignNotFound_Returns400(t *testing.T) {
	t.Parallel()
	h, _, upd := newDeviceHandler(t)
	id := uuid.New()
	body, _ := json.Marshal(map[string]string{"campaign_id": uuid.New().String(), "stage_id": uuid.New().String(), "result": "success"})
	upd.EXPECT().Report(mock.Anything, mock.Anything).Return(domain.UpdateAttempt{}, domain.ErrRolloutCampaignNotFound)

	w := httptest.NewRecorder()
	h.Report(w, withID(newReq(http.MethodPost, string(body)), id))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReport_StageNotInCampaign_Returns400(t *testing.T) {
	t.Parallel()
	h, _, upd := newDeviceHandler(t)
	id := uuid.New()
	body, _ := json.Marshal(map[string]string{"campaign_id": uuid.New().String(), "stage_id": uuid.New().String(), "result": "success"})
	upd.EXPECT().Report(mock.Anything, mock.Anything).Return(domain.UpdateAttempt{}, domain.ErrRolloutStageNotFoundInCampaign)

	w := httptest.NewRecorder()
	h.Report(w, withID(newReq(http.MethodPost, string(body)), id))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReport_WrongDeviceModel_Returns400(t *testing.T) {
	t.Parallel()
	h, _, upd := newDeviceHandler(t)
	id := uuid.New()
	body, _ := json.Marshal(map[string]string{"campaign_id": uuid.New().String(), "stage_id": uuid.New().String(), "result": "success"})
	upd.EXPECT().Report(mock.Anything, mock.Anything).Return(domain.UpdateAttempt{}, domain.ErrWrongDeviceModel)

	w := httptest.NewRecorder()
	h.Report(w, withID(newReq(http.MethodPost, string(body)), id))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReport_ProduceFailed_Returns503(t *testing.T) {
	t.Parallel()
	h, _, upd := newDeviceHandler(t)
	id := uuid.New()
	body, _ := json.Marshal(map[string]string{"campaign_id": uuid.New().String(), "stage_id": uuid.New().String(), "result": "success"})
	upd.EXPECT().Report(mock.Anything, mock.Anything).Return(domain.UpdateAttempt{}, domain.ErrUpdateResultNotProduced)

	w := httptest.NewRecorder()
	h.Report(w, withID(newReq(http.MethodPost, string(body)), id))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestReport_ServiceFails_Returns500(t *testing.T) {
	t.Parallel()
	h, _, upd := newDeviceHandler(t)
	id := uuid.New()
	body, _ := json.Marshal(map[string]string{"campaign_id": uuid.New().String(), "stage_id": uuid.New().String(), "result": "success"})
	upd.EXPECT().Report(mock.Anything, mock.Anything).Return(domain.UpdateAttempt{}, assertErr())

	w := httptest.NewRecorder()
	h.Report(w, withID(newReq(http.MethodPost, string(body)), id))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReport_Success_Returns200(t *testing.T) {
	t.Parallel()
	h, _, upd := newDeviceHandler(t)
	deviceID := uuid.New()
	campaignID := uuid.New()
	stageID := uuid.New()
	body, _ := json.Marshal(map[string]string{"campaign_id": campaignID.String(), "stage_id": stageID.String(), "result": "success"})
	attempt := domain.UpdateAttempt{ID: uuid.New(), DeviceID: deviceID, CampaignID: campaignID, StageID: stageID, Result: domain.UpdateAttemptsResultSuccess}
	upd.EXPECT().Report(mock.Anything, mock.MatchedBy(func(a domain.UpdateAttempt) bool {
		return a.DeviceID == deviceID && a.CampaignID == campaignID && a.StageID == stageID && a.Result == domain.UpdateAttemptsResultSuccess
	})).Return(attempt, nil)

	w := httptest.NewRecorder()
	h.Report(w, withID(newReq(http.MethodPost, string(body)), deviceID))
	assert.Equal(t, http.StatusOK, w.Code)
}

func assertErr() error {
	return errors.New("boom")
}
