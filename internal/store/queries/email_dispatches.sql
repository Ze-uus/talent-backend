-- name: InsertEmailDispatch :execrows
INSERT INTO email_dispatches (entity_type, entity_id, template_key, recipient)
VALUES ($1, $2, $3, $4)
ON CONFLICT (entity_type, entity_id, template_key, recipient) DO NOTHING;

-- name: EmailDispatchExists :one
SELECT EXISTS(
  SELECT 1 FROM email_dispatches
  WHERE entity_type = $1 AND entity_id = $2 AND template_key = $3 AND recipient = $4
);
