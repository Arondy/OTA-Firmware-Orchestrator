package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type UpdateResultsSvc interface {
	UpdateStageResults(ctx context.Context, event domain.UpdateResultsEvent) (int, error)
}

type UpdateResultsConsumer struct {
	reader    *kafka.Reader
	dlqWriter *kafka.Writer
	svc       UpdateResultsSvc
	logger    *zap.SugaredLogger
}

func NewUpdateResultsConsumer(svc UpdateResultsSvc, config config.BrokerConfig, logger *zap.SugaredLogger) (*UpdateResultsConsumer, error) {
	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	logger.Debugf("Connecting to UpdateResultsConsumer on %s", addr)

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

	p := &UpdateResultsConsumer{
		reader:    reader,
		dlqWriter: dlqWriter,
		svc:       svc,
		logger:    logger,
	}
	return p, nil
}

func (p *UpdateResultsConsumer) Run(ctx context.Context) error {
	for {
		message, err := p.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("failed to fetch event: %w", err)
		}

		logger := p.logger.With("event_time", message.Time, "event_offset", message.Offset)

		var event domain.UpdateResultsEvent
		if err = json.Unmarshal(message.Value, &event); err != nil {
			logger.Errorw("failed to unmarshal event", "error", err)
			go p.writeToDLQ(ctx, message, err, false)

			commitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err = p.reader.CommitMessages(commitCtx, message)
			cancel()
			if err != nil {
				logger.Errorw("failed to commit corrupted message", "error", err)
			}
			continue
		}

		updatedCount, err := p.svc.UpdateStageResults(ctx, event)
		if err != nil {
			logger.Errorw("failed to update stage results", "error", err)
			go p.writeToDLQ(ctx, message, err, true)
		}
		p.logger.Debugw("updated stage results", "stage_id", event.StageID, "result", event.Result, "updated", updatedCount)

		commitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err = p.reader.CommitMessages(commitCtx, message)
		cancel()
		if err != nil {
			logger.Errorw("failed to commit message", "error", err)
		}
	}
}

func (p *UpdateResultsConsumer) writeToDLQ(ctx context.Context, message kafka.Message, err error, retryable bool) {
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

	if err := p.dlqWriter.WriteMessages(ctx, dlqMessage); err != nil {
		p.logger.Errorw("failed to send message to DLQ", "error", err, "event_time", message.Time, "event_offset", message.Offset)
	}
}

func (p *UpdateResultsConsumer) Close() {
	if err := p.reader.Close(); err != nil {
		p.logger.Errorw("failed to close reader", "error", err)
	}
	if err := p.dlqWriter.Close(); err != nil {
		p.logger.Errorw("failed to close dlqWriter", "error", err)
	}
}
