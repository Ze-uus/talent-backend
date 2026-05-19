-- name: CreateViewer :exec
INSERT INTO campaign_viewers (id, campaign_id, token, name, active)
VALUES ($1,$2,$3,$4,$5);

-- name: GetViewerByToken :one
SELECT * FROM campaign_viewers WHERE token = $1;

-- name: ListViewersByCampaign :many
SELECT * FROM campaign_viewers WHERE campaign_id = $1 ORDER BY created_at;

-- name: AddViewerPassword :exec
INSERT INTO viewer_passwords (id, viewer_id, password_hash, label, active)
VALUES ($1,$2,$3,$4,$5);

-- name: ListViewerPasswords :many
SELECT * FROM viewer_passwords WHERE viewer_id = $1 AND active = true ORDER BY created_at;

-- name: DeactivateViewerPassword :exec
UPDATE viewer_passwords SET active = false WHERE id = $1;

-- name: GetAllViewerPasswords :many
SELECT * FROM viewer_passwords WHERE viewer_id = $1 AND active = true;
