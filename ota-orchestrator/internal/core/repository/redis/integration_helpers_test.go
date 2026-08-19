//go:build integration

package redis

import (
	"context"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

var testRDB *redis.Client

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcredis.Run(ctx, "redis:8-alpine")
	if err != nil {
		log.Fatalf("failed to start redis container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		log.Fatalf("failed to get redis host: %v", err)
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		log.Fatalf("failed to get redis port: %v", err)
	}

	testRDB = redis.NewClient(&redis.Options{
		Addr:         net.JoinHostPort(host, port.Port()),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	if err := testRDB.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to ping redis: %v", err)
	}

	code := m.Run()

	_ = testRDB.Close()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

func resetRedis(t *testing.T) {
	t.Helper()
	require.NoError(t, testRDB.FlushDB(context.Background()).Err())
}
