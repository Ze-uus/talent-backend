-- +goose Up
CREATE TABLE email_dispatches (
	id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	entity_type  TEXT NOT NULL,
	entity_id    TEXT NOT NULL,
	template_key TEXT NOT NULL,
	recipient    TEXT NOT NULL,
	sent_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX email_dispatches_uniq
	ON email_dispatches (entity_type, entity_id, template_key, recipient);

CREATE INDEX email_dispatches_entity
	ON email_dispatches (entity_type, entity_id);

-- +goose Down
DROP TABLE email_dispatches;
