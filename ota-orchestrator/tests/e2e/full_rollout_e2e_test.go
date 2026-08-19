//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	updateResultsTopic    = "firmware.update-results"
	checkinsTopic         = "device.checkins"
	updateResultsDLQTopic = "firmware.update-results.dlq"

	// Match docker-compose.yml image versions.
	kafkaImage    = "apache/kafka:4.3.1"
	postgresImage = "postgres:18-alpine"
	redisImage    = "redis:8-alpine"
	kafkaExtTCP   = "9093/tcp"

	statsPollTimeout     = 90 * time.Second
	readinessPollTimeout = 150 * time.Second
	controllerGroupID    = "e2e-test-group"
)

// kafkaStarterScript is copied into the KRaft container once the mapped port is
// known; the advertised listener must point at the host-reachable endpoint.
const kafkaStarterScript = `#!/bin/bash
export KAFKA_ADVERTISED_LISTENERS="PLAINTEXT://%s,BROKER://localhost:9092"
echo "Starting Kafka KRaft mode (apache/kafka:4.3.1)"
/etc/kafka/docker/run`

var (
	orchBaseURL     string
	ctrlBaseURL     string
	httpClient      *http.Client
	kafkaBrokerAddr string
	testCtx         context.Context
	cancelAll       context.CancelFunc
	controllerCmd   *exec.Cmd
	kafkaContainer  testcontainers.Container
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
	defer cancel()

	testCtx = ctx
	logger := newTestLogger()

	pgC, err := postgres.Run(ctx, postgresImage,
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("failed to start postgres: %v", err)
	}

	redisC, err := redis.Run(ctx, redisImage)
	if err != nil {
		log.Fatalf("failed to start redis: %v", err)
	}

	kafkaC, kafkaPort, err := startKafkaContainer(ctx)
	if err != nil {
		log.Fatalf("failed to start kafka: %v", err)
	}
	kafkaContainer = kafkaC

	dbHost, err := pgC.Host(ctx)
	if err != nil {
		log.Fatalf("postgres host: %v", err)
	}
	dbPort, err := mappedPortInt(pgC.MappedPort(ctx, "5432/tcp"))
	if err != nil {
		log.Fatalf("postgres port: %v", err)
	}

	redisHost, err := redisC.Host(ctx)
	if err != nil {
		log.Fatalf("redis host: %v", err)
	}
	redisPort, err := mappedPortInt(redisC.MappedPort(ctx, "6379/tcp"))
	if err != nil {
		log.Fatalf("redis port: %v", err)
	}

	kafkaHost := "127.0.0.1"
	kafkaBrokerAddr = net.JoinHostPort(kafkaHost, strconv.Itoa(kafkaPort))

	httpClient = &http.Client{Timeout: 20 * time.Second}

	connStr := fmt.Sprintf("postgres://test:test@%s:%d/testdb?sslmode=disable", dbHost, dbPort)
	applyMigrations(ctx, connStr)
	createKafkaTopics(ctx, kafkaBrokerAddr)
	waitForKafkaBroker(kafkaBrokerAddr)

	ctrlPort := freePort()
	ctrlBin := buildControllerBin()
	ctrlCmd := exec.Command(ctrlBin)
	ctrlCmd.Dir = filepath.Dir(ctrlBin)
	ctrlCmd.Env = append(os.Environ(), controllerEnv(ctrlPort, redisHost, redisPort, kafkaHost, kafkaPort)...)
	ctrlCmd.Stdout = os.Stdout
	ctrlCmd.Stderr = os.Stderr
	if err := ctrlCmd.Start(); err != nil {
		log.Fatalf("failed to start rollout-controller: %v", err)
	}
	controllerCmd = ctrlCmd
	ctrlBaseURL = fmt.Sprintf("http://127.0.0.1:%d", ctrlPort)
	if err := waitForConnectHealth(ctrlBaseURL, readinessPollTimeout); err != nil {
		log.Fatalf("rollout-controller not ready: %v", err)
	}

	orchPort := freePort()
	orchCfg := &orchestratorconfig.Config{
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
			Host:                    redisHost,
			Port:                    redisPort,
			DeviceLastSeenTTL:       24 * time.Hour,
			DeviceCurrentVersionTTL: 24 * time.Hour,
		},
		Broker: orchestratorconfig.BrokerConfig{
			Host:         kafkaHost,
			Port:         kafkaPort,
			BatchTimeout: 100 * time.Millisecond,
			Timeout:      5 * time.Second,
			BufferSize:   1024,
		},
		RolloutController: orchestratorconfig.RolloutControllerConfig{
			Scheme:  "http",
			Host:    "127.0.0.1",
			Port:    ctrlPort,
			Timeout: 5 * time.Second,
		},
	}

	go func() {
		_ = orchestratorcore.Run(ctx, orchCfg, logger)
	}()

	orchBaseURL = fmt.Sprintf("http://127.0.0.1:%d", orchPort)
	httpClient = &http.Client{Timeout: 20 * time.Second}
	if err := waitForHTTPReady(orchBaseURL+"/healthz", readinessPollTimeout); err != nil {
		log.Fatalf("ota-orchestrator not ready: %v", err)
	}

	code := m.Run()

	cancel()
	if controllerCmd != nil && controllerCmd.Process != nil {
		_ = controllerCmd.Process.Kill()
		_, _ = controllerCmd.Process.Wait()
	}
	if kafkaContainer != nil {
		_ = kafkaContainer.Terminate(context.Background())
	}
	_ = pgC.Terminate(context.Background())
	_ = redisC.Terminate(context.Background())
	os.Exit(code)
}

// ---- infrastructure helpers ----

// startKafkaContainer launches a single-node KRaft broker (apache/kafka:4.3.1)
// and returns it together with the host-mapped external port. The advertised
// listener is rewritten to the mapped endpoint so host processes can connect.
func startKafkaContainer(ctx context.Context) (testcontainers.Container, int, error) {
	req := testcontainers.ContainerRequest{
		Image:        kafkaImage,
		ExposedPorts: []string{kafkaExtTCP},
		Env: map[string]string{
			"CLUSTER_ID":                                     "5L6g3nShT-eMCtK--X86sw",
			"KAFKA_NODE_ID":                                  "1",
			"KAFKA_PROCESS_ROLES":                            "broker,controller",
			"KAFKA_CONTROLLER_LISTENER_NAMES":                "CONTROLLER",
			"KAFKA_CONTROLLER_QUORUM_VOTERS":                 "1@localhost:9094",
			"KAFKA_LISTENERS":                                "PLAINTEXT://0.0.0.0:9093,BROKER://0.0.0.0:9092,CONTROLLER://0.0.0.0:9094",
			"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP":           "PLAINTEXT:PLAINTEXT,BROKER:PLAINTEXT,CONTROLLER:PLAINTEXT",
			"KAFKA_INTER_BROKER_LISTENER_NAME":               "BROKER",
			"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR":         "1",
			"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR": "1",
			"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR":            "1",
			"KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS":         "0",
			"KAFKA_NUM_PARTITIONS":                           "3",
			"KAFKA_AUTO_CREATE_TOPICS_ENABLE":                "false",
			"KAFKA_LOG_DIRS":                                 "/var/lib/kafka/data",
		},
		Cmd: []string{"sh", "-c", "while [ ! -f /tmp/kafka_start.sh ]; do sleep 0.1; done; bash /tmp/kafka_start.sh"},
		LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
			{
				PostStarts: []testcontainers.ContainerHook{
					func(ctx context.Context, c testcontainers.Container) error {
						if err := wait.ForMappedPort(kafkaExtTCP).WaitUntilReady(ctx, c); err != nil {
							return err
						}
						endpoint, err := c.PortEndpoint(ctx, kafkaExtTCP, "")
						if err != nil {
							return err
						}
						script := fmt.Sprintf(kafkaStarterScript, endpoint)
						return c.CopyToContainer(ctx, []byte(script), "/tmp/kafka_start.sh", 0o755)
					},
				},
			},
		},
		WaitingFor: wait.ForLog("Kafka Server started"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, 0, err
	}

	port, err := container.MappedPort(ctx, kafkaExtTCP)
	if err != nil {
		return nil, 0, err
	}
	portInt, err := strconv.Atoi(port.Port())
	if err != nil {
		return nil, 0, err
	}
	return container, portInt, nil
}

func repoRoot() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("failed to resolve current file path")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
}

func buildControllerBin() string {
	dir, err := os.MkdirTemp("", "e2e-controller")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}
	bin := filepath.Join(dir, "rollout-controller.exe")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/rollout-controller")
	cmd.Dir = filepath.Join(repoRoot(), "rollout-controller")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("failed to build rollout-controller: %v\n%s", err, out)
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
		"CACHE_CAMPAIGN_EVENT_ID_SEEN_TTL=10m",
		"BROKER_HOST=127.0.0.1",
		fmt.Sprintf("BROKER_PORT=%d", kafkaPort),
		"BROKER_GROUP_ID=" + controllerGroupID,
		"BROKER_MIN_BYTES=1",
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
		log.Fatalf("failed to allocate free port: %v", err)
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
		log.Fatalf("failed to create migration pool: %v", err)
	}
	defer pool.Close()

	// The migrations rely on a uuidv7() function; provide a self-contained
	// implementation so the schema applies without the pg_uuidv7 extension.
	if _, err := pool.Exec(ctx, `CREATE OR REPLACE FUNCTION uuidv7() RETURNS uuid LANGUAGE sql AS $$ SELECT gen_random_uuid(); $$;`); err != nil {
		log.Fatalf("failed to create uuidv7 function: %v", err)
	}

	dir := migrationsDirPath()
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("failed to read migrations dir %s: %v", dir, err)
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
			log.Fatalf("failed to read migration %s: %v", name, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			log.Fatalf("migration %s failed: %v", name, err)
		}
	}
}

// createKafkaTopics creates each topic and blocks until its partition leaders
// are elected, so producers/consumers observe stable metadata.
func createKafkaTopics(ctx context.Context, broker string) {
	client := &kafka.Client{Addr: kafka.TCP(broker)}
	for _, name := range []string{checkinsTopic, updateResultsTopic, updateResultsDLQTopic} {
		createOneTopic(ctx, client, name, 3)
	}
}

func createOneTopic(ctx context.Context, client *kafka.Client, name string, partitions int) {
	deadline := time.Now().Add(120 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		_, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{
			Topics: []kafka.TopicConfig{{
				Topic:             name,
				NumPartitions:     partitions,
				ReplicationFactor: 1,
			}},
		})
		if err == nil || errors.Is(err, kafka.TopicAlreadyExists) {
			if waitTopicLeaderReady(ctx, client, name, partitions) {
				return
			}
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	if lastErr != nil {
		log.Fatalf("failed to create kafka topic %q: %v", name, lastErr)
	}
}

// waitTopicLeaderReady polls broker metadata until the topic reports the
// expected partitions, each with an elected leader.
func waitTopicLeaderReady(ctx context.Context, client *kafka.Client, name string, partitions int) bool {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	backoff := 100 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return false
		default:
		}

		res, err := client.Metadata(ctx, &kafka.MetadataRequest{Topics: []string{name}})
		if err != nil {
			time.Sleep(backoff)
			if backoff < 2*time.Second {
				backoff *= 2
			}
			continue
		}

		var topic *kafka.Topic
		for i := range res.Topics {
			if res.Topics[i].Name == name {
				topic = &res.Topics[i]
				break
			}
		}
		if topic == nil || topic.Error != nil || len(topic.Partitions) != partitions {
			time.Sleep(backoff)
			if backoff < 2*time.Second {
				backoff *= 2
			}
			continue
		}

		ready := true
		for _, p := range topic.Partitions {
			if p.Leader.ID < 0 {
				ready = false
				break
			}
		}
		if ready {
			return true
		}
		time.Sleep(backoff)
		if backoff < 2*time.Second {
			backoff *= 2
		}
	}
}

func waitForPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("port %s not ready within %s", addr, timeout)
}

func waitForKafkaBroker(addr string) {
	deadline := time.Now().Add(120 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := kafka.Dial("tcp", addr)
		if err == nil {
			_ = conn.Close()
			return
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}
	log.Fatalf("kafka broker %s not reachable: %v", addr, lastErr)
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

func waitForConnectHealth(baseURL string, timeout time.Duration) error {
	if err := waitForPort(strings.TrimPrefix(baseURL, "http://"), timeout); err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodPost, baseURL+"/health.v1.HealthService/CheckHealth", bytes.NewReader([]byte("{}")))
		if err != nil {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
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
	model := uniqueModel()
	deviceID := createDevice(t, model, "0.0.1")
	fwID := createFirmware(t, model, "1.0.0", strings.Repeat("c", 64), "http://example.com/fw-1.0.0.bin")
	campID, camp := createCampaign(t, fwID, twoStages())
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
