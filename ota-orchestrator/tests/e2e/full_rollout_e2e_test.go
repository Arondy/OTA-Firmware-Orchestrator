//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	orchestratorcore "github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core"
	orchestratorconfig "github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	tkafka "github.com/Arondy/OTA-Firmware-Orchestrator/testutil/kafka"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	updateResultsTopic       = "firmware.update-results"
	checkinsTopic            = "device.checkins"
	updateResultsDLQTopic    = "firmware.update-results.dlq"
	rolloutDecisionsTopic    = "rollout.decisions"
	rolloutDecisionsDLQTopic = "rollout.decisions.dlq"

	// Match docker-compose.yml image versions.
	postgresImage = "postgres:18-alpine"
	redisImage    = "redis:8-alpine"

	statsPollTimeout     = 90 * time.Second
	readinessPollTimeout = 150 * time.Second
	controllerGroupID    = "e2e-test-group"
	orchestratorGroupID  = "e2e-orchestrator-group"
)

var (
	orchBaseURL     string
	ctrlBaseURL     string
	httpClient      *http.Client
	kafkaBrokerAddr string
	testCtx         context.Context
	cancelAll       context.CancelFunc
	controllerCmd   *exec.Cmd
	pgContainer     *postgres.PostgresContainer
	redisContainer  *redis.RedisContainer
	ctrlBinDir      string
)

// ---- response/request DTOs (mirrors the HTTP handlers) ----

type deviceResp struct {
	ID             string `json:"id"`
	DeviceModel    string `json:"device_model"`
	CurrentVersion string `json:"current_version"`
	Status         string `json:"status"`
}

type firmwareResp struct {
	ID string `json:"id"`
}

type stageResp struct {
	ID               string  `json:"id"`
	CampaignID       string  `json:"campaign_id"`
	OrderIndex       int     `json:"order_index"`
	TargetPercent    int     `json:"target_percent"`
	MinSampleSize    int     `json:"min_sample_size"`
	SuccessThreshold float32 `json:"success_threshold"`
	Status           string  `json:"status"`
}

type statsResp struct {
	ActiveStageID string  `json:"active_stage_id"`
	SuccessRate   float32 `json:"success_rate"`
	SampleSize    int     `json:"sample_size"`
}

type campaignResp struct {
	ID                string      `json:"id"`
	FirmwareVersionID string      `json:"firmware_version_id"`
	DeviceModel       string      `json:"device_model"`
	Status            string      `json:"status"`
	RolloutStages     []stageResp `json:"rollout_stages"`
	Stats             *statsResp  `json:"stats"`
}

type checkinResp struct {
	UpdateAvailable bool    `json:"update_available"`
	StageID         *string `json:"stage_id"`
	BinaryUrl       string  `json:"binary_url"`
	FWChecksum      string  `json:"fw_checksum"`
}

type reportResp struct {
	ID         string `json:"id"`
	DeviceID   string `json:"device_id"`
	CampaignID string `json:"campaign_id"`
	StageID    string `json:"stage_id"`
	Result     string `json:"result"`
}

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())
	cancelAll = cancel

	testCtx = ctx
	logger := newTestLogger()

	pgC, err := postgres.Run(ctx, postgresImage,
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		fail("failed to start postgres: %v", err)
	}
	pgContainer = pgC

	redisC, err := redis.Run(ctx, redisImage)
	if err != nil {
		fail("failed to start redis: %v", err)
	}
	redisContainer = redisC

	kafkaBrokerAddr = tkafka.BrokerAddr()

	dbHost, err := pgC.Host(ctx)
	if err != nil {
		fail("postgres host: %v", err)
	}
	dbPort, err := mappedPortInt(pgC.MappedPort(ctx, "5432/tcp"))
	if err != nil {
		fail("postgres port: %v", err)
	}

	redisHost, err := redisC.Host(ctx)
	if err != nil {
		fail("redis host: %v", err)
	}
	redisPort, err := mappedPortInt(redisC.MappedPort(ctx, "6379/tcp"))
	if err != nil {
		fail("redis port: %v", err)
	}

	kafkaHost := "127.0.0.1"

	httpClient = &http.Client{Timeout: 20 * time.Second}

	connStr := fmt.Sprintf("postgres://test:test@%s:%d/testdb?sslmode=disable", dbHost, dbPort)
	applyMigrations(ctx, connStr)
	for _, name := range []string{checkinsTopic, updateResultsTopic, updateResultsDLQTopic, rolloutDecisionsTopic, rolloutDecisionsDLQTopic} {
		tkafka.CreateTopic(name, 3)
	}

	_, kafkaPort := tkafka.BrokerHostPort()
	ctrlPort := freePort()
	ctrlBin := buildControllerBin()
	ctrlCmd := exec.Command(ctrlBin)
	ctrlCmd.Dir = filepath.Dir(ctrlBin)
	ctrlCmd.Env = append(os.Environ(), controllerEnv(ctrlPort, redisHost, redisPort, kafkaHost, kafkaPort)...)
	ctrlCmd.Stdout = os.Stdout
	ctrlCmd.Stderr = os.Stderr
	if err := ctrlCmd.Start(); err != nil {
		fail("failed to start rollout-controller: %v", err)
	}
	controllerCmd = ctrlCmd
	// Контроллер валидирует env при старте и паникует на неполном конфиге —
	// падаем сразу, а не ждём весь readinessPollTimeout.
	controllerExited := make(chan error, 1)
	go func() { controllerExited <- ctrlCmd.Wait() }()
	ctrlBaseURL = fmt.Sprintf("http://127.0.0.1:%d", ctrlPort)
	if err := waitForConnectHealth(ctrlBaseURL, controllerExited, readinessPollTimeout); err != nil {
		fail("rollout-controller not ready: %v", err)
	}

	orchPort := freePort()
	orchCfg := &orchestratorconfig.Config{
		ShutdownTimeout: 10 * time.Second,
		HTTPServer: orchestratorconfig.HTTPServerConfig{
			Host:    "127.0.0.1",
			Port:    uint16(orchPort),
			Timeout: 30 * time.Second,
		},
		DB: orchestratorconfig.DBConfig{
			Host:           dbHost,
			Port:           dbPort,
			User:           "test",
			Password:       "test",
			Name:           "testdb",
			SSLMode:        "disable",
			RequestTimeout: 10 * time.Second,
		},
		Cache: orchestratorconfig.CacheConfig{
			Host:                 redisHost,
			Port:                 redisPort,
			ReadTimeout:          500 * time.Millisecond,
			WriteTimeout:         500 * time.Millisecond,
			DeviceCheckinDataTTL: 24 * time.Hour,
		},
		Broker: orchestratorconfig.BrokerConfig{
			Host:         kafkaHost,
			Port:         kafkaPort,
			BatchTimeout: 100 * time.Millisecond,
			Timeout:      5 * time.Second,
			BufferSize:   1024,
			// Без GroupID kafka-go читает только partition 0, а продюсер
			// контроллера раскладывает решения по партициям хешем campaign_id.
			GroupID:        orchestratorGroupID,
			MinBytes:       1,
			ReaderMaxWait:  time.Second,
			CommitTimeout:  2 * time.Second,
			DLQTimeout:     5 * time.Second,
			DLQMaxAttempts: 3,
		},
		RolloutController: orchestratorconfig.RolloutControllerConfig{
			Scheme:  "http",
			Host:    "127.0.0.1",
			Port:    ctrlPort,
			Timeout: 5 * time.Second,
		},
	}

	// Те же правила, что продовый LoadConfig: незаполненное обязательное поле
	// должно ронять сьют сразу, а не вешать тесты на таймаутах.
	if err := validator.New().Struct(orchCfg); err != nil {
		fail("invalid orchestrator test config: %v", err)
	}

	go func() {
		_ = orchestratorcore.Run(ctx, orchCfg, logger)
	}()

	orchBaseURL = fmt.Sprintf("http://127.0.0.1:%d", orchPort)
	if err := waitForHTTPReady(orchBaseURL+"/healthz", readinessPollTimeout); err != nil {
		fail("ota-orchestrator not ready: %v", err)
	}

	code := m.Run()

	shutdown()
	os.Exit(code)
}

func fail(format string, args ...any) {
	log.Printf("fatal: "+format, args...)
	shutdown()
	os.Exit(1)
}

func shutdown() {
	if cancelAll != nil {
		cancelAll()
	}
	if controllerCmd != nil && controllerCmd.Process != nil {
		_ = controllerCmd.Process.Kill()
		_, _ = controllerCmd.Process.Wait()
	}
	tkafka.Terminate()
	if pgContainer != nil {
		_ = pgContainer.Terminate(context.Background())
	}
	if redisContainer != nil {
		_ = redisContainer.Terminate(context.Background())
	}
	if ctrlBinDir != "" {
		_ = os.RemoveAll(ctrlBinDir)
	}
}

// ---- infrastructure helpers ----

func repoRoot() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		fail("failed to resolve current file path")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
}

func buildControllerBin() string {
	dir, err := os.MkdirTemp("", "e2e-controller")
	if err != nil {
		fail("failed to create temp dir: %v", err)
	}
	ctrlBinDir = dir
	bin := filepath.Join(dir, "rollout-controller.exe")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/rollout-controller")
	cmd.Dir = filepath.Join(repoRoot(), "rollout-controller")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fail("failed to build rollout-controller: %v\n%s", err, out)
	}
	return bin
}

func controllerEnv(port int, redisHost string, redisPort int, kafkaHost string, kafkaPort int) []string {
	return []string{
		"SHUTDOWN_TIMEOUT=10s",
		"SERVER_HOST=127.0.0.1",
		fmt.Sprintf("SERVER_PORT=%d", port),
		"SERVER_TIMEOUT=30s",
		"CACHE_HOST=" + redisHost,
		fmt.Sprintf("CACHE_PORT=%d", redisPort),
		"CACHE_READ_TIMEOUT=500ms",
		"CACHE_WRITE_TIMEOUT=500ms",
		"CACHE_CAMPAIGN_EVENT_ID_SEEN_TTL=10m",
		"BROKER_HOST=127.0.0.1",
		fmt.Sprintf("BROKER_PORT=%d", kafkaPort),
		"BROKER_BATCH_TIMEOUT=100ms",
		"BROKER_TIMEOUT=10s",
		"BROKER_GROUP_ID=" + controllerGroupID,
		"BROKER_MIN_BYTES=1",
		"BROKER_READER_MAX_WAIT=1s",
		"BROKER_COMMIT_TIMEOUT=2s",
		"BROKER_DLQ_TIMEOUT=5s",
		"BROKER_DLQ_MAX_ATTEMPTS=3",
		"EVALUATOR_FREQUENCY=1s",
		"EVALUATOR_REQUIRED_STABLE_CYCLES=3",
	}
}

func mappedPortInt(p interface{ Port() string }, err error) (int, error) {
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(p.Port())
}

func freePort() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fail("failed to allocate free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func newTestLogger() *zap.SugaredLogger {
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encCfg),
		zapcore.Lock(os.Stdout),
		zap.DebugLevel,
	)
	return zap.New(core, zap.AddStacktrace(zap.ErrorLevel)).Sugar()
}

func migrationsDirPath() string {
	return filepath.Join(repoRoot(), "migrations")
}

func applyMigrations(ctx context.Context, connStr string) {
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fail("failed to create migration pool: %v", err)
	}
	defer pool.Close()

	// The migrations rely on a uuidv7() function; provide a self-contained
	// implementation so the schema applies without the pg_uuidv7 extension.
	if _, err := pool.Exec(ctx, `CREATE OR REPLACE FUNCTION uuidv7() RETURNS uuid LANGUAGE sql AS $$ SELECT gen_random_uuid(); $$;`); err != nil {
		fail("failed to create uuidv7 function: %v", err)
	}

	dir := migrationsDirPath()
	entries, err := os.ReadDir(dir)
	if err != nil {
		fail("failed to read migrations dir %s: %v", dir, err)
	}

	var upFiles []string
	for _, e := range entries {
		if matched, _ := filepath.Match("*.up.sql", e.Name()); matched {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	for _, name := range upFiles {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			fail("failed to read migration %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			fail("migration %s failed: %v", name, err)
		}
	}
}

func waitForHTTPReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("http endpoint %s not ready within %s", url, timeout)
}

func waitForConnectHealth(baseURL string, exited <-chan error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case err := <-exited:
			return fmt.Errorf("rollout-controller exited before becoming ready: %v", err)
		default:
		}
		req, err := http.NewRequest(http.MethodPost, baseURL+"/health.v1.HealthService/CheckHealth", bytes.NewReader([]byte("{}")))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			resp, err := httpClient.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("connect health at %s not ready within %s", baseURL, timeout)
}

// ---- HTTP helpers ----

func doRequest(t *testing.T, method, path string, body any) (int, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, orchBaseURL+path, rdr)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, data
}

func createDevice(t *testing.T, model, version string) uuid.UUID {
	status, body := doRequest(t, http.MethodPost, "/api/v1/devices", map[string]string{
		"device_model":    model,
		"current_version": version,
	})
	require.Equal(t, http.StatusCreated, status, string(body))
	var resp deviceResp
	require.NoError(t, json.Unmarshal(body, &resp))
	return uuid.MustParse(resp.ID)
}

func createFirmware(t *testing.T, model, version, checksum, binaryURL string) uuid.UUID {
	status, body := doRequest(t, http.MethodPost, "/api/v1/firmware", map[string]string{
		"device_model": model,
		"fw_version":   version,
		"fw_checksum":  checksum,
		"binary_url":   binaryURL,
	})
	require.Equal(t, http.StatusCreated, status, string(body))
	var resp firmwareResp
	require.NoError(t, json.Unmarshal(body, &resp))
	return uuid.MustParse(resp.ID)
}

func twoStages() []map[string]any {
	return []map[string]any{
		{"order_index": 0, "target_percent": 100, "min_sample_size": 1, "success_threshold": 0.5},
		{"order_index": 1, "target_percent": 100, "min_sample_size": 1, "success_threshold": 0.5},
	}
}

func createCampaign(t *testing.T, fwID uuid.UUID, stages []map[string]any) (uuid.UUID, campaignResp) {
	status, body := doRequest(t, http.MethodPost, "/api/v1/campaigns", map[string]any{
		"firmware_version_id": fwID.String(),
		"rollout_stages":      stages,
	})
	require.Equal(t, http.StatusCreated, status, string(body))
	var resp campaignResp
	require.NoError(t, json.Unmarshal(body, &resp))
	return uuid.MustParse(resp.ID), resp
}

func startCampaign(t *testing.T, id uuid.UUID) {
	status, body := doRequest(t, http.MethodPost, "/api/v1/campaigns/"+id.String()+"/start", nil)
	require.Equal(t, http.StatusOK, status, string(body))
}

func checkin(t *testing.T, deviceID uuid.UUID, version string) checkinResp {
	status, body := doRequest(t, http.MethodPost, "/api/v1/devices/"+deviceID.String()+"/checkin", map[string]string{
		"current_version": version,
	})
	require.Equal(t, http.StatusOK, status, string(body))
	var resp checkinResp
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func report(t *testing.T, deviceID, campID, stageID uuid.UUID, result string) reportResp {
	status, body := doRequest(t, http.MethodPost, "/api/v1/devices/"+deviceID.String()+"/report", map[string]string{
		"campaign_id": campID.String(),
		"stage_id":    stageID.String(),
		"result":      result,
	})
	require.Equal(t, http.StatusOK, status, string(body))
	var resp reportResp
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func decommission(t *testing.T, deviceID uuid.UUID) {
	status, body := doRequest(t, http.MethodPost, "/api/v1/devices/"+deviceID.String()+"/decommission", nil)
	require.Equal(t, http.StatusOK, status, string(body))
}

func getCampaign(t *testing.T, id uuid.UUID) campaignResp {
	status, body := doRequest(t, http.MethodGet, "/api/v1/campaigns/"+id.String(), nil)
	require.Equal(t, http.StatusOK, status, string(body))
	var resp campaignResp
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func waitForStats(t *testing.T, campID uuid.UUID, pred func(statsResp) bool) statsResp {

	t.Helper()
	deadline := time.Now().Add(statsPollTimeout)
	var last statsResp
	for time.Now().Before(deadline) {
		camp := getCampaign(t, campID)
		if camp.Stats != nil {
			last = *camp.Stats
			if pred(*camp.Stats) {
				return *camp.Stats
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.FailNow(t, "stats predicate not satisfied within timeout", "last stats: %+v", last)
	return last
}

func produceUpdateResult(t *testing.T, eventID, deviceID, campID, stageID uuid.UUID, result string) {
	t.Helper()
	w := &kafka.Writer{
		Addr:         kafka.TCP(kafkaBrokerAddr),
		Topic:        updateResultsTopic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
	}
	defer w.Close()

	ev := domain.UpdateResultsEvent{
		EventID:    eventID,
		DeviceID:   deviceID,
		CampaignID: campID,
		StageID:    stageID,
		Result:     domain.UpdateAttemptsResult(result),
		Timestamp:  time.Now(),
	}
	val, err := json.Marshal(ev)
	require.NoError(t, err)
	err = w.WriteMessages(testCtx, kafka.Message{Key: campID[:], Value: val})
	require.NoError(t, err)
}

// ---- tests ----

func uniqueModel() string {
	return "sensor-" + uuid.New().String()[:8]
}

func TestE2E_FullCheckinReportStatsFlow(t *testing.T) {
	t.Parallel()
	model := uniqueModel()
	deviceID := createDevice(t, model, "0.0.1")
	fwID := createFirmware(t, model, "1.0.0", strings.Repeat("a", 64), "http://example.com/fw-1.0.0.bin")
	campID, _ := createCampaign(t, fwID, twoStages())
	startCampaign(t, campID)

	cr := checkin(t, deviceID, "0.0.1")
	require.True(t, cr.UpdateAvailable, "device on a lower version in a 100%% rollout should be offered an update")
	require.NotNil(t, cr.StageID, "checkin should return a stage_id when an update is available")
	require.NotEmpty(t, cr.BinaryUrl, "checkin should return the binary url")
	require.NotEmpty(t, cr.FWChecksum, "checkin should return the firmware checksum")

	stageID := uuid.MustParse(*cr.StageID)

	report(t, deviceID, campID, stageID, "success")
	waitForStats(t, campID, func(s statsResp) bool { return s.SampleSize >= 1 })

	report(t, deviceID, campID, stageID, "failure")
	finalStats := waitForStats(t, campID, func(s statsResp) bool { return s.SampleSize >= 2 })
	require.GreaterOrEqual(t, finalStats.SampleSize, 2, "stats should reflect both the success and failure reports")
}

func TestE2E_DecommissionedDevice_DoesNotReceiveUpdate(t *testing.T) {
	t.Parallel()
	model := uniqueModel()
	deviceID := createDevice(t, model, "0.0.1")
	fwID := createFirmware(t, model, "1.0.0", strings.Repeat("b", 64), "http://example.com/fw-1.0.0.bin")
	campID, _ := createCampaign(t, fwID, twoStages())
	startCampaign(t, campID)

	decommission(t, deviceID)

	cr := checkin(t, deviceID, "0.0.1")
	require.False(t, cr.UpdateAvailable, "a decommissioned device must not be offered an update")
	require.Nil(t, cr.StageID)
}

func TestE2E_DuplicateReportEvent_DoesNotDoubleCount(t *testing.T) {
	t.Parallel()
	model := uniqueModel()
	deviceID := createDevice(t, model, "0.0.1")
	fwID := createFirmware(t, model, "1.0.0", strings.Repeat("c", 64), "http://example.com/fw-1.0.0.bin")
	// Выский min_sample_size: evaluator не должен принять решение и сдвинуть
	// кампанию, пока тест проверяет дедупликацию счётчиков.
	noDecisionStages := []map[string]any{
		{"order_index": 0, "target_percent": 100, "min_sample_size": 100, "success_threshold": 0.5},
		{"order_index": 1, "target_percent": 100, "min_sample_size": 100, "success_threshold": 0.5},
	}
	campID, camp := createCampaign(t, fwID, noDecisionStages)
	startCampaign(t, campID)

	stageID := uuid.MustParse(camp.RolloutStages[0].ID)
	eventID := uuid.New()

	// The same event_id delivered twice must be de-duplicated by the controller.
	produceUpdateResult(t, eventID, deviceID, campID, stageID, "success")
	produceUpdateResult(t, eventID, deviceID, campID, stageID, "success")

	// Wait until the controller processes the event (sample size reaches 1).
	waitForStats(t, campID, func(s statsResp) bool { return s.SampleSize >= 1 })

	// Poll for a short grace window and assert the duplicate never increments.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		c := getCampaign(t, campID)
		if c.Stats != nil {
			require.LessOrEqual(t, c.Stats.SampleSize, 1, "a duplicate event_id must not double count the sample size")
		}
		time.Sleep(300 * time.Millisecond)
	}
	final := getCampaign(t, campID)
	require.NotNil(t, final.Stats, "campaign stats should be present")
	require.Equal(t, 1, final.Stats.SampleSize, "a duplicate event_id must not double count the sample size")
}

func stageByOrder(t *testing.T, camp campaignResp, orderIndex int) stageResp {
	t.Helper()
	for _, s := range camp.RolloutStages {
		if s.OrderIndex == orderIndex {
			return s
		}
	}
	require.FailNow(t, "stage not found", "campaign: %s order_index: %d", camp.ID, orderIndex)
	return stageResp{}
}

func waitForActiveStage(t *testing.T, campID, stageID uuid.UUID) {
	t.Helper()
	deadline := time.Now().Add(statsPollTimeout)
	for time.Now().Before(deadline) {
		camp := getCampaign(t, campID)
		for _, s := range camp.RolloutStages {
			if s.ID == stageID.String() && s.Status == "active" {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.FailNow(t, "stage did not become active within timeout", "campaign: %s stage: %s", campID, stageID)
}

func waitForCampaignStatus(t *testing.T, campID uuid.UUID, status string) {
	t.Helper()
	deadline := time.Now().Add(statsPollTimeout)
	var last string
	for time.Now().Before(deadline) {
		camp := getCampaign(t, campID)
		last = camp.Status
		if camp.Status == status {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.FailNow(t, "campaign did not reach status within timeout", "campaign: %s want: %s last: %s", campID, status, last)
}

func TestE2E_EvaluatorAdvancesAndCompletesOnSuccess(t *testing.T) {
	t.Parallel()
	model := uniqueModel()
	deviceID := createDevice(t, model, "0.0.1")
	fwID := createFirmware(t, model, "1.0.0", strings.Repeat("d", 64), "http://example.com/fw-1.0.0.bin")
	campID, camp := createCampaign(t, fwID, twoStages())
	startCampaign(t, campID)

	stage0ID := uuid.MustParse(stageByOrder(t, camp, 0).ID)
	stage1ID := uuid.MustParse(stageByOrder(t, camp, 1).ID)

	cr := checkin(t, deviceID, "0.0.1")
	require.True(t, cr.UpdateAvailable)
	require.NotNil(t, cr.StageID)
	require.Equal(t, stage0ID.String(), *cr.StageID)

	// Success above the threshold must promote the campaign to stage 1:
	// evaluator -> rollout.decisions -> ApplyDecision -> redis projection.
	report(t, deviceID, campID, stage0ID, "success")
	waitForActiveStage(t, campID, stage1ID)

	cr = checkin(t, deviceID, "0.0.1")
	require.True(t, cr.UpdateAvailable, "checkin must offer the update from the new active stage")
	require.NotNil(t, cr.StageID)
	require.Equal(t, stage1ID.String(), *cr.StageID)

	// Success on the last stage must complete the campaign (no next stage).
	report(t, deviceID, campID, stage1ID, "success")
	waitForCampaignStatus(t, campID, "completed")

	cr = checkin(t, deviceID, "0.0.1")
	require.False(t, cr.UpdateAvailable, "a completed campaign must not offer updates")
	require.Nil(t, cr.StageID)
}

func TestE2E_EvaluatorRollsBackOnFailure(t *testing.T) {
	t.Parallel()
	model := uniqueModel()
	deviceID := createDevice(t, model, "0.0.1")
	fwID := createFirmware(t, model, "1.0.0", strings.Repeat("e", 64), "http://example.com/fw-1.0.0.bin")
	campID, camp := createCampaign(t, fwID, twoStages())
	startCampaign(t, campID)

	stage0ID := uuid.MustParse(stageByOrder(t, camp, 0).ID)

	cr := checkin(t, deviceID, "0.0.1")
	require.True(t, cr.UpdateAvailable)
	require.NotNil(t, cr.StageID)

	// Failure below the threshold must roll the campaign back.
	report(t, deviceID, campID, stage0ID, "failure")
	waitForCampaignStatus(t, campID, "rolled_back")

	cr = checkin(t, deviceID, "0.0.1")
	require.False(t, cr.UpdateAvailable, "a rolled back campaign must not offer updates")
	require.Nil(t, cr.StageID)
}
