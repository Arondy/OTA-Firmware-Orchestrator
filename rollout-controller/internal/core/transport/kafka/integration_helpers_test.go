//go:build integration

package core_kafka

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	tkafka "github.com/Arondy/OTA-Firmware-Orchestrator/testutil/kafka"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/goleak"
)

func uniqueTopic(base string) string {
	return fmt.Sprintf("%s-%s", base, uuid.NewString())
}

func TestMain(m *testing.M) {
	tkafka.BrokerAddr()

	code := m.Run()

	tkafka.Terminate()

	if code == 0 {
		if err := goleak.Find(
			goleak.IgnoreAnyFunction("github.com/Microsoft/go-winio.ioCompletionProcessor"),
		); err != nil {
			fmt.Fprintf(os.Stderr, "goleak: Errors on successful test run: %v\n", err)
			code = 1
		}
	}

	os.Exit(code)
}

func createTopic(t *testing.T, name string, partitions int) {
	t.Helper()
	tkafka.CreateTopic(name, partitions)
}

func brokerHostPort() (string, int) {
	return tkafka.BrokerHostPort()
}

func produceMessage(t *testing.T, topic string, key, value []byte) {
	t.Helper()

	writer := tkafka.NewWriter(topic)
	defer writer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := writer.WriteMessages(ctx, kafka.Message{Key: key, Value: value}); err != nil {
		t.Fatalf("produce message: %v", err)
	}
}

func headerValue(msg kafka.Message, key string) string {
	for _, h := range msg.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func readMatchingFromTopic(t *testing.T, topic string, timeout time.Duration, match func(kafka.Message) bool) (kafka.Message, bool) {
	t.Helper()

	reader := tkafka.NewReader(topic)
	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			return kafka.Message{}, false
		}
		if match(msg) {
			return msg, true
		}
	}
}
