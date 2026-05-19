-- name: LogConversionEvent :exec
INSERT INTO conversion_events (id, link_token, talent_id, campaign_id, cycle_id,
  pipeline_type, event_type, kpb_type, valid_lead, idempotency_key, occurred_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11);

-- name: GetDailyConversionCount :one
SELECT COUNT(*)::float8 FROM conversion_events
WHERE talent_id = $1 AND cycle_id = $2
  AND DATE(occurred_at AT TIME ZONE 'UTC') = DATE($3 AT TIME ZONE 'UTC');

-- name: GetTotalConversions :one
SELECT COUNT(*)::float8 FROM conversion_events WHERE cycle_id = $1;

-- name: GetTalentConversions :one
SELECT COUNT(*)::float8 FROM conversion_events
WHERE talent_id = $1 AND cycle_id = $2;

-- name: FlagFallbackConversions :exec
UPDATE conversion_events
SET fallback_flagged = true
WHERE cycle_id = $1
  AND DATE(occurred_at AT TIME ZONE 'UTC') = DATE($2 AT TIME ZONE 'UTC')
  AND fallback_flagged = false;

-- name: LockFallbackConversions :exec
UPDATE conversion_events
SET fallback_flagged = true
WHERE cycle_id = $1 AND fallback_flagged = true;
