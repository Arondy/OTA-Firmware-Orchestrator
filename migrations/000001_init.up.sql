CREATE TYPE DEVICE_STATUS AS ENUM ('active', 'decommissioned');
CREATE TYPE ROLLOUT_CAMPAIGNS_STATUS AS ENUM (
    'draft', 'running', 'paused', 'completed', 'rolled_back'
);
CREATE TYPE ROLLOUT_STAGES_STATUS AS ENUM (
    'pending', 'active', 'passed', 'failed'
);


CREATE TABLE devices (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    device_model TEXT NOT NULL,
    current_version TEXT NOT NULL,
    status DEVICE_STATUS NOT NULL DEFAULT 'active',
    last_seen TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE firmware_versions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    device_model TEXT NOT NULL,
    fw_version TEXT NOT NULL,
    fw_checksum TEXT NOT NULL,
    binary_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (device_model, fw_version)
);

CREATE TABLE rollout_campaigns (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    firmware_version_id UUID NOT NULL REFERENCES firmware_versions (id),
    device_model TEXT NOT NULL,
    status ROLLOUT_CAMPAIGNS_STATUS NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

-- Only one running campaign for one model at the same time
CREATE UNIQUE INDEX one_running_campaign_per_model
ON rollout_campaigns (device_model)
WHERE status = 'running';

CREATE TABLE rollout_stages (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    campaign_id UUID NOT NULL REFERENCES rollout_campaigns (id),
    order_index INT NOT NULL,
    target_percent INT NOT NULL CHECK (target_percent BETWEEN 1 AND 100),
    min_sample_size INT NOT NULL,
    success_threshold REAL NOT NULL CHECK (
        success_threshold > 0 AND success_threshold <= 1
    ),
    status ROLLOUT_STAGES_STATUS NOT NULL DEFAULT 'pending',
    entered_at TIMESTAMPTZ,
    UNIQUE (campaign_id, order_index)
);
