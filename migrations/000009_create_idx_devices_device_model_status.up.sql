CREATE INDEX CONCURRENTLY idx_devices_device_model_status
ON devices (device_model, status);
