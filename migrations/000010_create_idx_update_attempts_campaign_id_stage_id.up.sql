CREATE INDEX CONCURRENTLY idx_update_attempts_campaign_id_stage_id
ON update_attempts (campaign_id, stage_id);
