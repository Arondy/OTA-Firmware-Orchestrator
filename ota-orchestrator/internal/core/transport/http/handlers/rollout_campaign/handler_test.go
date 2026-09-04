package rollout_campaign_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers/rollout_campaign"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers/rollout_campaign/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func newHandler(t *testing.T) (*rollout_campaign.RolloutCampaignHandler, *mocks.MockRolloutCampaignService) {
	svc := mocks.NewMockRolloutCampaignService(t)
	return rollout_campaign.NewRolloutCampaignHandler(svc), svc
}

func req(method, body string) *http.Request {
	r := httptest.NewRequest(method, "/x", strings.NewReader(body))
	return r.WithContext(config.LoggerToContext(r.Context(), zap.NewNop().Sugar()))
}

func withID(r *http.Request, id uuid.UUID) *http.Request {
	r.SetPathValue("id", id.String())
	return r
}

func validCreateBody() string {
	b, _ := json.Marshal(map[string]any{
		"firmware_version_id": uuid.New().String(),
		"rollout_stages": []map[string]any{
			{"order_index": 0, "target_percent": 50, "min_sample_size": 10, "success_threshold": 0.5},
		},
	})
	return string(b)
}

func invalidSequenceBody() string {
	b, _ := json.Marshal(map[string]any{
		"firmware_version_id": uuid.New().String(),
		"rollout_stages": []map[string]any{
			{"order_index": 0, "target_percent": 50, "min_sample_size": 10, "success_threshold": 0.5},
			{"order_index": 2, "target_percent": 50, "min_sample_size": 10, "success_threshold": 0.5},
		},
	})
	return string(b)
}

// --- Create ---

func TestCreateCampaign_ValidRequest_Returns201(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	svc.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.RolloutCampaign{ID: uuid.New()}, nil)

	w := httptest.NewRecorder()
	h.Create(w, req(http.MethodPost, validCreateBody()))
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateCampaign_InvalidStagesSequence_Returns400(t *testing.T) {
	t.Parallel()
	h, _ := newHandler(t)
	w := httptest.NewRecorder()
	h.Create(w, req(http.MethodPost, invalidSequenceBody()))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateCampaign_FirmwareNotFound_Returns400(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	svc.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.RolloutCampaign{}, domain.ErrFirmwareVersionNotFound)

	w := httptest.NewRecorder()
	h.Create(w, req(http.MethodPost, validCreateBody()))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateCampaign_StageAlreadyExists_Returns400(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	svc.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.RolloutCampaign{}, domain.ErrRolloutStageAlreadyExists)

	w := httptest.NewRecorder()
	h.Create(w, req(http.MethodPost, validCreateBody()))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateCampaign_ServiceFails_Returns500(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	svc.EXPECT().Create(mock.Anything, mock.Anything).Return(domain.RolloutCampaign{}, errors.New("boom"))

	w := httptest.NewRecorder()
	h.Create(w, req(http.MethodPost, validCreateBody()))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- Get ---

func TestGetCampaign_InvalidUUID_Returns400(t *testing.T) {
	t.Parallel()
	h, _ := newHandler(t)
	r := req(http.MethodGet, "")
	r.SetPathValue("id", "bad")
	w := httptest.NewRecorder()
	h.Get(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetCampaign_NotFound_Returns404(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Get(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	w := httptest.NewRecorder()
	h.Get(w, withID(req(http.MethodGet, ""), id))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCampaign_Success_Returns200WithStagesAndStats(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	stageID := uuid.New()
	campaign := domain.RolloutCampaign{
		ID:          id,
		DeviceModel: "model-a",
		Status:      domain.RolloutCampaignsStatusRunning,
		RolloutStages: []domain.RolloutStage{
			{ID: stageID, OrderIndex: 0, Status: domain.RolloutStagesStatusActive},
		},
		Stats: &domain.RolloutCampaignStats{ActiveStageID: stageID, SampleSize: 1},
	}
	svc.EXPECT().Get(mock.Anything, id).Return(campaign, nil)

	w := httptest.NewRecorder()
	h.Get(w, withID(req(http.MethodGet, ""), id))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "rollout_stages")
	assert.Contains(t, w.Body.String(), "stats")
}

// --- List ---

func TestListCampaigns_Success_Returns200(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	svc.EXPECT().List(mock.Anything).Return([]domain.RolloutCampaign{{ID: uuid.New()}}, nil)

	w := httptest.NewRecorder()
	h.List(w, req(http.MethodGet, ""))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListCampaigns_ServiceFails_Returns500(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	svc.EXPECT().List(mock.Anything).Return(nil, errors.New("boom"))

	w := httptest.NewRecorder()
	h.List(w, req(http.MethodGet, ""))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- Start ---

func TestStartCampaign_InvalidUUID_Returns400(t *testing.T) {
	t.Parallel()
	h, _ := newHandler(t)
	r := req(http.MethodPost, "")
	r.SetPathValue("id", "bad")
	w := httptest.NewRecorder()
	h.Start(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStartCampaign_NotFound_Returns404(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Start(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	w := httptest.NewRecorder()
	h.Start(w, withID(req(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestStartCampaign_AlreadyRunning_Returns409(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Start(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrCampaignAlreadyRunning)

	w := httptest.NewRecorder()
	h.Start(w, withID(req(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestStartCampaign_WrongStatus_Returns400(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Start(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignWrongStatus)

	w := httptest.NewRecorder()
	h.Start(w, withID(req(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStartCampaign_Success_Returns200(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Start(mock.Anything, id).Return(domain.RolloutCampaign{ID: id, Status: domain.RolloutCampaignsStatusRunning}, nil)

	w := httptest.NewRecorder()
	h.Start(w, withID(req(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Pause ---

func TestPauseCampaign_InvalidUUID_Returns400(t *testing.T) {
	t.Parallel()
	h, _ := newHandler(t)
	r := req(http.MethodPost, "")
	r.SetPathValue("id", "bad")
	w := httptest.NewRecorder()
	h.Pause(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPauseCampaign_NotFound_Returns404(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Pause(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	w := httptest.NewRecorder()
	h.Pause(w, withID(req(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPauseCampaign_WrongStatus_Returns400(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Pause(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignWrongStatus)

	w := httptest.NewRecorder()
	h.Pause(w, withID(req(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPauseCampaign_Success_Returns200(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Pause(mock.Anything, id).Return(domain.RolloutCampaign{ID: id, Status: domain.RolloutCampaignsStatusPaused}, nil)

	w := httptest.NewRecorder()
	h.Pause(w, withID(req(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Resume ---

func TestResumeCampaign_InvalidUUID_Returns400(t *testing.T) {
	t.Parallel()
	h, _ := newHandler(t)
	r := req(http.MethodPost, "")
	r.SetPathValue("id", "bad")
	w := httptest.NewRecorder()
	h.Resume(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResumeCampaign_NotFound_Returns404(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Resume(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignNotFound)

	w := httptest.NewRecorder()
	h.Resume(w, withID(req(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestResumeCampaign_AlreadyRunning_Returns409(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Resume(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrCampaignAlreadyRunning)

	w := httptest.NewRecorder()
	h.Resume(w, withID(req(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestResumeCampaign_WrongStatus_Returns400(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Resume(mock.Anything, id).Return(domain.RolloutCampaign{}, domain.ErrRolloutCampaignWrongStatus)

	w := httptest.NewRecorder()
	h.Resume(w, withID(req(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResumeCampaign_Success_Returns200(t *testing.T) {
	t.Parallel()
	h, svc := newHandler(t)
	id := uuid.New()
	svc.EXPECT().Resume(mock.Anything, id).Return(domain.RolloutCampaign{ID: id, Status: domain.RolloutCampaignsStatusRunning}, nil)

	w := httptest.NewRecorder()
	h.Resume(w, withID(req(http.MethodPost, ""), id))
	assert.Equal(t, http.StatusOK, w.Code)
}
