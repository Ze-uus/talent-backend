-- +goose Up
ALTER TABLE users
    ADD COLUMN phone_number TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN phone_number;
