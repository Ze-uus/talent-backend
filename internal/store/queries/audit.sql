-- name: WriteAuditLog :exec
INSERT INTO audit_log (id, actor_id, action_type, entity_type, entity_id,
  before_state, after_state, request_id, seq, prev_hash, entry_hash, signature,
  archive_uri, ip_address, user_agent)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15);

-- name: ListAuditLog :many
SELECT * FROM audit_log
WHERE entity_type = $1 AND entity_id = $2
ORDER BY created_at DESC;

-- name: GetAuditLogByID :one
SELECT * FROM audit_log WHERE id = $1;

-- name: GetAuditChainTipForUpdate :one
SELECT last_hash, last_seq FROM audit_chain_tip WHERE id = 1 FOR UPDATE;

-- name: UpdateAuditChainTip :exec
UPDATE audit_chain_tip
SET last_hash = $1, last_seq = $2, updated_at = NOW()
WHERE id = 1;

-- name: GetAuditChainTip :one
SELECT last_hash, last_seq FROM audit_chain_tip WHERE id = 1;
