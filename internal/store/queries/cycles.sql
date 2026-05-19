-- name: CreateCycle :exec
INSERT INTO cycles (id, human_id, campaign_id, cycle_number, status, cycle_budget,
  remaining_budget, cycle_objective, campaign_type, kpb_config, z_factor, start_date, end_date)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13);

-- name: GetCycleByID :one
SELECT * FROM cycles WHERE id = $1;

-- name: ListCyclesByCampaign :many
SELECT * FROM cycles WHERE campaign_id = $1 ORDER BY cycle_number;

-- name: GetActiveCycles :many
SELECT * FROM cycles WHERE status = 'active' ORDER BY start_date;

-- name: CloseCycle :exec
UPDATE cycles
SET status = 'closed', remaining_budget = remaining_budget - $2, updated_at = NOW()
WHERE id = $1;

-- name: ListSlotsByCycle :many
SELECT * FROM budget_slots WHERE cycle_id = $1 ORDER BY tier_value, slot_index;

-- name: AssignSlot :exec
UPDATE budget_slots SET allocated = true, talent_id = $2 WHERE id = $1;
