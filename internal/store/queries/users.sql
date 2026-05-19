-- name: CreateUser :exec
INSERT INTO users (email, password_hash, role, provider, google_id, full_name,
  avatar_url, totp_secret, totp_enabled, totp_verified, totp_last_verified_at,
  invite_token, invite_expires_at, active)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14);

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByGoogleID :one
SELECT * FROM users WHERE google_id = $1;

-- name: GetUserByInviteToken :one
SELECT * FROM users WHERE invite_token = $1 AND invite_expires_at > NOW();

-- name: ListUsers :many
SELECT * FROM users
WHERE ($1::text = '' OR role = $1)
  AND ($2::boolean IS NULL OR active = $2)
ORDER BY created_at DESC;
