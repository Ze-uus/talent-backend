-- name: CreateBrand :exec
INSERT INTO brands (id, name, shortcode, industry, description, website, status)
VALUES ($1,$2,$3,$4,$5,$6,$7);

-- name: GetBrandByID :one
SELECT * FROM brands WHERE id = $1;

-- name: GetBrandByShortcode :one
SELECT * FROM brands WHERE shortcode = $1;

-- name: ListBrands :many
SELECT * FROM brands
WHERE ($1::text = '' OR status = $1)
ORDER BY created_at DESC
LIMIT NULLIF($2::int, 0) OFFSET $3::int;

-- name: ShortcodeExists :one
SELECT EXISTS(SELECT 1 FROM brands WHERE shortcode = $1);

-- name: CreateBrandContact :exec
INSERT INTO brand_contacts (id, brand_id, first_name, last_name, role, email,
  whatsapp_number, viewer_token, access_password_hash, token_active)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10);

-- name: GetBrandContactByID :one
SELECT * FROM brand_contacts WHERE id = $1;

-- name: GetBrandContactByViewerToken :one
SELECT * FROM brand_contacts WHERE viewer_token = $1 AND token_active = true;

-- name: ListBrandContacts :many
SELECT * FROM brand_contacts WHERE brand_id = $1 ORDER BY created_at;

-- name: DeactivateBrandContact :exec
UPDATE brand_contacts SET token_active = false, updated_at = NOW() WHERE id = $1;

-- name: RegenerateBrandContactPassword :exec
UPDATE brand_contacts SET access_password_hash = $2, updated_at = NOW() WHERE id = $1;

-- name: GetBrandContactForAuth :one
SELECT * FROM brand_contacts WHERE viewer_token = $1 AND token_active = true;
