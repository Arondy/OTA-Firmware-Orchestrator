CREATE TYPE DECISION_TYPE AS ENUM ('advance_stage', 'rollback');

CREATE TABLE applied_decisions (
    decision_id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES rollout_campaigns (id),
    decision_type DECISION_TYPE NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
