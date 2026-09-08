package repository

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/tests/seed/config"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func MustRedisClient(ctx context.Context, cfg config.CacheConfig) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port)),
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("ping redis: %v", err)
	}
	return rdb
}

func SeedCampaignProjection(ctx context.Context, rdb *redis.Client, campaignID, activeStageID uuid.UUID, targetPercent int, seedRunningSet bool) {
	checkinKey := fmt.Sprintf("campaign:%s:checkin_data", campaignID)
	if err := rdb.HSet(ctx, checkinKey,
		"stage_id", activeStageID,
		"target_percent", targetPercent,
	).Err(); err != nil {
		log.Fatalf("redis: set %s: %v", checkinKey, err)
	}

	if seedRunningSet {
		if err := rdb.SAdd(ctx, "campaign:running_campaigns", campaignID).Err(); err != nil {
			log.Fatalf("redis: sadd running campaigns: %v", err)
		}
	}

	statsKey := fmt.Sprintf("stage:%s:stats", activeStageID)
	if err := rdb.HSet(ctx, statsKey,
		"min_sample_size", config.StageMinSampleSize,
		"success_threshold", config.StageSuccessThreshold,
	).Err(); err != nil {
		log.Fatalf("redis: set %s: %v", statsKey, err)
	}
}

func SeedDeviceCache(ctx context.Context, rdb *redis.Client, cache config.CacheConfig, deviceIDs []uuid.UUID, versions []string) {
	for start := 0; start < len(deviceIDs); start += config.BatchSize {
		end := min(start+config.BatchSize, len(deviceIDs))
		_, err := rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
			for i, id := range deviceIDs[start:end] {
				key := fmt.Sprintf("device:%s:checkin_data", id)
				p.HSet(ctx, key, "current_version", versions[start+i], "last_seen", time.Now().UTC())
				p.Expire(ctx, key, cache.DeviceCheckinDataTTL)
			}

			return nil
		})
		if err != nil {
			log.Fatalf("redis: seed devices [%d:%d]: %v", start, end, err)
		}
	}
}

func SeedCounters(ctx context.Context, rdb *redis.Client, campaignID, stageID uuid.UUID, counts map[string]int) {
	pipe := rdb.Pipeline()
	pipe.Set(ctx, fmt.Sprintf("campaign:%s:stage:%s:success", campaignID, stageID), counts["success"], 0)
	pipe.Set(ctx, fmt.Sprintf("campaign:%s:stage:%s:failure", campaignID, stageID), counts["failure"], 0)
	pipe.Set(ctx, fmt.Sprintf("campaign:%s:stage:%s:timeout", campaignID, stageID), counts["timeout"], 0)

	if _, err := pipe.Exec(ctx); err != nil {
		log.Fatalf("redis: set counters: %v", err)
	}
}

func CleanupRedisKeys(ctx context.Context, rdb *redis.Client, campaignIDs, stageIDs, deviceIDs []uuid.UUID) {
	if len(campaignIDs) == 0 && len(stageIDs) == 0 && len(deviceIDs) == 0 {
		return
	}

	// Внутри Pipelined нельзя читать через rdb, поэтому SCAN заранее.
	var seenKeys []string
	for _, campaignID := range campaignIDs {
		var cursor uint64

		for {
			keys, next, err := rdb.Scan(ctx, cursor, fmt.Sprintf("campaign:%s:stage:*:seen:*", campaignID), 500).Result()
			if err != nil {
				log.Fatalf("cleanup redis scan seen: %v", err)
			}

			seenKeys = append(seenKeys, keys...)
			cursor = next

			if cursor == 0 {
				break
			}
		}
	}

	_, err := rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, id := range campaignIDs {
			p.Del(ctx, fmt.Sprintf("campaign:%s:checkin_data", id))
			p.Del(ctx, fmt.Sprintf("campaign:%s:stable_cycles", id))
			p.Del(ctx, fmt.Sprintf("campaign:%s:decision", id))
			p.SRem(ctx, "campaign:running_campaigns", id)
		}

		for _, id := range stageIDs {
			p.Del(ctx, fmt.Sprintf("stage:%s:stats", id))
		}

		// Del не раскрывает glob, только точные ключи.
		for _, campaignID := range campaignIDs {
			for _, stageID := range stageIDs {
				for _, res := range []string{"success", "failure", "timeout"} {
					p.Del(ctx, fmt.Sprintf("campaign:%s:stage:%s:%s", campaignID, stageID, res))
				}
			}
		}

		for _, k := range seenKeys {
			p.Del(ctx, k)
		}

		for _, id := range deviceIDs {
			p.Del(ctx, fmt.Sprintf("device:%s:checkin_data", id))
		}

		return nil
	})

	if err != nil {
		log.Fatalf("cleanup redis: %v", err)
	}
}
