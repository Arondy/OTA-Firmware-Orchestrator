package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type UpdateResultsProducer struct {
	writer  *kafka.Writer
	timeout time.Duration
}

func NewUpdateResultsProducer(logger *zap.SugaredLogger, config config.BrokerConfig) (*UpdateResultsProducer, error) {
	addr := kafka.TCP(net.JoinHostPort(config.Host, strconv.Itoa(config.Port)))
	logger.Debugf("Connecting to UpdateResultsProducer on %s", addr.String())

	if err := Ping(addr.String(), config.Timeout); err != nil {
		return nil, err
	}

	writer := &kafka.Writer{
		Addr:         addr,
		Topic:        config.Topic,
		Balancer:     &kafka.Hash{},
		BatchTimeout: config.BatchTimeout,
		RequiredAcks: kafka.RequireOne,
	}

	p := &UpdateResultsProducer{
		writer:  writer,
		timeout: config.Timeout,
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
