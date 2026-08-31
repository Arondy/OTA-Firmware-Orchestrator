package postgres

import (
	"context"
	"fmt"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AppliedDecisionRepo struct {
	*DB
}

func NewAppliedDecisionRepo(db *DB) *AppliedDecisionRepo {
	return &AppliedDecisionRepo{DB: db}
}

func (r *AppliedDecisionRepo) Get(ctx context.Context, decisionID uuid.UUID) (domain.AppliedDecision, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	SELECT decision_id, campaign_id, decision_type, applied_at
	FROM applied_decisions
	WHERE decision_id = $1
	`

	row := exec.QueryRow(reqCtx, query, decisionID)

	var decision domain.AppliedDecision
	err := row.Scan(
		&decision.DecisionID,
		&decision.CampaignID,
		&decision.DecisionType,
		&decision.AppliedAt,
	)
	if err == pgx.ErrNoRows {
		return domain.AppliedDecision{}, domain.ErrAppliedDecisionNotFound
	} else if err != nil {
		return domain.AppliedDecision{}, fmt.Errorf("failed to get decision: %w", err)
	}

	return decision, nil
}

func (r *AppliedDecisionRepo) Create(ctx context.Context, decision domain.AppliedDecision) (domain.AppliedDecision, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	exec := r.exec(reqCtx)

	query := `
	INSERT INTO applied_decisions (decision_id, campaign_id, decision_type)
	VALUES ($1, $2, $3)
	ON CONFLICT (decision_id)
	DO UPDATE SET decision_id = EXCLUDED.decision_id
	RETURNING decision_id, campaign_id, decision_type, applied_at
	`

	row := exec.QueryRow(reqCtx, query, decision.DecisionID, decision.CampaignID, decision.DecisionType)

	var createdDecision domain.AppliedDecision
	err := row.Scan(
		&createdDecision.DecisionID,
		&createdDecision.CampaignID,
		&createdDecision.DecisionType,
		&createdDecision.AppliedAt,
	)
	if err != nil {
		return domain.AppliedDecision{}, fmt.Errorf("failed to create decision: %w", err)
	}

	return createdDecision, nil
}
