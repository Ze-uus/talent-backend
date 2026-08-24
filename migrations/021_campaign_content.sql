-- +goose Up
ALTER TABLE campaigns
    ADD COLUMN content JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE cycles
    ADD COLUMN content_override JSONB;

-- +goose Down
ALTER TABLE cycles DROP COLUMN content_override;
ALTER TABLE campaigns DROP COLUMN content;
