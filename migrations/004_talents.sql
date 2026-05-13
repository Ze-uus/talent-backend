-- +goose Up
CREATE TABLE talents (
	id                 TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	user_id            TEXT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	category           TEXT NOT NULL CHECK (category IN ('student','micro','community')),
	status             TEXT NOT NULL DEFAULT 'pending'
	                       CHECK (status IN ('pending','active','suspended','rejected')),
	skills             TEXT[] NOT NULL DEFAULT '{}',
	rate_per_day       NUMERIC(12,2) NOT NULL DEFAULT 0,
	max_tier           INT NOT NULL DEFAULT 5000,
	bio                TEXT NOT NULL DEFAULT '',
	portfolio_url      TEXT NOT NULL DEFAULT '',
	report_compliance  NUMERIC(5,4) NOT NULL DEFAULT 1.0,
	created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_talents_status ON talents(status);
CREATE INDEX idx_talents_category ON talents(category);

-- +goose Down
DROP TABLE talents;
