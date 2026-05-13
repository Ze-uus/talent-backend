-- +goose Up
CREATE TABLE talent_reports (
	id                TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	talent_id         TEXT NOT NULL REFERENCES talents(id),
	cycle_id          TEXT NOT NULL REFERENCES cycles(id) ON DELETE CASCADE,
	submitted_at      TIMESTAMPTZ,
	deadline_at       TIMESTAMPTZ NOT NULL,
	audience_reached  TEXT,
	offer_promoted    TEXT,
	friction_score    NUMERIC(4,2),
	audience_reaction TEXT,
	blocker_category  TEXT,
	status            TEXT NOT NULL DEFAULT 'pending'
	                      CHECK (status IN ('pending','submitted','late','flagged')),
	created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(talent_id, cycle_id)
);

CREATE TABLE cycle_reports (
	id                               TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	cycle_id                         TEXT UNIQUE NOT NULL REFERENCES cycles(id),
	offer_decision_label             TEXT,
	max_spend_decision_label         TEXT,
	predicted_daily_conversions_next NUMERIC(10,4),
	generated_at                     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE cycle_reports;
DROP TABLE talent_reports;
