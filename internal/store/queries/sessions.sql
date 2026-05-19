-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, token, ip_address, user_agent, expires_at)
VALUES ($1,$2,$3,$4,$5,$6);

-- name: GetSession :one
SELECT * FROM sessions
WHERE token = $1 AND invalidated = false AND expires_at > NOW();

-- name: TouchSession :exec
UPDATE sessions SET last_active_at = $2 WHERE token = $1;

-- name: InvalidateSession :exec
UPDATE sessions SET invalidated = true WHERE token = $1;

-- name: InvalidateAllUserSessions :exec
UPDATE sessions SET invalidated = true WHERE user_id = $1;

-- name: ListSessionsByUser :many
SELECT * FROM sessions WHERE user_id = $1 ORDER BY last_active_at DESC;
