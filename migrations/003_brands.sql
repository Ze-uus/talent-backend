-- +goose Up
CREATE TABLE brands (
	id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	name        TEXT NOT NULL,
	shortcode   TEXT UNIQUE NOT NULL,  -- 3-letter auto-generated, e.g. "CRD"
	industry    TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	website     TEXT NOT NULL DEFAULT '',
	status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended')),
	created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_brands_shortcode ON brands(shortcode);
CREATE INDEX idx_brands_status ON brands(status);

-- Brand contacts: mutable list of people associated with a brand.
-- Each contact gets their own viewer_token and access_password_hash.
-- Contacts are NOT users — they access campaign dashboards via token+password only.
-- Up to 3 contacts per brand enforced at service layer (not DB constraint).
-- token_active=false when contact is removed (soft-delete for audit trail).
CREATE TABLE brand_contacts (
	id                      TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	brand_id                TEXT NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
	first_name              TEXT NOT NULL DEFAULT '',
	last_name               TEXT NOT NULL DEFAULT '',
	role                    TEXT NOT NULL DEFAULT '',  -- free text: CMO, Marketing Manager, etc.
	email                   TEXT NOT NULL DEFAULT '',
	whatsapp_number         TEXT NOT NULL DEFAULT '',
	viewer_token            TEXT UNIQUE NOT NULL,      -- unique access token for campaign view
	access_password_hash    TEXT NOT NULL,             -- system-generated on creation; admin can regenerate
	token_active            BOOLEAN NOT NULL DEFAULT true,
	future_user_account_id  TEXT,                      -- null until brand contact portal is built
	created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_brand_contacts_brand ON brand_contacts(brand_id);
CREATE INDEX idx_brand_contacts_token ON brand_contacts(viewer_token) WHERE token_active = true;

-- +goose Down
DROP TABLE brand_contacts;
DROP TABLE brands;
