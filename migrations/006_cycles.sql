-- +goose Up
CREATE TABLE cycles (
	id               TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	human_id         TEXT UNIQUE NOT NULL, -- "CRD-26-01-C2" format
	campaign_id      TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
	cycle_number     INT NOT NULL,
	status           TEXT NOT NULL DEFAULT 'pending'
	                     CHECK (status IN ('pending','active','closed')),
	cycle_budget     NUMERIC(14,2) NOT NULL,
	remaining_budget NUMERIC(14,2) NOT NULL,
	cycle_objective  TEXT NOT NULL DEFAULT '',
	campaign_type    TEXT NOT NULL,
	kpb_config       JSONB NOT NULL DEFAULT '[]',
	z_factor         NUMERIC(4,2) NOT NULL DEFAULT 1.0,
	start_date       TIMESTAMPTZ NOT NULL,
	end_date         TIMESTAMPTZ NOT NULL,
	created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(campaign_id, cycle_number)
);

CREATE TABLE budget_slots (
	id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	cycle_id    TEXT NOT NULL REFERENCES cycles(id) ON DELETE CASCADE,
	tier_value  INT NOT NULL,
	slot_index  INT NOT NULL,
	allocated   BOOLEAN NOT NULL DEFAULT false,
	talent_id   TEXT REFERENCES talents(id),
	UNIQUE(cycle_id, tier_value, slot_index)
);

CREATE TABLE tracking_links (
	id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	campaign_id TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
	cycle_id    TEXT NOT NULL REFERENCES cycles(id) ON DELETE CASCADE,
	talent_id   TEXT NOT NULL REFERENCES talents(id),
	token       TEXT UNIQUE NOT NULL,
	active      BOOLEAN NOT NULL DEFAULT true,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- pipeline_type and valid_lead are critical for payout algorithm branching.
-- fallback_flagged is set by cycle-end process when downstream data unavailable.
CREATE TABLE conversion_events (
	id               TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	link_token       TEXT NOT NULL,
	talent_id        TEXT NOT NULL,
	campaign_id      TEXT NOT NULL,
	cycle_id         TEXT NOT NULL,
	pipeline_type    TEXT NOT NULL CHECK (pipeline_type IN ('direct_traffic','lead_validation')),
	event_type       TEXT NOT NULL,
	kpb_type         TEXT,
	valid_lead       BOOLEAN NOT NULL DEFAULT false,
	fallback_flagged BOOLEAN NOT NULL DEFAULT false,
	idempotency_key  TEXT UNIQUE NOT NULL,
	occurred_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conversions_talent_cycle ON conversion_events(talent_id, cycle_id);
CREATE INDEX idx_conversions_cycle ON conversion_events(cycle_id);
CREATE INDEX idx_conversions_occurred ON conversion_events(occurred_at);
CREATE INDEX idx_conversions_fallback ON conversion_events(cycle_id, fallback_flagged);

-- Full assignment audit data per Story 17 P.10 and Story 18 A.11
CREATE TABLE talent_assignments (
	talent_id         TEXT NOT NULL REFERENCES talents(id),
	campaign_id       TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
	cycle_id          TEXT NOT NULL REFERENCES cycles(id) ON DELETE CASCADE,
	slot_id           TEXT NOT NULL REFERENCES budget_slots(id),
	role_label        TEXT NOT NULL DEFAULT '',
	status            TEXT NOT NULL DEFAULT 'active'
	                      CHECK (status IN ('active','completed','removed_forfeit','removed_payout')),
	assignment_source TEXT NOT NULL DEFAULT 'algorithm'
	                      CHECK (assignment_source IN ('algorithm','pinned','manual')),
	pdc_mode          TEXT NOT NULL DEFAULT 'cold_start'
	                      CHECK (pdc_mode IN ('cold_start','stec','ltec')),
	pdc_value         NUMERIC(10,4) NOT NULL DEFAULT 0,
	match_score       NUMERIC(6,4) NOT NULL DEFAULT 0,
	match_dm          NUMERIC(6,4) NOT NULL DEFAULT 0,
	match_gp          NUMERIC(6,4) NOT NULL DEFAULT 0,
	match_oh          NUMERIC(6,4) NOT NULL DEFAULT 0,
	pinned_tier       INT NOT NULL DEFAULT 0,
	effective_tier    INT NOT NULL DEFAULT 0,
	breakout_flag     BOOLEAN NOT NULL DEFAULT false,
	assigned_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (talent_id, cycle_id)
);

-- +goose Down
DROP TABLE talent_assignments;
DROP TABLE conversion_events;
DROP TABLE tracking_links;
DROP TABLE budget_slots;
DROP TABLE cycles;
