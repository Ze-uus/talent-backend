-- +goose Up
-- Backfill talent profiles for users with role=talent who never got a talents row.
INSERT INTO talents (id, user_id, category, status, skills, rate_per_day, max_tier, bio, portfolio_url, report_compliance)
SELECT
	gen_random_uuid()::text,
	u.id,
	'student',
	CASE u.status
		WHEN 'active' THEN 'active'
		WHEN 'rejected' THEN 'rejected'
		WHEN 'suspended' THEN 'suspended'
		WHEN 'banned' THEN 'suspended'
		WHEN 'deleted' THEN 'suspended'
		ELSE 'pending'
	END,
	'{}',
	0,
	5000,
	'',
	'',
	1.0
FROM users u
WHERE u.role = 'talent'
  AND NOT EXISTS (SELECT 1 FROM talents t WHERE t.user_id = u.id);

-- +goose Down
-- Irreversible backfill; no-op on down.
SELECT 1;
