// Вносит данные для нагрузочного тестирования и демо в Postgres/Redis
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/Arondy/OTA-Firmware-Orchestrator/tests/seed/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/tests/seed/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type loadSummary struct {
	model         string
	campaignID    uuid.UUID
	activeStageID uuid.UUID
	devices       repository.DeviceSeed
}

type demoSummary struct {
	spec       demoCampaignSpec
	campaignID uuid.UUID
	stageIDs   []uuid.UUID
	devices    repository.DeviceSeed
	counts     map[string]int
}

func main() {
	ctx := context.Background()
	cfg := config.LoadConfig()
	rng := rand.New(rand.NewSource(config.RandSeed))
	models := AllSeedModels()

	pool := repository.MustPostgresPool(ctx, cfg.DB)
	defer pool.Close()
	rdb := repository.MustRedisClient(ctx, cfg.Cache)
	defer func() { _ = rdb.Close() }()

	if config.CleanupBeforeSeed {
		campaignIDs, stageIDs, deviceIDs := repository.ListSeedIDs(ctx, pool, models)
		repository.CleanupRedisKeys(ctx, rdb, campaignIDs, stageIDs, deviceIDs)
		repository.DeleteSeedRows(ctx, pool, models, campaignIDs)
	}

	loads := make([]loadSummary, 0, config.NumLoadCampaigns)
	for i, model := range LoadCampaignModels() {
		fwID := repository.InsertFirmwareVersion(ctx, pool, model, config.TargetVersion)
		devices := repository.InsertDevicesForModel(ctx, pool, rng, model, config.BaseVersion, config.TargetVersion, splitTotal(config.NumDevices, config.NumLoadCampaigns, i))
		campaignID, stages := repository.InsertLoadCampaign(ctx, pool, model, fwID)
		activeStageID := stages[0]

		repository.SeedCampaignProjection(ctx, rdb, campaignID, activeStageID, config.StageTargetPercent, true)
		repository.SeedDeviceCache(ctx, rdb, cfg.Cache, devices.IDs, devices.Versions)

		loads = append(loads, loadSummary{model: model, campaignID: campaignID, activeStageID: activeStageID, devices: devices})
	}

	summaries := make([]demoSummary, 0, len(DemoCampaignSpecs()))
	for _, spec := range DemoCampaignSpecs() {
		demoFwID := repository.InsertFirmwareVersion(ctx, pool, spec.DeviceModel, spec.TargetVersion)
		devices := repository.InsertDevicesForModel(ctx, pool, rng, spec.DeviceModel, spec.BaseVersion, spec.TargetVersion, spec.NumDevices)
		campaignID, stageIDs := repository.InsertCampaignFromSpec(ctx, pool, spec, demoFwID)

		summary := demoSummary{spec: spec, campaignID: campaignID, stageIDs: stageIDs, devices: devices}
		if idx, ok := attemptsStageIndex(spec); ok {
			if spec.Status == "rolled_back" {
				summary.counts = repository.InsertDemoAttempts(ctx, pool, rng, devices.IDs, campaignID, stageIDs[idx], spec.NumAttempts)
			} else {
				summary.counts = repository.InsertUniformAttempts(ctx, pool, rng, devices.IDs, campaignID, stageIDs[idx], spec.NumAttempts, "success")
			}

			if spec.Status == "running" || spec.Status == "paused" {
				repository.SeedCampaignProjection(ctx, rdb, campaignID, stageIDs[idx], targetPercentOf(spec, idx), spec.SeedRunningSet)
			}

			repository.SeedCounters(ctx, rdb, campaignID, stageIDs[idx], summary.counts)
		}

		repository.SeedDeviceCache(ctx, rdb, cfg.Cache, devices.IDs, devices.Versions)
		summaries = append(summaries, summary)
	}

	repository.InsertFirmwareVersion(ctx, pool, config.DemoModelOrphan, config.DemoOrphanVersion)
	orphanDevices := repository.InsertDevicesForModel(ctx, pool, rng, config.DemoModelOrphan, config.DemoOrphanVersion, config.DemoOrphanVersion, config.DemoOrphanDevices)
	repository.SeedDeviceCache(ctx, rdb, cfg.Cache, orphanDevices.IDs, orphanDevices.Versions)

	verify(ctx, pool, rdb, models, loads, summaries)
	printSeedStats(loads, summaries, orphanDevices)
}

// splitTotal делит total на parts почти поровну: первые total%parts долей больше на 1.
func splitTotal(total, parts, i int) int {
	n := total / parts

	if i < total%parts {
		n++
	}

	return n
}

func attemptsStageIndex(spec demoCampaignSpec) (int, bool) {
	if spec.NumAttempts == 0 {
		return 0, false
	}

	for i, status := range spec.StageStatuses {
		if status == "active" {
			return i, true
		}
	}

	for i, status := range spec.StageStatuses {
		if status == "failed" {
			return i, true
		}
	}

	return len(spec.StageStatuses) - 1, true
}

func targetPercentOf(spec demoCampaignSpec, idx int) int {
	if idx < len(spec.TargetPercents) {
		return spec.TargetPercents[idx]
	}

	return config.StageTargetPercent
}

func verify(ctx context.Context, pool *pgxpool.Pool, rdb *redis.Client, models []string, loads []loadSummary, summaries []demoSummary) {
	var devices int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM devices WHERE device_model = ANY($1)`, models).Scan(&devices); err != nil {
		log.Fatalf("verify devices: %v", err)
	}

	loadIDs := make([]uuid.UUID, len(loads))
	for i, l := range loads {
		loadIDs[i] = l.campaignID
	}

	var loadAttempts int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM update_attempts WHERE campaign_id = ANY($1)`, loadIDs).Scan(&loadAttempts); err != nil {
		log.Fatalf("verify attempts: %v", err)
	}

	for _, l := range loads {
		checkinLen, err := rdb.HLen(ctx, fmt.Sprintf("campaign:%s:checkin_data", l.campaignID)).Result()
		if err != nil {
			log.Fatalf("verify campaign cache %s: %v", l.model, err)
		}

		statsLen, err := rdb.HLen(ctx, fmt.Sprintf("stage:%s:stats", l.activeStageID)).Result()
		if err != nil {
			log.Fatalf("verify stage cache %s: %v", l.model, err)
		}

		isMember, err := rdb.SIsMember(ctx, "campaign:running_campaigns", l.campaignID).Result()
		if err != nil {
			log.Fatalf("verify running set %s: %v", l.model, err)
		}

		if checkinLen != 2 || statsLen != 2 || !isMember {
			log.Fatalf("verify failed: load campaign %s projection broken", l.model)
		}
	}

	expectedDevices := config.NumDevices + config.DemoOrphanDevices
	for _, s := range summaries {
		expectedDevices += s.spec.NumDevices
	}

	if devices != expectedDevices || loadAttempts != 0 {
		log.Fatalf("verify failed: load counts do not match constants")
	}

	for _, s := range summaries {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM update_attempts WHERE campaign_id = $1`, s.campaignID).Scan(&n); err != nil {
			log.Fatalf("verify demo attempts %s: %v", s.spec.Name, err)
		}

		if n != s.spec.NumAttempts {
			log.Fatalf("verify failed: campaign %s attempts=%d, want %d", s.spec.Name, n, s.spec.NumAttempts)
		}

		inSet, err := rdb.SIsMember(ctx, "campaign:running_campaigns", s.campaignID).Result()
		if err != nil {
			log.Fatalf("verify demo running set %s: %v", s.spec.Name, err)
		}

		if inSet != s.spec.SeedRunningSet {
			log.Fatalf("verify failed: campaign %s running_set=%v, want %v", s.spec.Name, inSet, s.spec.SeedRunningSet)
		}
	}
}

func printSeedStats(loads []loadSummary, summaries []demoSummary, orphans repository.DeviceSeed) {
	devices := len(orphans.IDs)
	for _, l := range loads {
		devices += len(l.devices.IDs)
	}

	for _, s := range summaries {
		devices += len(s.devices.IDs)
	}

	stages := config.NumStages * len(loads)
	for _, s := range summaries {
		stages += len(s.stageIDs)
	}

	attempts := 0
	byResult := map[string]int{"success": 0, "failure": 0, "timeout": 0}
	for _, s := range summaries {
		for result, n := range s.counts {
			byResult[result] += n
			attempts += n
		}
	}

	log.Printf("stats: firmware=%d campaigns=%d (load=%d demo=%d) stages=%d devices=%d attempts=%d",
		len(loads)+len(summaries)+1, len(loads)+len(summaries), len(loads), len(summaries), stages, devices, attempts)
	log.Printf("stats: demo attempts by result=%v", byResult)
}
