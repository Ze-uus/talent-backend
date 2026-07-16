-- name: GetCycleState :one
SELECT * FROM talent_cycle_states WHERE talent_id = $1 AND cycle_id = $2;

-- name: UpsertCycleState :exec
INSERT INTO talent_cycle_states (talent_id, cycle_id, cycle_number, pdc_allocated,
  pdc_next, pattern, delta_st, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (talent_id, cycle_id) DO UPDATE SET
  pdc_allocated = EXCLUDED.pdc_allocated,
  pdc_next      = EXCLUDED.pdc_next,
  pattern       = EXCLUDED.pattern,
  delta_st      = EXCLUDED.delta_st,
  updated_at    = EXCLUDED.updated_at;

-- name: GetDailyOutputs :many
SELECT COUNT(*)::float8 AS daily_count
FROM conversion_events
WHERE talent_id = $1 AND cycle_id = $2
GROUP BY DATE(occurred_at AT TIME ZONE 'UTC')
ORDER BY DATE(occurred_at AT TIME ZONE 'UTC');

-- name: GetTalentBaseline :one
SELECT * FROM talent_baselines WHERE talent_id = $1;

-- name: UpsertTalentBaseline :exec
INSERT INTO talent_baselines (talent_id, alpha_lt, beta_lt, lambda_lt, sigma_hist,
  delta_lt, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,NOW())
ON CONFLICT (talent_id) DO UPDATE SET
  alpha_lt   = EXCLUDED.alpha_lt,
  beta_lt    = EXCLUDED.beta_lt,
  lambda_lt  = EXCLUDED.lambda_lt,
  sigma_hist = EXCLUDED.sigma_hist,
  delta_lt   = EXCLUDED.delta_lt,
  updated_at = EXCLUDED.updated_at;

-- name: GetCategoryBaseline :one
SELECT * FROM category_baselines WHERE category = $1;

-- name: GetTalentTodayOutput :one
SELECT COUNT(*)::float8 FROM conversion_events
WHERE talent_id = $1
  AND DATE(occurred_at AT TIME ZONE 'UTC') = DATE($2 AT TIME ZONE 'UTC');

-- name: GetTalentOutputWindow :many
SELECT COUNT(*)::float8 AS daily_count
FROM conversion_events
WHERE talent_id = $1
  AND occurred_at >= NOW() - ($2::int * INTERVAL '1 day')
GROUP BY DATE(occurred_at AT TIME ZONE 'UTC')
ORDER BY DATE(occurred_at AT TIME ZONE 'UTC');
