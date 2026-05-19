-- name: CreatePayoutRecord :exec
INSERT INTO payout_records (id, talent_id, cycle_id, campaign_id, pipeline_type,
  status, allocated_budget, gross_base, kpb_total, gross_total, cost_per_unit,
  cap_applied, cap_exceeded, excess_forfeited, commission_rate, commission_amount,
  e_net, scale_factor, final_payout, kpb_pool_source, fallback_flagged,
  report_submitted, admin_override, override_reason, approved_by, approved_at,
  paid_at, failure_reason, retry_count)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
        $21,$22,$23,$24,$25,$26,$27,$28,$29);

-- name: GetPayoutRecord :one
SELECT * FROM payout_records WHERE talent_id = $1 AND cycle_id = $2;

-- name: ListPayoutsByCycle :many
SELECT * FROM payout_records WHERE cycle_id = $1 ORDER BY created_at;

-- name: ListPayoutsByTalent :many
SELECT * FROM payout_records WHERE talent_id = $1 ORDER BY created_at DESC;
