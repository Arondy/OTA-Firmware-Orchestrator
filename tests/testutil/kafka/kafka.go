// Package kafka provides shared testcontainers helpers for spinning up a
// single-node KRaft Kafka broker for the duration of a test process. A fresh
// container is created per process and terminated on exit, so every run starts
// from a clean broker. It is intentionally free of build tags so it can be
// imported from integration and e2e test packages.
package kafka

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// KafkaImage is the broker image used by integration tests. It matches the
	// image pinned in docker-compose.yml.
	KafkaImage = "apache/kafka:4.3.1"
	// KafkaExtTCP is the advertised-listener port exposed to the host.
	KafkaExtTCP = "9093/tcp"
)

var (
	mu              sync.Mutex
	brokerAddrOnce  sync.Once
	brokerAddr      string
	brokerContainer testcontainers.Container
)

const kafkaStarterScript = `#!/bin/bash
export KAFKA_ADVERTISED_LISTENERS="PLAINTEXT://%s,BROKER://localhost:9092"
echo "Starting Kafka KRaft mode (apache/kafka:4.3.1)"
/etc/kafka/docker/run`

// BrokerAddr returns the host:port of the shared Kafka broker. The first call
// starts (or reuses) the container; subsequent calls return the cached value.
func BrokerAddr() string {
	brokerAddrOnce.Do(func() {
		ctx := context.Background()
		container, addr, err := startKafkaContainer(ctx)
		if err != nil {
			log.Fatalf("failed to start kafka container: %v", err)
		}
		brokerAddr = addr
		brokerContainer = container
	})
	return brokerAddr
}

// startKafkaContainer launches a single-node KRaft broker. A fresh container is
// created for each test process and is terminated on exit (see Terminate), so
// every run starts from a clean broker with no leftover topics or messages.
func startKafkaContainer(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        KafkaImage,
		ExposedPorts: []string{KafkaExtTCP},
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
						if err := wait.ForMappedPort(KafkaExtTCP).WaitUntilReady(ctx, c); err != nil {
							return err
						}
						endpoint, err := c.PortEndpoint(ctx, KafkaExtTCP, "")
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

	endpoint, err := container.PortEndpoint(ctx, KafkaExtTCP, "")
	if err != nil {
		return nil, "", err
	}

	return container, endpoint, nil
}

// CreateTopic creates a topic with the given number of partitions and blocks
// until every partition has an elected leader, so the next producer's cached
// metadata is stable.
func CreateTopic(name string, partitions int) {
	client := &kafka.Client{Addr: kafka.TCP(brokerAddr)}

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
				log.Printf("createTopic %q attempt %d: %v", name, attempt, err)
			}
			time.Sleep(time.Second)
			continue
		}

		if waitTopicReady(client, name, partitions) {
			return
		}
		time.Sleep(time.Second)
	}
	if lastErr != nil {
		// The broker is useless without its topics; stop it explicitly so the
		// fatal exit does not rely on the best-effort reaper.
		Terminate()
		log.Fatalf("createTopic %q: %v", name, lastErr)
	}
}

// DeleteTopic removes a topic so the next test starts from a clean slate. It is
// a no-op (logged) when the topic does not exist. Used by consumer integration
// tests that otherwise replay messages left behind by previous runs or by the
// producer tests in the sibling module.
func DeleteTopic(name string) {
	client := &kafka.Client{Addr: kafka.TCP(brokerAddr)}
	if _, err := client.DeleteTopics(context.Background(), &kafka.DeleteTopicsRequest{Topics: []string{name}}); err != nil {
		log.Printf("deleteTopic %q: %v", name, err)
	}
}

// waitTopicReady polls the broker metadata until the topic reports the expected
// number of partitions, each with an elected leader.
func waitTopicReady(client *kafka.Client, name string, partitions int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	backoff := 100 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			log.Printf("waitTopicReady %q: timed out waiting for metadata", name)
			return false
		default:
		}

		res, err := client.Metadata(ctx, &kafka.MetadataRequest{Topics: []string{name}})
		if err != nil {
			log.Printf("waitTopicReady %q: metadata: %v", name, err)
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
			log.Printf("waitTopicReady %q: topic not present in metadata yet", name)
			time.Sleep(backoff)
			if backoff < 2*time.Second {
				backoff *= 2
			}
			continue
		}

		if len(topic.Partitions) != partitions {
			log.Printf("waitTopicReady %q: partitions=%d, want %d", name, len(topic.Partitions), partitions)
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
		log.Printf("waitTopicReady %q: partitions present but no leader elected yet", name)
		time.Sleep(backoff)
		if backoff < 2*time.Second {
			backoff *= 2
		}
	}
}

// BrokerHostPort returns the host and port of the shared broker.
func BrokerHostPort() (string, int) {
	host, portStr, err := net.SplitHostPort(brokerAddr)
	if err != nil {
		panic(err)
	}
	var port int
	_, _ = fmt.Sscanf(portStr, "%d", &port)
	return host, port
}

// NewReader returns a kafka.Reader for the given topic, starting from the first
// offset, with a unique group id so it never conflicts with other tests.
func NewReader(topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{brokerAddr},
		Topic:       topic,
		GroupID:     fmt.Sprintf("integration-%s-%d", topic, time.Now().UnixNano()),
		StartOffset: kafka.FirstOffset,
	})
}

// NewWriter returns a kafka.Writer for the given topic with hash balancing and
// RequireOne acks.
func NewWriter(topic string) *kafka.Writer {
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{brokerAddr},
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: int(kafka.RequireOne),
	})
}

// Terminate stops the Kafka broker container started by this process. It is a
// no-op if BrokerAddr was never called. TestMain calls it on exit so the
// container does not leak; testcontainers' reaper also cleans it up.
func Terminate() {
	mu.Lock()
	defer mu.Unlock()
	if brokerContainer != nil {
		_ = brokerContainer.Terminate(context.Background())
	}
	// The broker is gone: release the shared kafka-go pool so its background
	// conn/discover goroutines do not outlive the test process (goleak).
	if t, ok := kafka.DefaultTransport.(*kafka.Transport); ok {
		t.CloseIdleConnections()
	}
}
