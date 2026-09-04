//go:build integration

package redis

import (
	"context"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

var (
	testRDB        *redis.Client
	redisContainer *tcredis.RedisContainer
)

// terminateAndFatal stops the already-started container before exiting, so a
// failed startup does not leak it (the testcontainers reaper is best-effort).
func terminateAndFatal(format string, args ...any) {
	if redisContainer != nil {
		_ = redisContainer.Terminate(context.Background())
	}
	log.Fatalf(format, args...)
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcredis.Run(ctx, "redis:8-alpine")
	if err != nil {
		log.Fatalf("failed to start redis container: %v", err)
	}
	redisContainer = container

	host, err := container.Host(ctx)
	if err != nil {
		terminateAndFatal("failed to get redis host: %v", err)
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		terminateAndFatal("failed to get redis port: %v", err)
	}

	testRDB = redis.NewClient(&redis.Options{
		Addr:         net.JoinHostPort(host, port.Port()),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	if err := testRDB.Ping(ctx).Err(); err != nil {
		terminateAndFatal("failed to ping redis: %v", err)
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

// setCurrentStage writes the active stage hash directly so that GetCampaignStats
// can resolve the stage id without a dedicated setter on the repo.
func setCurrentStage(t *testing.T, id, stageID uuid.UUID) {
	t.Helper()
	key := "campaign:" + id.String() + ":checkin_data"
	require.NoError(t, testRDB.HSet(context.Background(), key,
		"stage_id", stageID,
		"target_percent", 100,
	).Err())
}
