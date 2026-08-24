-- +goose Up
ALTER TABLE cycles DROP CONSTRAINT IF EXISTS cycles_status_check;
ALTER TABLE cycles
    ADD CONSTRAINT cycles_status_check
    CHECK (status IN ('pending','active','paused','closed'));

-- +goose Down
UPDATE cycles SET status = 'pending' WHERE status = 'paused';
ALTER TABLE cycles DROP CONSTRAINT IF EXISTS cycles_status_check;
ALTER TABLE cycles
    ADD CONSTRAINT cycles_status_check
    CHECK (status IN ('pending','active','closed'));
