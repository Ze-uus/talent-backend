-- name: CreateAssignment :exec
INSERT INTO talent_assignments (talent_id, campaign_id, cycle_id, slot_id, role_label,
  status, assignment_source, pdc_mode, pdc_value, match_score, match_dm, match_gp,
  match_oh, pinned_tier, effective_tier, breakout_flag)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16);

-- name: GetAssignment :one
SELECT * FROM talent_assignments WHERE talent_id = $1 AND cycle_id = $2;

-- name: ListAssignedTalents :many
SELECT * FROM talent_assignments WHERE cycle_id = $1 ORDER BY effective_tier DESC;

-- name: ListAssignmentsByTalent :many
SELECT * FROM talent_assignments WHERE talent_id = $1 ORDER BY assigned_at DESC;
