-- +goose Up
CREATE TABLE campaigns (
	id                TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	human_id          TEXT UNIQUE NOT NULL, -- "CRD-26-01" format
	brand_id          TEXT NOT NULL REFERENCES brands(id),
	name              TEXT NOT NULL,
	status            TEXT NOT NULL DEFAULT 'draft'
	                      CHECK (status IN ('draft','active','paused','closed','archived')),
	campaign_type     TEXT NOT NULL CHECK (campaign_type IN ('direct_traffic','lead_validation')),
	total_budget      NUMERIC(14,2) NOT NULL,
	remaining_budget  NUMERIC(14,2) NOT NULL,
	market_cap        TEXT NOT NULL DEFAULT 'M',
	audience          TEXT NOT NULL DEFAULT '',
	target_cpa        NUMERIC(10,2) NOT NULL DEFAULT 100,
	max_cpa           NUMERIC(10,2) NOT NULL DEFAULT 200,
	urgency_level     TEXT NOT NULL DEFAULT 'normal'
	                      CHECK (urgency_level IN ('low','normal','high')),
	cycle_length      INT NOT NULL DEFAULT 7 CHECK (cycle_length IN (5,6,7,8,9,10)),
	creators_allowed  BOOLEAN NOT NULL DEFAULT false,
	start_date        TIMESTAMPTZ NOT NULL,
	end_date          TIMESTAMPTZ NOT NULL,
	created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_campaigns_status ON campaigns(status);
CREATE INDEX idx_campaigns_brand_id ON campaigns(brand_id);

-- Generic viewer access (not tied to brand contacts — separate mechanism)
CREATE TABLE campaign_viewers (
	id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	campaign_id TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
	token       TEXT UNIQUE NOT NULL,
	name        TEXT NOT NULL DEFAULT '',
	active      BOOLEAN NOT NULL DEFAULT true,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE viewer_passwords (
	id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	viewer_id     TEXT NOT NULL REFERENCES campaign_viewers(id) ON DELETE CASCADE,
	password_hash TEXT NOT NULL,
	label         TEXT NOT NULL DEFAULT 'primary',
	active        BOOLEAN NOT NULL DEFAULT true,
	created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE viewer_passwords;
DROP TABLE campaign_viewers;
DROP TABLE campaigns;
