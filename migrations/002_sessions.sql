-- +goose Up
CREATE TABLE sessions (
	id             TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	user_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	token          TEXT UNIQUE NOT NULL,
	ip_address     TEXT NOT NULL DEFAULT '',
	user_agent     TEXT NOT NULL DEFAULT '',
	last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	expires_at     TIMESTAMPTZ NOT NULL,
	invalidated    BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_sessions_token ON sessions(token) WHERE invalidated = false;
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

-- +goose Down
DROP TABLE sessions;
