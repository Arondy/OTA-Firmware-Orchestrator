CREATE TYPE UPDATE_ATTEMPTS_RESULT AS ENUM ('success', 'failure', 'timeout');


CREATE TABLE update_attempts (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    device_id UUID NOT NULL REFERENCES devices (id),
    campaign_id UUID NOT NULL REFERENCES rollout_campaigns (id),
    stage_id UUID NOT NULL REFERENCES rollout_stages (id),
    result UPDATE_ATTEMPTS_RESULT NOT NULL,
    reported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
