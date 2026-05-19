-- name: CreateTrackingLink :exec
INSERT INTO tracking_links (id, campaign_id, cycle_id, talent_id, token, active)
VALUES ($1,$2,$3,$4,$5,$6);

-- name: GetTrackingLinkByToken :one
SELECT * FROM tracking_links WHERE token = $1;

-- name: ListTrackingLinksByCycle :many
SELECT * FROM tracking_links WHERE cycle_id = $1 ORDER BY created_at;
