-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
	id                     TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
	email                  TEXT UNIQUE NOT NULL,
	password_hash          TEXT,
	role                   TEXT NOT NULL CHECK (role IN ('admin','talent','viewer')),
	provider               TEXT NOT NULL DEFAULT 'local' CHECK (provider IN ('local','google')),
	google_id              TEXT UNIQUE,
	full_name              TEXT NOT NULL DEFAULT '',
	avatar_url             TEXT NOT NULL DEFAULT '',
	totp_secret            TEXT NOT NULL DEFAULT '',
	totp_enabled           BOOLEAN NOT NULL DEFAULT false,
	totp_verified          BOOLEAN NOT NULL DEFAULT false,
	totp_last_verified_at  TIMESTAMPTZ,
	invite_token           TEXT UNIQUE,
	invite_expires_at      TIMESTAMPTZ,
	active                 BOOLEAN NOT NULL DEFAULT true,
	created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_google_id ON users(google_id) WHERE google_id IS NOT NULL;
CREATE INDEX idx_users_invite_token ON users(invite_token) WHERE invite_token IS NOT NULL;

-- +goose Down
DROP TABLE users;
