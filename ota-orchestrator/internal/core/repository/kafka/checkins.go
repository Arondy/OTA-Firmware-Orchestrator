package kafka

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

const checkinsTopic = "device.checkins"

type CheckinsProducer struct {
	writer  *kafka.Writer
	buffer  chan domain.CheckinEvent
	done    chan struct{}
	timeout time.Duration
	logger  *zap.SugaredLogger
}

func NewCheckinsProducer(logger *zap.SugaredLogger, config config.BrokerConfig) (*CheckinsProducer, error) {
	addr := kafka.TCP(net.JoinHostPort(config.Host, strconv.Itoa(config.Port)))
	logger.Debugf("Connecting to CheckinsProducer on %s", addr.String())

	if err := Ping(addr.String()); err != nil {
		return nil, err
	}

	writer := &kafka.Writer{
		Addr:         addr,
		Topic:        checkinsTopic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireNone,
	}

	p := &CheckinsProducer{
		writer:  writer,
		buffer:  make(chan domain.CheckinEvent, config.BufferSize),
		timeout: config.Timeout,
		done:    make(chan struct{}, 1),
		logger:  logger,
	}

	go p.run()
	return p, nil
}

func (p *CheckinsProducer) run() {
	for event := range p.buffer {
		value, err := json.Marshal(event)
		if err != nil {
			p.logger.Errorw("failed to marshal event", "error", err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
		err = p.writer.WriteMessages(ctx, kafka.Message{
			Key:   event.DeviceID[:],
			Value: value,
			Time:  event.Timestamp,
		})
		if err != nil {
			p.logger.Errorw("failed to publish checkin event", "error", err)
		}

		cancel()
	}
	p.done <- struct{}{}
}

func (p *CheckinsProducer) Produce(event domain.CheckinEvent) {
	select {
	case p.buffer <- event:
	default:
		p.logger.Warnw("device buffer is full, discarding event", "device_id", event.DeviceID, "campaign_id", event.CampaignID)
	}
}

func (p *CheckinsProducer) Close() {
	close(p.buffer)
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	select {
	case <-p.done:
	case <-ctx.Done():
		p.logger.Warn("device producer: graceful shutdown timed out, forcing it")
	}

	p.writer.Close()
}
