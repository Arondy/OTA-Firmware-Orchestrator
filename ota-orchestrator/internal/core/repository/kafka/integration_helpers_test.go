//go:build integration

package kafka

import (
	"context"
	"fmt"
	"os"
	"runtime"
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
	if runtime.GOOS == "windows" && os.Getenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE") == "" {
		os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "/var/run/docker.sock")
	}
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
			if ctx.Err() != nil {
				return kafka.Message{}, false
			}
			continue
		}
		if match(msg) {
			return msg, true
		}
	}
}
