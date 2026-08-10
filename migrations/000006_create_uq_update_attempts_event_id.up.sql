CREATE UNIQUE INDEX CONCURRENTLY uq_update_attempts_event_id
ON update_attempts (event_id);
