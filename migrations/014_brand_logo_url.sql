-- +goose Up
ALTER TABLE brands ADD COLUMN logo_url TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE brands DROP COLUMN logo_url;
