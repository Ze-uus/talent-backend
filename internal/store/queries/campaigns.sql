-- name: CreateCampaign :exec
INSERT INTO campaigns (id, human_id, brand_id, name, status, campaign_type,
  total_budget, remaining_budget, market_cap, audience, target_cpa, max_cpa,
  urgency_level, cycle_length, creators_allowed, start_date, end_date)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17);

-- name: GetCampaignByID :one
SELECT * FROM campaigns WHERE id = $1;

-- name: GetCampaignByHumanID :one
SELECT * FROM campaigns WHERE human_id = $1;

-- name: ListCampaigns :many
SELECT * FROM campaigns
WHERE ($1::text = '' OR status = $1)
  AND ($2::text = '' OR brand_id = $2)
ORDER BY created_at DESC
LIMIT NULLIF($3::int, 0) OFFSET $4::int;

-- name: ListActiveCampaigns :many
SELECT * FROM campaigns WHERE status = 'active' ORDER BY start_date;

-- name: DecrementRemainingBudget :exec
UPDATE campaigns
SET remaining_budget = remaining_budget - $2, updated_at = NOW()
WHERE id = $1;

-- name: CountCampaignsByBrandMonth :one
SELECT COUNT(*) FROM campaigns
WHERE brand_id = $1
  AND DATE_TRUNC('month', created_at) = DATE_TRUNC('month', NOW());

-- name: GetCampaignsByManagerID :many
SELECT c.* FROM campaigns c
JOIN manager_campaign_assignments m ON m.campaign_id = c.id
WHERE m.manager_id = $1
ORDER BY c.created_at DESC;

-- name: AssignManagerToCampaign :exec
INSERT INTO manager_campaign_assignments (manager_id, campaign_id, assigned_by)
VALUES ($1,$2,$3) ON CONFLICT DO NOTHING;

-- name: UnassignManagerFromCampaign :exec
DELETE FROM manager_campaign_assignments
WHERE manager_id = $1 AND campaign_id = $2;

-- name: ListManagersByCampaignID :many
SELECT u.* FROM users u
JOIN manager_campaign_assignments m ON m.manager_id = u.id
WHERE m.campaign_id = $1
ORDER BY u.full_name;

