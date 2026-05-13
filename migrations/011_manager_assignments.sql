-- +goose Up
CREATE TABLE manager_campaign_assignments (
	manager_id  TEXT NOT NULL REFERENCES users(id),
	campaign_id TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
	assigned_by TEXT NOT NULL REFERENCES users(id),
	assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (manager_id, campaign_id)
);

CREATE INDEX idx_manager_campaigns ON manager_campaign_assignments(manager_id);

-- +goose Down
DROP TABLE manager_campaign_assignments;
