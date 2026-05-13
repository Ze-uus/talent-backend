-- +goose Up
-- Full per-advocate per-cycle payout audit record (Story 17 P.10).
CREATE TABLE payout_records (
	id                  TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	talent_id           TEXT NOT NULL REFERENCES talents(id),
	cycle_id            TEXT NOT NULL REFERENCES cycles(id) ON DELETE CASCADE,
	campaign_id         TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
	pipeline_type       TEXT NOT NULL CHECK (pipeline_type IN ('direct_traffic','lead_validation')),
	status              TEXT NOT NULL DEFAULT 'report_pending'
	                        CHECK (status IN ('report_pending','pending','approved','paid','forfeited','failed')),

	allocated_budget    NUMERIC(12,2) NOT NULL DEFAULT 0,
	gross_base          NUMERIC(12,2) NOT NULL DEFAULT 0,
	kpb_total           NUMERIC(12,2) NOT NULL DEFAULT 0,
	gross_total         NUMERIC(12,2) NOT NULL DEFAULT 0,

	cost_per_unit       NUMERIC(12,2) NOT NULL DEFAULT 0,
	cap_applied         NUMERIC(12,2) NOT NULL DEFAULT 0,
	cap_exceeded        BOOLEAN NOT NULL DEFAULT false,
	excess_forfeited    NUMERIC(12,2) NOT NULL DEFAULT 0,

	commission_rate     NUMERIC(5,4) NOT NULL DEFAULT 0,
	commission_amount   NUMERIC(12,2) NOT NULL DEFAULT 0,
	e_net               NUMERIC(12,2) NOT NULL DEFAULT 0,

	scale_factor        NUMERIC(8,6) NOT NULL DEFAULT 1.0,
	final_payout        NUMERIC(12,2) NOT NULL DEFAULT 0,

	kpb_pool_source     TEXT NOT NULL DEFAULT 'n/a',
	fallback_flagged    BOOLEAN NOT NULL DEFAULT false,
	report_submitted    BOOLEAN NOT NULL DEFAULT false,
	admin_override      BOOLEAN NOT NULL DEFAULT false,
	override_reason     TEXT NOT NULL DEFAULT '',
	approved_by         TEXT,
	approved_at         TIMESTAMPTZ,
	paid_at             TIMESTAMPTZ,
	failure_reason      TEXT NOT NULL DEFAULT '',
	retry_count         INT NOT NULL DEFAULT 0,
	created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

	UNIQUE(talent_id, cycle_id)
);

CREATE INDEX idx_payout_cycle ON payout_records(cycle_id);
CREATE INDEX idx_payout_status ON payout_records(status);
CREATE INDEX idx_payout_talent ON payout_records(talent_id);

-- +goose Down
DROP TABLE payout_records;
