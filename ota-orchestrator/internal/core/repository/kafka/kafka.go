package kafka

import (
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

func Ping(address string, timeout time.Duration) error {
	const attempts = 15
	const pause = time.Second
	var lastErr error
	dialer := &kafka.Dialer{Timeout: timeout}

	for i := range attempts {
		conn, err := dialer.Dial("tcp", address)
		if err != nil {
			return fmt.Errorf("connection to kafka address failed: %w", err)
		}

		if _, err = conn.Brokers(); err == nil {
			conn.Close()
			return nil
		}
		lastErr = err
		conn.Close()

		if i < attempts-1 {
			time.Sleep(pause)
		}
	}
	return fmt.Errorf("kafka ping failed after %d attempts: %w", attempts, lastErr)
}
