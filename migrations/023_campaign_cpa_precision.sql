-- +goose Up
ALTER TABLE campaigns
    ALTER COLUMN target_cpa TYPE NUMERIC(14,2),
    ALTER COLUMN max_cpa TYPE NUMERIC(14,2);

-- +goose Down
ALTER TABLE campaigns
    ALTER COLUMN target_cpa TYPE NUMERIC(10,2),
    ALTER COLUMN max_cpa TYPE NUMERIC(10,2);
