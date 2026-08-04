-- +goose Up
-- Replace status CHECK to include 'invited'.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_status_check;
ALTER TABLE users
	ADD CONSTRAINT users_status_check
	CHECK (status IN ('active','suspended','banned','deleted','pending','rejected','invited'));

-- Open staff invites that were wrongly labelled active → invited.
UPDATE users
SET status = 'invited'
WHERE active = false
  AND status = 'active'
  AND invite_token IS NOT NULL;

-- Legacy deactivated staff (no open invite) still labelled active → suspended.
UPDATE users
SET status = 'suspended'
WHERE active = false
  AND status = 'active'
  AND role <> 'talent'
  AND invite_token IS NULL;

-- +goose Down
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_status_check;
-- Move invited rows back before restoring old CHECK.
UPDATE users SET status = 'active' WHERE status = 'invited';
ALTER TABLE users
	ADD CONSTRAINT users_status_check
	CHECK (status IN ('active','suspended','banned','deleted','pending','rejected'));
