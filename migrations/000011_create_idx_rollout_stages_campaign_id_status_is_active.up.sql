CREATE INDEX CONCURRENTLY idx_rollout_stages_campaign_id_status_is_active
ON rollout_stages (campaign_id)
WHERE status = 'active';
