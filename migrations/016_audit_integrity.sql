-- +goose Up
ALTER TABLE audit_log
	ALTER COLUMN actor_id DROP NOT NULL,
	ADD COLUMN request_id TEXT NOT NULL DEFAULT '',
	ADD COLUMN seq BIGSERIAL,
	ADD COLUMN prev_hash TEXT NOT NULL DEFAULT '',
	ADD COLUMN entry_hash TEXT NOT NULL DEFAULT '',
	ADD COLUMN signature TEXT NOT NULL DEFAULT '',
	ADD COLUMN archive_uri TEXT NOT NULL DEFAULT '',
	ADD COLUMN ip_address TEXT NOT NULL DEFAULT '',
	ADD COLUMN user_agent TEXT NOT NULL DEFAULT '';

-- Drop FK so soft-deleted / system actors do not block inserts; keep index.
ALTER TABLE audit_log DROP CONSTRAINT IF EXISTS audit_log_actor_id_fkey;

CREATE UNIQUE INDEX idx_audit_seq ON audit_log(seq);
CREATE INDEX idx_audit_created ON audit_log(created_at DESC);
CREATE INDEX idx_audit_action ON audit_log(action_type);

-- Tip row for hash-chain locking.
CREATE TABLE audit_chain_tip (
	id         INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
	last_hash  TEXT NOT NULL DEFAULT repeat('0', 64),
	last_seq   BIGINT NOT NULL DEFAULT 0,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO audit_chain_tip (id, last_hash, last_seq) VALUES (1, repeat('0', 64), 0);

-- Append-only enforcement.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION audit_log_append_only() RETURNS trigger AS $$
BEGIN
	RAISE EXCEPTION 'audit_log is append-only';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_audit_log_no_update
	BEFORE UPDATE ON audit_log
	FOR EACH ROW EXECUTE FUNCTION audit_log_append_only();

CREATE TRIGGER trg_audit_log_no_delete
	BEFORE DELETE ON audit_log
	FOR EACH ROW EXECUTE FUNCTION audit_log_append_only();

-- +goose Down
DROP TRIGGER IF EXISTS trg_audit_log_no_delete ON audit_log;
DROP TRIGGER IF EXISTS trg_audit_log_no_update ON audit_log;
DROP FUNCTION IF EXISTS audit_log_append_only();
DROP TABLE IF EXISTS audit_chain_tip;
DROP INDEX IF EXISTS idx_audit_action;
DROP INDEX IF EXISTS idx_audit_created;
DROP INDEX IF EXISTS idx_audit_seq;
ALTER TABLE audit_log
	DROP COLUMN IF EXISTS user_agent,
	DROP COLUMN IF EXISTS ip_address,
	DROP COLUMN IF EXISTS archive_uri,
	DROP COLUMN IF EXISTS signature,
	DROP COLUMN IF EXISTS entry_hash,
	DROP COLUMN IF EXISTS prev_hash,
	DROP COLUMN IF EXISTS seq,
	DROP COLUMN IF EXISTS request_id;
ALTER TABLE audit_log ALTER COLUMN actor_id SET NOT NULL;
