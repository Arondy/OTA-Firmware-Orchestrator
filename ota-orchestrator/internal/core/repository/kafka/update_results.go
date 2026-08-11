package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/domain"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

const updateResultsTopic = "firmware.update-results"

type UpdateResultsProducer struct {
	writer  *kafka.Writer
	timeout time.Duration
	logger  *zap.SugaredLogger
}

func NewUpdateResultsProducer(logger *zap.SugaredLogger, config config.BrokerConfig) (*UpdateResultsProducer, error) {
	addr := kafka.TCP(fmt.Sprintf("%s:%d", config.Host, config.Port))
	if err := Ping(addr.String()); err != nil {
		return nil, err
	}

	updateResults := &kafka.Writer{
		Addr:         addr,
		Topic:        updateResultsTopic,
		Balancer:     &kafka.Hash{},
		BatchTimeout: config.BatchTimeout,
		RequiredAcks: kafka.RequireOne,
	}

	p := &UpdateResultsProducer{
		writer:  updateResults,
		timeout: config.Timeout,
		logger:  logger,
	}
	return p, nil
}

func (p *UpdateResultsProducer) Produce(event domain.UpdateResultsEvent) error {
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   event.CampaignID[:],
		Value: value,
		Time:  event.Timestamp,
	})
	if err != nil {
		return fmt.Errorf("failed to publish update results event: %w", err)
	}

	return nil
}

func (p *UpdateResultsProducer) Close() {
	p.writer.Close()
}
