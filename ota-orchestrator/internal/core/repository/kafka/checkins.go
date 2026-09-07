package kafka

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type CheckinsProducer struct {
	writer         *kafka.Writer
	buffer         chan domain.CheckinEvent
	batch          []kafka.Message
	batchFlushSize int
	done           chan struct{}
	timeout        time.Duration
	logger         *zap.SugaredLogger

	mu     sync.RWMutex
	closed bool
}

func NewCheckinsProducer(logger *zap.SugaredLogger, config config.BrokerConfig) (*CheckinsProducer, error) {
	addr := kafka.TCP(net.JoinHostPort(config.Host, strconv.Itoa(config.Port)))
	logger.Debugf("Connecting to CheckinsProducer on %s", addr.String())

	if err := Ping(addr.String(), config.Timeout); err != nil {
		return nil, err
	}

	writer := &kafka.Writer{
		Addr:         addr,
		Topic:        config.Topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireNone,

		// Для мгновенной отправки, т.к. нет других пишущих в топик горутин
		BatchTimeout: time.Nanosecond,
	}

	p := &CheckinsProducer{
		writer:         writer,
		buffer:         make(chan domain.CheckinEvent, config.BufferSize),
		batch:          make([]kafka.Message, 0, config.BufferSize),
		batchFlushSize: config.BufferSize,
		timeout:        config.Timeout,
		done:           make(chan struct{}, 1),
		logger:         logger,
	}

	go p.run()
	return p, nil
}

func (p *CheckinsProducer) run() {
	ticker := time.Tick(1 * time.Second)

	for {
		select {
		case <-ticker:
			p.sendBatch()
		case event, ok := <-p.buffer:
			if !ok {
				p.sendBatch()
				p.done <- struct{}{}
				return
			}

			value, err := json.Marshal(event)
			if err != nil {
				p.logger.Errorw("failed to marshal event", "error", err)
				continue
			}

			p.batch = append(p.batch, kafka.Message{
				Key:   event.DeviceID[:],
				Value: value,
				Time:  event.Timestamp,
			})

			if len(p.batch) >= p.batchFlushSize {
				p.sendBatch()
			}
		}
	}
}

func (p *CheckinsProducer) sendBatch() {
	if len(p.batch) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	err := p.writer.WriteMessages(ctx, p.batch...)
	if err != nil {
		p.logger.Errorw("failed to publish checkin event", "error", err)
	}

	p.batch = p.batch[:0]
	cancel()
}

func (p *CheckinsProducer) Produce(event domain.CheckinEvent) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		p.logger.Warnw("producer is closed, discarding event", "device_id", event.DeviceID, "campaign_id", event.CampaignID)
		return
	}

	select {
	case p.buffer <- event:
	default:
		p.logger.Warnw("device buffer is full, discarding event", "device_id", event.DeviceID, "campaign_id", event.CampaignID)
	}
}

func (p *CheckinsProducer) Close() {
	p.mu.Lock()

	if p.closed {
		p.logger.Warn("repeated close on CheckinsProducer")
		p.mu.Unlock()
		return
	}

	p.closed = true
	close(p.buffer)
	p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	select {
	case <-p.done:
	case <-ctx.Done():
		p.logger.Warn("device producer: graceful shutdown timed out, forcing it")
	}

	p.writer.Close()
}
