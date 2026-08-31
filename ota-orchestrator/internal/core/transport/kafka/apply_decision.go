package core_kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type RolloutDecisionsSvc interface {
	ApplyDecision(ctx context.Context, decision domain.DecisionEvent) error
}

type RolloutDecisionsConsumer struct {
	reader    *kafka.Reader
	dlqWriter *kafka.Writer
	svc       RolloutDecisionsSvc
	logger    *zap.SugaredLogger
}

func NewRolloutDecisionsConsumer(svc RolloutDecisionsSvc, config config.BrokerConfig, logger *zap.SugaredLogger) (*RolloutDecisionsConsumer, error) {
	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	logger.Debugf("Connecting to RolloutDecisionsConsumer on %s", addr)

	if err := Ping(addr); err != nil {
		return nil, err
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{addr},
		GroupID:  config.GroupID,
		Topic:    config.Topic,
		MinBytes: config.MinBytes,
		MaxBytes: 1 << 20,
		MaxWait:  1 * time.Second,
	})

	dlqWriter := &kafka.Writer{
		Addr:         kafka.TCP(addr),
		Topic:        config.Topic + ".dlq",
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
		WriteTimeout: 5 * time.Second,
		MaxAttempts:  3,
	}

	p := &RolloutDecisionsConsumer{
		reader:    reader,
		dlqWriter: dlqWriter,
		svc:       svc,
		logger:    logger,
	}
	return p, nil
}

func (c *RolloutDecisionsConsumer) Run(ctx context.Context) error {
	for {
		message, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("failed to fetch event: %w", err)
		}

		logger := c.logger.With("event_time", message.Time, "event_offset", message.Offset)

		var event domain.DecisionEvent
		if err = json.Unmarshal(message.Value, &event); err != nil {
			logger.Errorw("failed to unmarshal event", "error", err)
			go c.writeToDLQ(ctx, message, err, false)

			commitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err = c.reader.CommitMessages(commitCtx, message)
			cancel()
			if err != nil {
				logger.Errorw("failed to commit corrupted message", "error", err)
			}
			continue
		}

		err = c.svc.ApplyDecision(ctx, event)
		if err != nil {
			logger.Errorw("failed to update stage results", "error", err)
			go c.writeToDLQ(ctx, message, err, true)
		}
		c.logger.Debugw("applied rollout decision", "stage_id", event.PreviousStageID, "decision", event.DecisionType)

		commitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err = c.reader.CommitMessages(commitCtx, message)
		cancel()
		if err != nil {
			logger.Errorw("failed to commit message", "error", err)
		}
	}
}

func (c *RolloutDecisionsConsumer) writeToDLQ(ctx context.Context, message kafka.Message, err error, retryable bool) {
	headers := []kafka.Header{
		{Key: "x-retryable", Value: []byte(strconv.FormatBool(retryable))},
		{Key: "x-error", Value: []byte(err.Error())},
		{Key: "x-failed-at", Value: []byte(time.Now().Format(time.RFC3339))},
	}

	dlqMessage := kafka.Message{
		Key:     message.Key,
		Value:   message.Value,
		Time:    message.Time,
		Headers: headers,
	}

	if err := c.dlqWriter.WriteMessages(ctx, dlqMessage); err != nil {
		c.logger.Errorw("failed to send message to DLQ", "error", err, "event_time", message.Time, "event_offset", message.Offset)
	}
}

func (c *RolloutDecisionsConsumer) Close() {
	if err := c.reader.Close(); err != nil {
		c.logger.Errorw("failed to close reader", "error", err)
	}
	if err := c.dlqWriter.Close(); err != nil {
		c.logger.Errorw("failed to close dlqWriter", "error", err)
	}
}
