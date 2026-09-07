package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"math/rand"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/tests/seed/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CampaignSpec struct {
	Name           string
	DeviceModel    string
	BaseVersion    string
	TargetVersion  string
	Status         string
	SeedRunningSet bool
	StageStatuses  []string
	TargetPercents []int
	NumDevices     int
	NumAttempts    int
}

type DeviceSeed struct {
	IDs      []uuid.UUID
	Versions []string
}

func MustPostgresPool(ctx context.Context, cfg config.DBConfig) *pgxpool.Pool {
	pgxConfig, err := pgxpool.ParseConfig(cfg.ConnString())
	if err != nil {
		log.Fatalf("parse pg config: %v", err)
	}

	pgxConfig.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		log.Fatalf("create pg pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	return pool
}

// ID нужны чистке Redis до удаления строк из БД.
func ListSeedIDs(ctx context.Context, pool *pgxpool.Pool, models []string) (campaignIDs, stageIDs, deviceIDs []uuid.UUID) {
	rows, err := pool.Query(ctx, `SELECT id FROM rollout_campaigns WHERE device_model = ANY($1)`, models)
	if err != nil {
		log.Fatalf("cleanup: list campaigns: %v", err)
	}
	campaignIDs, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		log.Fatalf("cleanup: collect campaigns: %v", err)
	}

	if len(campaignIDs) > 0 {
		rows, err = pool.Query(ctx, `SELECT id FROM rollout_stages WHERE campaign_id = ANY($1)`, campaignIDs)
		if err != nil {
			log.Fatalf("cleanup: list stages: %v", err)
		}
		stageIDs, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if err != nil {
			log.Fatalf("cleanup: collect stages: %v", err)
		}
	}

	devRows, err := pool.Query(ctx, `SELECT id FROM devices WHERE device_model = ANY($1)`, models)
	if err != nil {
		log.Fatalf("cleanup: list devices: %v", err)
	}
	deviceIDs, err = pgx.CollectRows(devRows, pgx.RowTo[uuid.UUID])
	if err != nil {
		log.Fatalf("cleanup: collect devices: %v", err)
	}

	return campaignIDs, stageIDs, deviceIDs
}

func DeleteSeedRows(ctx context.Context, pool *pgxpool.Pool, models []string, campaignIDs []uuid.UUID) {
	if len(campaignIDs) > 0 {
		if _, err := pool.Exec(ctx, `DELETE FROM update_attempts WHERE campaign_id = ANY($1)`, campaignIDs); err != nil {
			log.Fatalf("cleanup: delete attempts: %v", err)
		}

		if _, err := pool.Exec(ctx, `DELETE FROM applied_decisions WHERE campaign_id = ANY($1)`, campaignIDs); err != nil {
			log.Fatalf("cleanup: delete decisions: %v", err)
		}

		if _, err := pool.Exec(ctx, `DELETE FROM rollout_stages WHERE campaign_id = ANY($1)`, campaignIDs); err != nil {
			log.Fatalf("cleanup: delete stages: %v", err)
		}

		if _, err := pool.Exec(ctx, `DELETE FROM rollout_campaigns WHERE id = ANY($1)`, campaignIDs); err != nil {
			log.Fatalf("cleanup: delete campaigns: %v", err)
		}
	}

	if _, err := pool.Exec(ctx, `DELETE FROM devices WHERE device_model = ANY($1)`, models); err != nil {
		log.Fatalf("cleanup: delete devices: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM firmware_versions WHERE device_model = ANY($1)`, models); err != nil {
		log.Fatalf("cleanup: delete firmware: %v", err)
	}
}

func InsertFirmwareVersion(ctx context.Context, pool *pgxpool.Pool, deviceModel, fwVersion string) uuid.UUID {
	sum := sha256.Sum256([]byte("load-firmware-" + deviceModel + "-" + fwVersion))
	checksum := hex.EncodeToString(sum[:])
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO firmware_versions (device_model, fw_version, fw_checksum, binary_url)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		deviceModel, fwVersion, checksum, "http://files.local/fw/"+deviceModel+"/"+fwVersion+".bin",
	).Scan(&id)
	if err != nil {
		log.Fatalf("insert firmware %s %s: %v", deviceModel, fwVersion, err)
	}

	return id
}

func InsertLoadCampaign(ctx context.Context, pool *pgxpool.Pool, deviceModel string, fwID uuid.UUID) (uuid.UUID, []uuid.UUID) {
	statuses := make([]string, config.NumStages)
	for i := range statuses {
		statuses[i] = "pending"
	}

	statuses[0] = "active"

	percents := make([]int, config.NumStages)
	for i := range percents {
		percents[i] = config.StageTargetPercent
	}

	return InsertCampaignFromSpec(ctx, pool, CampaignSpec{
		Name:           "load-" + deviceModel,
		DeviceModel:    deviceModel,
		Status:         "running",
		StageStatuses:  statuses,
		TargetPercents: percents,
	}, fwID)
}

func InsertDevicesForModel(ctx context.Context, pool *pgxpool.Pool, rng *rand.Rand, deviceModel, baseVersion, targetVersion string, n int) DeviceSeed {
	seed := DeviceSeed{
		IDs:      make([]uuid.UUID, n),
		Versions: make([]string, n),
	}
	rows := make([][]any, n)
	for i := range rows {
		id := mustV7()
		version := deviceVersion(rng, baseVersion, targetVersion)
		status := deviceStatus(rng)
		seed.IDs[i] = id
		seed.Versions[i] = version
		rows[i] = []any{id, deviceModel, version, status}
	}

	for start := 0; start < len(rows); start += config.BatchSize {
		end := min(start+config.BatchSize, len(rows))
		_, err := pool.CopyFrom(ctx,
			pgx.Identifier{"devices"},
			[]string{"id", "device_model", "current_version", "status"},
			pgx.CopyFromRows(rows[start:end]),
		)
		if err != nil {
			log.Fatalf("copy devices %s [%d:%d]: %v", deviceModel, start, end, err)
		}
	}

	return seed
}

func InsertCampaignFromSpec(ctx context.Context, pool *pgxpool.Pool, spec CampaignSpec, fwID uuid.UUID) (uuid.UUID, []uuid.UUID) {
	var campaignID uuid.UUID
	var startedAt, completedAt any
	switch spec.Status {
	case "running", "paused":
		startedAt = time.Now()
	case "completed", "rolled_back":
		startedAt = time.Now()
		completedAt = time.Now()
	}

	err := pool.QueryRow(ctx, `
		INSERT INTO rollout_campaigns (firmware_version_id, device_model, status, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`, fwID, spec.DeviceModel, spec.Status, startedAt, completedAt).Scan(&campaignID)
	if err != nil {
		log.Fatalf("insert campaign %s: %v", spec.Name, err)
	}

	stages := make([]uuid.UUID, len(spec.StageStatuses))
	for i, status := range spec.StageStatuses {
		targetPercent := config.StageTargetPercent
		if i < len(spec.TargetPercents) {
			targetPercent = spec.TargetPercents[i]
		}

		var enteredAt any
		if status != "pending" {
			enteredAt = time.Now()
		}

		var stageID uuid.UUID
		err := pool.QueryRow(ctx, `
			INSERT INTO rollout_stages
				(campaign_id, order_index, target_percent, min_sample_size, success_threshold, status, entered_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`,
			campaignID, i, targetPercent, config.StageMinSampleSize, config.StageSuccessThreshold, status, enteredAt,
		).Scan(&stageID)
		if err != nil {
			log.Fatalf("insert stage %s %d: %v", spec.Name, i, err)
		}

		stages[i] = stageID
	}

	return campaignID, stages
}

func copyAttempts(ctx context.Context, pool *pgxpool.Pool, rows [][]any, what string) {
	for start := 0; start < len(rows); start += config.BatchSize {
		end := min(start+config.BatchSize, len(rows))

		_, err := pool.CopyFrom(ctx,
			pgx.Identifier{"update_attempts"},
			[]string{"device_id", "campaign_id", "stage_id", "result", "event_id"},
			pgx.CopyFromRows(rows[start:end]),
		)
		if err != nil {
			log.Fatalf("copy %s [%d:%d]: %v", what, start, end, err)
		}
	}
}

func InsertDemoAttempts(ctx context.Context, pool *pgxpool.Pool, rng *rand.Rand, deviceIDs []uuid.UUID, campaignID, stageID uuid.UUID, n int) map[string]int {
	counts := map[string]int{"success": 0, "failure": 0, "timeout": 0}
	if n == 0 || len(deviceIDs) == 0 {
		return counts
	}

	rows := make([][]any, 0, n)
	for _, result := range []string{"success", "failure", "timeout"} {
		if len(rows) >= n {
			break
		}
		counts[result]++
		rows = append(rows, []any{
			deviceIDs[rng.Intn(len(deviceIDs))],
			campaignID,
			stageID,
			result,
			mustV7(),
		})
	}

	for len(rows) < n {
		result := rollResult(rng)
		counts[result]++
		rows = append(rows, []any{
			deviceIDs[rng.Intn(len(deviceIDs))],
			campaignID,
			stageID,
			result,
			mustV7(),
		})
	}

	copyAttempts(ctx, pool, rows, "demo attempts")

	return counts
}

func InsertUniformAttempts(ctx context.Context, pool *pgxpool.Pool, rng *rand.Rand, deviceIDs []uuid.UUID, campaignID, stageID uuid.UUID, n int, result string) map[string]int {
	counts := map[string]int{"success": 0, "failure": 0, "timeout": 0}
	rows := make([][]any, n)

	for i := range rows {
		counts[result]++
		rows[i] = []any{
			deviceIDs[rng.Intn(len(deviceIDs))],
			campaignID,
			stageID,
			result,
			mustV7(),
		}
	}

	copyAttempts(ctx, pool, rows, "demo attempts")

	return counts
}

func mustV7() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		log.Fatalf("new uuidv7: %v", err)
	}
	return id
}

func rollResult(rng *rand.Rand) string {
	n := rng.Intn(100)
	switch {
	case n < config.SuccessPercent:
		return "success"
	case n < config.SuccessPercent+config.FailurePercent:
		return "failure"
	default:
		return "timeout"
	}
}

func deviceVersion(rng *rand.Rand, baseVersion, targetVersion string) string {
	if rng.Intn(100) < config.UpToDatePercent {
		return targetVersion
	}

	return baseVersion
}

func deviceStatus(rng *rand.Rand) string {
	if rng.Intn(100) < config.DecommissionedPercent {
		return "decommissioned"
	}

	return "active"
}
