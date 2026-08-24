-- +goose Up
-- Seed cold-start baselines for active talents that never received one.
INSERT INTO talent_baselines (talent_id, alpha_lt, beta_lt, lambda_lt, sigma_hist, delta_lt)
SELECT t.id, c.median, 1.0, c.median, 0, 0.97
FROM talents t
JOIN category_baselines c ON c.category = t.category
WHERE t.status = 'active'
  AND NOT EXISTS (SELECT 1 FROM talent_baselines b WHERE b.talent_id = t.id);

-- +goose Down
-- Irreversible backfill; no-op on down.
SELECT 1;
