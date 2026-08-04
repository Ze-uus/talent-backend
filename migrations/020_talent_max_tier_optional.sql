-- +goose Up
-- Zero means the backend derives the tier automatically. Remove the legacy
-- 5,000 ceiling assigned to backfilled and default-created talent profiles.
ALTER TABLE talents ALTER COLUMN max_tier SET DEFAULT 0;
UPDATE talents SET max_tier = 0 WHERE max_tier = 5000;

-- +goose Down
ALTER TABLE talents ALTER COLUMN max_tier SET DEFAULT 5000;
UPDATE talents SET max_tier = 5000 WHERE max_tier = 0;
