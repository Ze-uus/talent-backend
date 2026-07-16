-- +goose Up
CREATE TABLE talent_cycle_states (
	talent_id      TEXT NOT NULL REFERENCES talents(id),
	cycle_id       TEXT NOT NULL REFERENCES cycles(id) ON DELETE CASCADE,
	cycle_number   INT NOT NULL DEFAULT 1,
	pdc_allocated  NUMERIC(10,4) NOT NULL DEFAULT 0,
	pdc_next       NUMERIC(10,4) NOT NULL DEFAULT 0,
	pattern        TEXT NOT NULL DEFAULT 'stable',
	delta_st       NUMERIC(6,4) NOT NULL DEFAULT 0.20,
	updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (talent_id, cycle_id)
);

CREATE TABLE talent_baselines (
	talent_id  TEXT PRIMARY KEY REFERENCES talents(id) ON DELETE CASCADE,
	alpha_lt   NUMERIC(12,4) NOT NULL DEFAULT 0,
	beta_lt    NUMERIC(12,4) NOT NULL DEFAULT 1,
	lambda_lt  NUMERIC(10,4) NOT NULL DEFAULT 0,
	sigma_hist NUMERIC(10,4) NOT NULL DEFAULT 0,
	delta_lt   NUMERIC(6,4) NOT NULL DEFAULT 0.97,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE category_baselines (
	category TEXT PRIMARY KEY,
	median   NUMERIC(10,4) NOT NULL
);

INSERT INTO category_baselines VALUES
	('student',   2),
	('micro',     5),
	('community', 15);

-- +goose Down
DROP TABLE category_baselines;
DROP TABLE talent_baselines;
DROP TABLE talent_cycle_states;
