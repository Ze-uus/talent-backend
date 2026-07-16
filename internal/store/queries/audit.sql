-- name: WriteAuditLog :exec
INSERT INTO audit_log (id, actor_id, action_type, entity_type, entity_id,
  before_state, after_state)
VALUES ($1,$2,$3,$4,$5,$6,$7);

-- name: ListAuditLog :many
SELECT * FROM audit_log
WHERE entity_type = $1 AND entity_id = $2
ORDER BY created_at DESC;
