-- +goose Up
ALTER TABLE users
	ADD COLUMN status TEXT NOT NULL DEFAULT 'active'
		CHECK (status IN ('active','suspended','banned','deleted','pending','rejected')),
	ADD COLUMN deleted_at TIMESTAMPTZ;

-- Copy talent workflow status onto the linked user.
UPDATE users u
SET status = t.status
FROM talents t
WHERE t.user_id = u.id
  AND t.status IN ('active','suspended','pending','rejected');

-- Deactivated non-talent users (completed invites) → suspended.
UPDATE users
SET status = 'suspended'
WHERE active = false
  AND role <> 'talent'
  AND invite_token IS NULL
  AND status = 'active';

CREATE INDEX idx_users_status ON users(status);

-- +goose Down
DROP INDEX IF EXISTS idx_users_status;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS status;
