package kafka

import (
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

func Ping(address string) error {
	dialer := &kafka.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("kafka ping failed: cannot connect to broker: %w", err)
	}
	defer conn.Close()

	if _, err = conn.Brokers(); err != nil {
		return fmt.Errorf("kafka ping failed: cannot fetch metadata: %w", err)
	}
	return nil
}
