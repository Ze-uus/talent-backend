-- name: CreateTalent :exec
INSERT INTO talents (id, user_id, category, status, skills, rate_per_day, max_tier,
  bio, portfolio_url, report_compliance)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10);

-- name: GetTalentByID :one
SELECT * FROM talents WHERE id = $1;

-- name: GetTalentByUserID :one
SELECT * FROM talents WHERE user_id = $1;

-- name: ListTalents :many
SELECT * FROM talents
WHERE ($1::text = '' OR status = $1)
  AND ($2::text = '' OR category = $2)
ORDER BY created_at DESC
LIMIT NULLIF($3::int, 0) OFFSET $4::int;

-- name: ListAllTalents :many
SELECT * FROM talents ORDER BY created_at;
