//go:build integration

package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/google/uuid"
	"sync"
	"testing"
	"time"

	tkafka "github.com/Arondy/OTA-Firmware-Orchestrator/testutil/kafka"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeUpdateResultsSvc struct {
	mu       sync.Mutex
	received []domain.UpdateResultsEvent
	err      error
}

func (f *fakeUpdateResultsSvc) UpdateStageResults(ctx context.Context, event domain.UpdateResultsEvent) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.received = append(f.received, event)
	return 1, f.err
}

func (f *fakeUpdateResultsSvc) assertNotReceived(t *testing.T, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		got := len(f.received)
		f.mu.Unlock()
		require.Zero(t, got, "corrupted message must not be passed to the service")
		time.Sleep(50 * time.Millisecond)
	}
}

func (f *fakeUpdateResultsSvc) waitReceived(t *testing.T, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		got := len(f.received)
		f.mu.Unlock()
		if got >= n {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("svc did not receive %d events within %s", n, timeout)
}

func newTestConsumer(t *testing.T, svc UpdateResultsSvc) (*UpdateResultsConsumer, func()) {
	t.Helper()

	tkafka.DeleteTopic(updateResultsTopic)
	tkafka.DeleteTopic(updateResultsDLQTopic)
	createTopic(t, updateResultsTopic, 3)
	createTopic(t, updateResultsDLQTopic, 3)

	host, port := brokerHostPort()
	consumer, err := NewUpdateResultsConsumer(svc, config.BrokerConfig{
		Host:     host,
		Port:     port,
		GroupID:  "integration-consumer-group-" + uuid.New().String(),
		MinBytes: 1,
	}, zap.NewNop().Sugar())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = consumer.Run(ctx) }()

	return consumer, func() {
		cancel()
		consumer.Close()
	}
}

func TestUpdateResultsConsumer_ProcessesEvent(t *testing.T) {

	svc := &fakeUpdateResultsSvc{}
	_, stop := newTestConsumer(t, svc)
	defer stop()

	event := domain.UpdateResultsEvent{
		EventID:    uuid.New(),
		DeviceID:   uuid.New(),
		CampaignID: uuid.New(),
		StageID:    uuid.New(),
		Result:     domain.UpdateAttemptsResultSuccess,
		Timestamp:  time.Now().UTC(),
	}
	value, err := json.Marshal(event)
	require.NoError(t, err)
	produceMessage(t, updateResultsTopic, event.CampaignID[:], value)

	svc.waitReceived(t, 1, 30*time.Second)

	svc.mu.Lock()
	got := svc.received[0]
	svc.mu.Unlock()

	require.Equal(t, event.EventID, got.EventID)
	require.Equal(t, event.CampaignID, got.CampaignID)
	require.Equal(t, event.StageID, got.StageID)
	require.Equal(t, event.Result, got.Result)
}

func TestUpdateResultsConsumer_DeadLetterOnServiceError(t *testing.T) {

	svc := &fakeUpdateResultsSvc{err: errors.New("boom")}
	_, stop := newTestConsumer(t, svc)
	defer stop()

	event := domain.UpdateResultsEvent{
		EventID:    uuid.New(),
		DeviceID:   uuid.New(),
		CampaignID: uuid.New(),
		StageID:    uuid.New(),
		Result:     domain.UpdateAttemptsResultSuccess,
		Timestamp:  time.Now().UTC(),
	}
	value, err := json.Marshal(event)
	require.NoError(t, err)
	produceMessage(t, updateResultsTopic, event.CampaignID[:], value)

	svc.waitReceived(t, 1, 30*time.Second)

	msg, ok := readMatchingFromTopic(t, updateResultsDLQTopic, 20*time.Second, func(m kafka.Message) bool {
		var got domain.UpdateResultsEvent
		return json.Unmarshal(m.Value, &got) == nil && got.EventID == event.EventID
	})
	require.True(t, ok, "event that fails the service must be written to DLQ")

	require.Equal(t, "true", headerValue(msg, "x-retryable"))
	require.Contains(t, headerValue(msg, "x-error"), "boom")
}

func TestUpdateResultsConsumer_DeadLetterOnCorruptedMessage(t *testing.T) {

	svc := &fakeUpdateResultsSvc{}
	_, stop := newTestConsumer(t, svc)
	defer stop()

	id := uuid.New()
	key := id[:]
	payload := []byte("not-json-at-all")
	produceMessage(t, updateResultsTopic, key, payload)

	msg, ok := readMatchingFromTopic(t, updateResultsDLQTopic, 20*time.Second, func(m kafka.Message) bool {
		return string(m.Value) == string(payload)
	})
	require.True(t, ok, "corrupted message must be written to DLQ")

	require.Equal(t, "false", headerValue(msg, "x-retryable"))

	svc.assertNotReceived(t, 2*time.Second)
}

