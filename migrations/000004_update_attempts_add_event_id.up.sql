ALTER TABLE update_attempts ADD COLUMN event_id UUID;

UPDATE update_attempts SET event_id = uuidv7()
WHERE event_id IS NULL;
