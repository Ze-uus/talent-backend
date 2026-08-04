# WebSocket Events

Real-time updates for admin and talent dashboards. Campaign viewers use SSE instead — see `/stream/{viewer_token}`.

## Connecting

| Endpoint | Auth | Audience |
|----------|------|----------|
| `GET /ws/admin/live` | `Authorization: Bearer <session_token>` | Admin / superadmin (global ops + audit) |
| `GET /ws/admin/{cycle_id}` | `Authorization: Bearer <session_token>` | Admin / campaign manager |
| `GET /ws/talent/{cycle_id}` | `Authorization: Bearer <session_token>` | Talent (own data only) |

The server sends WebSocket ping frames; browsers respond automatically. On connect or reconnect, **refetch REST state first**, then apply WS deltas.

## Message envelope

Every message is a JSON text frame:

```json
{
  "type": "conversion",
  "cycle_id": "cycle-abc",
  "talent_id": "talent-xyz",
  "timestamp": "2026-07-02T12:00:00Z",
  "payload": { }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Event kind — see catalog below |
| `cycle_id` | string | Cycle this event belongs to |
| `talent_id` | string | Present when event is talent-scoped |
| `timestamp` | string (RFC3339) | Server broadcast time |
| `payload` | object | Shard — only the fields needed to patch UI state |

## Shard-first pattern

Treat WebSocket as a **realtime hint + patch**, not the source of truth.

1. On WS open → refetch cycle/talent stats via REST.
2. On each message → patch local store from `payload` (especially `payload.shard`).
3. Periodically reconcile via REST (channels can drop events under load).

```typescript
ws.onmessage = (raw) => {
  const msg = JSON.parse(raw.data);
  switch (msg.type) {
    case "conversion":
      store.patchConversion(msg.cycle_id, msg.talent_id, msg.payload);
      break;
    case "cycle_update":
      store.patchCycle(msg.cycle_id, msg.payload);
      break;
    case "talent_update":
      store.patchTalent(msg.talent_id, msg.payload);
      break;
    case "anomaly":
      store.flagAnomaly(msg.cycle_id, msg.payload);
      break;
    case "audit":
      store.appendAudit(msg.payload);
      break;
  }
};
```

---

## Event catalog

### `audit`

**Producer:** `audit.Recorder` after every successful append (HTTP mutations + domain actions)  
**Recipients:** Admin-live clients (`/ws/admin/live`) only  
**Channel buffer:** 256

**Payload:** full audit entry (same shape as `GET /admin/audit/{id}`):

```json
{
  "id": "…",
  "actor_id": "…",
  "action_type": "user_suspended",
  "entity_type": "user",
  "entity_id": "…",
  "request_id": "…",
  "seq": 42,
  "prev_hash": "…",
  "entry_hash": "…",
  "signature": "…",
  "archive_uri": "file://… or s3://…",
  "timestamp": "2026-07-25T12:00:00Z",
  "before": { },
  "after": { }
}
```

REST companions: `GET /admin/audit`, `GET /admin/audit/{id}`, `GET /admin/audit/{id}/verify`.

---

### `conversion`

**Producer:** `POST /track/{token}` after successful DB write  
**Recipients:** Admin (all talents in cycle) + matching talent client

**Payload:**

```json
{
  "update_type": "created",
  "talent_id": "talent-1",
  "cycle_id": "cycle-1",
  "campaign_id": "camp-1",
  "event_type": "click",
  "kpb_type": "",
  "pipeline_type": "direct_traffic",
  "occurred_at": "2026-07-02T12:00:00Z",
  "shard": {
    "cycle_total": 1201,
    "talent_total": 42
  }
}
```

| Field | UI action |
|-------|-----------|
| `shard.cycle_total` | Set cycle-wide counter directly |
| `shard.talent_total` | Set talent row counter directly |
| `event_type` | Append to activity feed; style by type |

Duplicate `idempotency_key` → HTTP 204, **no WS push**.

---

### `anomaly`

**Producer:** `daily_compute` job (00:00 UTC) on collapse/breakout detection  
**Recipients:** Admin only

**Payload:**

```json
{
  "talent_id": "talent-1",
  "cycle_id": "cycle-1",
  "timestamp": "2026-07-02T00:00:05Z",
  "shard": {
    "anomaly_type": "collapse",
    "value": 0.42
  }
}
```

| `shard.anomaly_type` | UI action |
|----------------------|-----------|
| `collapse` | Highlight talent row warning; show admin alert |
| `breakout` | Highlight talent row opportunity; show admin badge |

---

### `cycle_update`

**Producer:** Campaign service — cycle lifecycle changes  
**Recipients:** Admin only

**`update_type` values:** `created`, `activated`, `paused`, `closed`, `patched`, `payouts_finalised`

**Payload:**

```json
{
  "cycle_id": "cycle-1",
  "campaign_id": "camp-1",
  "update_type": "paused",
  "shard": {
    "status": "paused",
    "remaining_budget": 4500.00
  }
}
```

| `update_type` | UI action |
|---------------|-----------|
| `created` | Add cycle to list; show pending state |
| `activated` | Enable live tracking UI |
| `paused` | Show paused badge; disable edits |
| `closed` | Lock cycle UI; disable assignment actions |
| `patched` | Patch cycle header fields from shard |
| `payouts_finalised` | Show payout-pending state; refresh payout tab |

---

### `talent_update`

**Producer:** Admin service (approve/reject/suspend/reinstate/patch) + assignment service (`assigned`)  
**Recipients:** Admin (when `cycle_id` set) + matching talent client(s)

**`update_type` values:** `approved`, `rejected`, `suspended`, `reinstated`, `patched`, `assigned`

**Payload (assignment):**

```json
{
  "talent_id": "talent-1",
  "cycle_id": "cycle-1",
  "update_type": "assigned",
  "shard": {
    "status": "active",
    "cycle_id": "cycle-1",
    "effective_tier": 5000,
    "slot_id": "slot-abc",
    "tracking_token": "xyz..."
  }
}
```

**Payload (account-level, no cycle):**

```json
{
  "talent_id": "talent-1",
  "update_type": "approved",
  "shard": {
    "status": "active",
    "category": "student"
  }
}
```

| `update_type` | UI action |
|---------------|-----------|
| `approved` | Show welcome state; unlock onboarding |
| `rejected` | Show rejection notice |
| `suspended` | Force logout / block actions |
| `reinstated` | Restore access |
| `patched` | Patch profile fields from shard |
| `assigned` | Show tracking link prompt; refresh cycle detail |

When `cycle_id` is empty, the event fans out to **all** WS connections for that talent across cycles.

---

## Drop policy

Channels are buffered (conversion: 512, anomaly: 256, cycle/talent: 64). If a client send buffer is full, the server **drops** the message and logs a warning. Clients must tolerate missed events via periodic REST refresh.
