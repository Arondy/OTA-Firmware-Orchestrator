//go:build integration

package kafka

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	tkafka "github.com/Arondy/OTA-Firmware-Orchestrator/testutil/kafka"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func uniqueTopic(base string) string {
	return fmt.Sprintf("%s-%s", base, uuid.NewString())
}

func TestMain(m *testing.M) {
	tkafka.BrokerAddr()

	code := m.Run()

	tkafka.Terminate()
	os.Exit(code)
}

func createTopic(t *testing.T, name string, partitions int) {
	t.Helper()
	tkafka.CreateTopic(name, partitions)
}

func brokerHostPort() (string, int) {
	return tkafka.BrokerHostPort()
}

func consumerReader(topic string) *kafka.Reader {
	return tkafka.NewReader(topic)
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
