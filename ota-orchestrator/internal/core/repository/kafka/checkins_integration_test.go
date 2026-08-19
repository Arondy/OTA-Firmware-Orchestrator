//go:build integration

package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestCheckinsProducer_PublishesEvent(t *testing.T) {
	createTopic(t, checkinsTopic, 3)

	host, port := brokerHostPort()
	producer, err := NewCheckinsProducer(zap.NewNop().Sugar(), config.BrokerConfig{
		Host:         host,
		Port:         port,
		BatchTimeout: time.Second,
		Timeout:      10 * time.Second,
		BufferSize:   16,
	})
	require.NoError(t, err)
	defer producer.Close()

	event := domain.CheckinEvent{
		DeviceID:       uuid.New(),
		DeviceModel:    "model-x",
		CurrentVersion: "1.0.0",
		CampaignID:     uuid.New(),
		Timestamp:      time.Now().UTC(),
	}

	producer.Produce(event)

	reader := consumerReader(checkinsTopic)
	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msg, err := reader.ReadMessage(ctx)
	require.NoError(t, err)

	require.Equal(t, event.DeviceID[:], msg.Key)

	var got domain.CheckinEvent
	require.NoError(t, json.Unmarshal(msg.Value, &got))
	require.Equal(t, event.DeviceID, got.DeviceID)
	require.Equal(t, event.CampaignID, got.CampaignID)
	require.Equal(t, event.DeviceModel, got.DeviceModel)
	require.Equal(t, event.CurrentVersion, got.CurrentVersion)
}

func TestCheckinsProducer_BufferFull(t *testing.T) {
	createTopic(t, checkinsTopic, 3)

	core, logs := observer.New(zapcore.WarnLevel)
	logger := zap.New(core).Sugar()

	host, port := brokerHostPort()
	producer, err := NewCheckinsProducer(logger, config.BrokerConfig{
		Host:         host,
		Port:         port,
		BatchTimeout: time.Second,
		Timeout:      10 * time.Second,
		BufferSize:   4,
	})
	require.NoError(t, err)

	const total = 200
	start := time.Now()
	for i := 0; i < total; i++ {
		producer.Produce(domain.CheckinEvent{
			DeviceID:       uuid.New(),
			DeviceModel:    "model-x",
			CurrentVersion: "1.0.0",
			CampaignID:     uuid.New(),
			Timestamp:      time.Now().UTC(),
		})
	}
	elapsed := time.Since(start)

	dropped := logs.FilterMessage("device buffer is full, discarding event").Len()
	require.Greater(t, dropped, 0, "saturated producer must drop events instead of blocking")
	require.Less(t, elapsed, 2*time.Second, "Produce must stay non-blocking under saturation")

	producer.Close()

	reader := consumerReader(checkinsTopic)
	defer reader.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	delivered := 0
	for delivered < total {
		if _, err := reader.ReadMessage(ctx); err != nil {
			break
		}
		delivered++
	}
	require.Greater(t, delivered, 0, "events that fit the buffer must still be delivered")
}

func TestCheckinsProducer_Close(t *testing.T) {
	createTopic(t, checkinsTopic, 3)

	host, port := brokerHostPort()
	producer, err := NewCheckinsProducer(zap.NewNop().Sugar(), config.BrokerConfig{
		Host:         host,
		Port:         port,
		BatchTimeout: time.Second,
		Timeout:      10 * time.Second,
		BufferSize:   16,
	})
	require.NoError(t, err)

	event := domain.CheckinEvent{
		DeviceID:       uuid.New(),
		DeviceModel:    "model-x",
		CurrentVersion: "1.0.0",
		CampaignID:     uuid.New(),
		Timestamp:      time.Now().UTC(),
	}
	producer.Produce(event)

	done := make(chan struct{})
	go func() {
		producer.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("Close blocked past timeout")
	}

	reader := consumerReader(checkinsTopic)
	defer reader.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	found := false
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			break
		}
		if string(msg.Key) == string(event.DeviceID[:]) {
			found = true
			break
		}
	}
	require.True(t, found, "Close must flush buffered events before returning")
}
