-- search running campaign by device_model for checkin
CREATE INDEX CONCURRENTLY idx_campaigns_device_model_status
ON rollout_campaigns (device_model, status);
