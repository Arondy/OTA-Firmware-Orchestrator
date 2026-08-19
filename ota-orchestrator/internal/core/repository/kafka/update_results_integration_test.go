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

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateResultsProducer_PublishesEvent(t *testing.T) {
	t.Parallel()
	topic := uniqueTopic("update-results")
	createTopic(t, topic, 3)

	host, port := brokerHostPort()
	producer, err := NewUpdateResultsProducer(zap.NewNop().Sugar(), config.BrokerConfig{
		Host:         host,
		Port:         port,
		BatchTimeout: time.Second,
		Timeout:      10 * time.Second,
		BufferSize:   16,
		Topic:        topic,
	})
	require.NoError(t, err)
	defer producer.Close()

	event := domain.UpdateResultsEvent{
		EventID:    uuid.New(),
		DeviceID:   uuid.New(),
		CampaignID: uuid.New(),
		StageID:    uuid.New(),
		Result:     domain.UpdateAttemptsResultSuccess,
		Timestamp:  time.Now().UTC(),
	}

	require.NoError(t, producer.Produce(event))

	msg, ok := readMatchingFromTopic(t, topic, 30*time.Second, func(m kafka.Message) bool {
		return string(m.Key) == string(event.CampaignID[:])
	})
	require.True(t, ok, "produced event must be readable")
	require.Equal(t, event.CampaignID[:], msg.Key)

	var got domain.UpdateResultsEvent
	require.NoError(t, json.Unmarshal(msg.Value, &got))
	require.Equal(t, event.EventID, got.EventID)
	require.Equal(t, event.CampaignID, got.CampaignID)
	require.Equal(t, event.StageID, got.StageID)
	require.Equal(t, event.Result, got.Result)
}

func TestUpdateResultsProducer_Close(t *testing.T) {
	t.Parallel()
	topic := uniqueTopic("update-results")
	createTopic(t, topic, 3)

	host, port := brokerHostPort()
	producer, err := NewUpdateResultsProducer(zap.NewNop().Sugar(), config.BrokerConfig{
		Host:         host,
		Port:         port,
		BatchTimeout: time.Second,
		Timeout:      10 * time.Second,
		BufferSize:   16,
		Topic:        topic,
	})
	require.NoError(t, err)

	event := domain.UpdateResultsEvent{
		EventID:    uuid.New(),
		DeviceID:   uuid.New(),
		CampaignID: uuid.New(),
		StageID:    uuid.New(),
		Result:     domain.UpdateAttemptsResultSuccess,
		Timestamp:  time.Now().UTC(),
	}
	require.NoError(t, producer.Produce(event))

	require.NotPanics(t, func() { producer.Close() })

	reader := consumerReader(topic)
	defer reader.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	found := false
	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			break
		}
		var got domain.UpdateResultsEvent
		if json.Unmarshal(msg.Value, &got) == nil && got.EventID == event.EventID {
			found = true
			break
		}
	}
	require.True(t, found, "produced event must be readable after Close")
}
