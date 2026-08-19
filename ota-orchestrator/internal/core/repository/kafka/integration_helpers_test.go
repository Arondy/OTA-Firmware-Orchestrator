//go:build integration

package kafka

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	kafkaImage  = "apache/kafka:4.3.1"
	kafkaExtTCP = "9093/tcp"
)

var testBrokerAddr string

const kafkaStarterScript = `#!/bin/bash
export KAFKA_ADVERTISED_LISTENERS="PLAINTEXT://%s,BROKER://localhost:9092"
echo "Starting Kafka KRaft mode (apache/kafka:4.3.1)"
/etc/kafka/docker/run`

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, addr, err := startKafkaContainer(ctx)
	if err != nil {
		log.Fatalf("failed to start kafka container: %v", err)
	}
	testBrokerAddr = addr

	code := m.Run()

	_ = container.Terminate(ctx)
	os.Exit(code)
}

func startKafkaContainer(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        kafkaImage,
		ExposedPorts: []string{kafkaExtTCP},
		Env: map[string]string{
			"CLUSTER_ID":                                     "5L6g3nShT-eMCtK--X86sw",
			"KAFKA_NODE_ID":                                  "1",
			"KAFKA_PROCESS_ROLES":                            "broker,controller",
			"KAFKA_CONTROLLER_LISTENER_NAMES":                "CONTROLLER",
			"KAFKA_CONTROLLER_QUORUM_VOTERS":                 "1@localhost:9094",
			"KAFKA_LISTENERS":                                "PLAINTEXT://0.0.0.0:9093,BROKER://0.0.0.0:9092,CONTROLLER://0.0.0.0:9094",
			"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP":           "PLAINTEXT:PLAINTEXT,BROKER:PLAINTEXT,CONTROLLER:PLAINTEXT",
			"KAFKA_INTER_BROKER_LISTENER_NAME":               "BROKER",
			"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR":         "1",
			"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR": "1",
			"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR":            "1",
			"KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS":         "0",
			"KAFKA_NUM_PARTITIONS":                           "3",
			"KAFKA_AUTO_CREATE_TOPICS_ENABLE":                "false",
			"KAFKA_LOG_DIRS":                                 "/var/lib/kafka/data",
		},
		Cmd: []string{"sh", "-c", "while [ ! -f /tmp/kafka_start.sh ]; do sleep 0.1; done; bash /tmp/kafka_start.sh"},
		LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
			{
				PostStarts: []testcontainers.ContainerHook{
					func(ctx context.Context, c testcontainers.Container) error {
						if err := wait.ForMappedPort(kafkaExtTCP).WaitUntilReady(ctx, c); err != nil {
							return err
						}
						endpoint, err := c.PortEndpoint(ctx, kafkaExtTCP, "")
						if err != nil {
							return err
						}
						script := fmt.Sprintf(kafkaStarterScript, endpoint)
						return c.CopyToContainer(ctx, []byte(script), "/tmp/kafka_start.sh", 0o755)
					},
				},
			},
		},
		WaitingFor: wait.ForLog("Kafka Server started"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}

	endpoint, err := container.PortEndpoint(ctx, kafkaExtTCP, "")
	if err != nil {
		return nil, "", err
	}

	return container, endpoint, nil
}

func createTopic(t *testing.T, name string, partitions int) {
	t.Helper()

	client := &kafka.Client{Addr: kafka.TCP(testBrokerAddr)}

	var lastErr error
	for attempt := range 30 {
		_, err := client.CreateTopics(context.Background(), &kafka.CreateTopicsRequest{
			Topics: []kafka.TopicConfig{{
				Topic:             name,
				NumPartitions:     partitions,
				ReplicationFactor: 1,
			}},
		})
		if err != nil && !errors.Is(err, kafka.TopicAlreadyExists) {
			lastErr = err
			if attempt%5 == 0 {
				t.Logf("createTopic %q attempt %d: %v", name, attempt, err)
			}
			time.Sleep(time.Second)
			continue
		}

		// Topic creation is asynchronous in Kafka: the partition leader is
		// elected after the partition count becomes visible, and the broker
		// metadata may still report UnknownTopicOrPartition for the freshly
		// created topic. Poll Metadata until every partition has an elected
		// leader so the next producer's cached metadata is stable.
		if waitTopicReady(t, client, name, partitions) {
			return
		}
		time.Sleep(time.Second)
	}
	requireNoErr(t, lastErr)
}

// waitTopicReady polls the broker metadata until the topic reports the
// expected number of partitions, each with an elected leader. It returns true
// as soon as the topic is ready, or false if the deadline is exceeded.
func waitTopicReady(t *testing.T, client *kafka.Client, name string, partitions int) bool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	backoff := 100 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			t.Logf("waitTopicReady %q: timed out waiting for metadata", name)
			return false
		default:
		}

		res, err := client.Metadata(ctx, &kafka.MetadataRequest{Topics: []string{name}})
		if err != nil {
			t.Logf("waitTopicReady %q: metadata: %v", name, err)
			time.Sleep(backoff)
			if backoff < 2*time.Second {
				backoff *= 2
			}
			continue
		}

		var topic *kafka.Topic
		for i := range res.Topics {
			if res.Topics[i].Name == name {
				topic = &res.Topics[i]
				break
			}
		}
		if topic == nil || topic.Error != nil {
			t.Logf("waitTopicReady %q: topic not present in metadata yet", name)
			time.Sleep(backoff)
			if backoff < 2*time.Second {
				backoff *= 2
			}
			continue
		}

		if len(topic.Partitions) != partitions {
			t.Logf("waitTopicReady %q: partitions=%d, want %d", name, len(topic.Partitions), partitions)
			time.Sleep(backoff)
			if backoff < 2*time.Second {
				backoff *= 2
			}
			continue
		}

		ready := true
		for _, p := range topic.Partitions {
			if p.Leader.ID < 0 {
				ready = false
				break
			}
		}
		if ready {
			return true
		}
		t.Logf("waitTopicReady %q: partitions present but no leader elected yet", name)
		time.Sleep(backoff)
		if backoff < 2*time.Second {
			backoff *= 2
		}
	}
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func brokerHostPort() (string, int) {
	host, portStr, err := net.SplitHostPort(testBrokerAddr)
	if err != nil {
		panic(err)
	}
	var port int
	_, _ = fmt.Sscanf(portStr, "%d", &port)
	return host, port
}

func consumerReader(topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{testBrokerAddr},
		Topic:       topic,
		GroupID:     "integration-" + topic + "-" + time.Now().Format("150405.000"),
		StartOffset: kafka.FirstOffset,
	})
}
