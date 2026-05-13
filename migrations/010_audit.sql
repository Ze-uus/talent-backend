-- +goose Up
CREATE TABLE audit_log (
	id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	actor_id     TEXT NOT NULL REFERENCES users(id),
	action_type  TEXT NOT NULL,
	entity_type  TEXT NOT NULL,
	entity_id    TEXT NOT NULL,
	before_state JSONB,
	after_state  JSONB,
	created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_actor ON audit_log(actor_id);

-- +goose Down
DROP TABLE audit_log;
