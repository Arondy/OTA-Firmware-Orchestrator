package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/rollout-controller/internal/core/domain"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type RolloutDecisionsProducer struct {
	writer  *kafka.Writer
	timeout time.Duration
}

func NewRolloutDecisionsProducer(config config.BrokerConfig, logger *zap.SugaredLogger) (*RolloutDecisionsProducer, error) {
	addr := kafka.TCP(net.JoinHostPort(config.Host, strconv.Itoa(config.Port)))
	logger.Debugf("Connecting to RolloutDecisionsProducer on %s", addr.String())

	if err := Ping(addr.String()); err != nil {
		return nil, err
	}

	writer := &kafka.Writer{
		Addr:         addr,
		Topic:        config.Topic,
		Balancer:     &kafka.Hash{},
		BatchTimeout: config.BatchTimeout,
		RequiredAcks: kafka.RequireOne,
	}

	p := &RolloutDecisionsProducer{
		writer:  writer,
		timeout: config.Timeout,
	}
	return p, nil
}

func (p *RolloutDecisionsProducer) Produce(event domain.DecisionEvent) error {
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
		return fmt.Errorf("failed to publish rollout decisions event: %w", err)
	}

	return nil
}

func (p *RolloutDecisionsProducer) Close() {
	p.writer.Close()
}
