# Frontend updates — User status + append-only audit

Backend shipped shared `users.status`, richer login routing fields, user lifecycle admin APIs, and an append-only audit layer (REST + `/ws/admin/live`).

This guide maps those changes onto the Scaloo frontend stack:

```
#/types  →  ENDPOINTS  →  api-client  →  backend.service  →  ApiSurface / integrations/api  →  routes/hooks/UI
```

Wire casing matches Go (`ID`, `Full_name`, `Status`, …) unless the façade already remaps.

---

## Backend contract (what changed)

### User model

| Field | Notes |
|-------|--------|
| `Status` | `active \| invited \| suspended \| banned \| deleted \| pending \| rejected` — **source of truth** for lifecycle |
| `Active` | `true` only when account may authenticate; kept in sync with Status |
| `Deleted_at` | Present only when soft-deleted (omitted otherwise; never `0001-01-01`) |

### Login success (`POST /auth/login`, Google callback)

```ts
{
  token: string
  user_id: string
  role: UserRole
  status: UserStatus   // single field for all roles — not talent_status
  active: boolean
  require_totp_setup: boolean
  totp_recheck_due?: boolean  // login only
}
```

### Login / session error messages (403 unless credentials)

| Message | Meaning |
|---------|---------|
| `account_pending_approval` | Pending approval |
| `account_invited` | Staff invite not yet accepted |
| `account_suspended` | Suspended |
| `account_banned` | Banned |
| `account_deleted` | Soft-deleted |
| `account_rejected` | Rejected |
| `invalid_credentials` | Bad password |
| `invalid_active_transition` | PATCH `active` not allowed for current status (use suspend/ban/reinstate/delete) |

### Admin user lifecycle

| Method | Path |
|--------|------|
| GET | `/admin/users?role=&active=&status=` |
| GET | `/admin/users/{id}` |
| PATCH | `/admin/users/{id}` |
| POST | `/admin/users/{id}/suspend` |
| POST | `/admin/users/{id}/ban` |
| POST | `/admin/users/{id}/reinstate` |
| DELETE | `/admin/users/{id}` → soft-delete (`status=deleted`) |

### Audit

| Method | Path |
|--------|------|
| GET | `/admin/audit?entity_type&entity_id&actor_id&action_type&after_seq&limit` |
| GET | `/admin/audit/{id}` |
| GET | `/admin/audit/{id}/verify` |
| WS | `GET /ws/admin/live` — `type: "audit"` (admin/superadmin only) |

---

## File / folder update map

### 1. `src/types` (`#/types`)

**Create / update**

| File (typical) | Instructions |
|----------------|--------------|
| `enums` / `common` | Add `UserStatus = 'active' \| 'invited' \| 'suspended' \| 'banned' \| 'deleted' \| 'pending' \| 'rejected'`. Export alongside roles. |
| `models/user` (or `User`) | Add `Status: UserStatus`, `Deleted_at?: string` (ISO). Keep `Active: boolean`. Do **not** add `talent_status`. |
| `auth` DTOs (`LoginData`) | Extend with `user_id`, `role`, `status`, `active` (plus existing totp flags). Align Google login type the same way. |
| `models/audit` (**new**) | `AuditLog` with: `ID`, `Actor_id`, `Action_type`, `Entity_type`, `Entity_id`, `Before_state`, `After_state`, `Request_id`, `Seq`, `Prev_hash`, `Entry_hash`, `Signature`, `Archive_uri`, `IP_address`, `User_agent`, `Created_at`. |
| `audit` DTOs (**new**) | `AuditListParams`, `AuditVerifyResult` (`valid`, `broken_chain`, `bad_signature`, `expected_hash?`, `expected_sig?`). |
| Barrel `#/types` | Re-export new types. |

**UI routing rule:** after login, branch on `status` + `role`. Map `invited` → “accept invite” / blocked (login should not succeed). Treat missing `Deleted_at` as not deleted.

---

### 2. `src/lib/services/ENDPOINTS.ts`

**Update `ADMIN` (and WS helpers if separate)**

```ts
// Users
USERS: {
  LIST: '/admin/users',                    // + ?status=
  GET: (id: string) => `/admin/users/${id}`,
  PATCH: (id: string) => `/admin/users/${id}`,
  SUSPEND: (id: string) => `/admin/users/${id}/suspend`,
  BAN: (id: string) => `/admin/users/${id}/ban`,
  REINSTATE: (id: string) => `/admin/users/${id}/reinstate`,
  DELETE: (id: string) => `/admin/users/${id}`,    // soft-delete
},

// Audit
AUDIT: {
  LIST: '/admin/audit',
  GET: (id: string) => `/admin/audit/${id}`,
  VERIFY: (id: string) => `/admin/audit/${id}/verify`,
},
```

**WS** (wherever WS URLs live — often next to SSE):

```ts
WS: {
  ADMIN_LIVE: '/ws/admin/live',
  ADMIN_CYCLE: (cycleId: string) => `/ws/admin/${cycleId}`,
  TALENT_CYCLE: (cycleId: string) => `/ws/talent/${cycleId}`,
}
```

Path strings only — no fetch.

---

### 3. HTTP plumbing (`api-init` / `api-client`)

| File | Instructions |
|------|--------------|
| `api-init` | Map new 403 `message` codes to user-facing copy / dedicated `ApiError.code`. On `account_suspended\|banned\|deleted\|rejected`, clear token and redirect to a blocked screen (not generic login failure). |
| `api-client` | No structural change unless you add typed query helpers for audit filters (`after_seq`, `limit`). |
| 401 handler | Unchanged for expired sessions; also handle mid-session 403 when status flips (middleware rejects blocked statuses). |

---

### 4. `src/lib/services/backend.service.ts`

**`authService`**

- Ensure login/Google return typed `LoginData` including `user_id`, `role`, `status`, `active`.

**`usersService` (or `adminUsersService`)**

```ts
list(params?: { role?: string; active?: boolean; status?: UserStatus })
get(id: string)
patch(id: string, body: { full_name?: string; active?: boolean })
suspend(id: string)      // POST
ban(id: string)          // POST
reinstate(id: string)    // POST
remove(id: string)       // DELETE soft-delete — rename from deactivate if needed
```

**`auditService` (new)**

```ts
list(params: AuditListParams)
get(id: string)
verify(id: string)
```

Do not put WS connect here unless you already colocate WS helpers with backend.service — prefer a small `ws.adminLive()` helper next to SSE.

---

### 5. `integrations/api` + `api-types` (`ApiSurface`)

**Extend `ApiSurface`**

```ts
platform: {
  users: {
    list, get, invite, /* existing */
    suspend(id: string): Promise<void>
    ban(id: string): Promise<void>
    reinstate(id: string): Promise<void>
    softDelete(id: string): Promise<void>   // was deactivate
  }
  audit: {
    list(params: AuditListParams): Promise<AuditLog[]>
    get(id: string): Promise<AuditLog | undefined>   // soften 404
    verify(id: string): Promise<AuditVerifyResult>
  }
}

auth: {
  login(...): Promise<LoginData>  // now includes status/role/user_id
  // persist role+status in session store after login
}
```

**Façade instructions**

- `api.auth.login` → store token **and** `user_id` / `role` / `status` for routing.
- Soften audit `get` 404 → `undefined`.
- Map UI labels: Suspend / Ban / Reinstate / Delete → backend methods.
- Stub removal: if platform settings previously faked “ban”, delete stub and call real API.
- Orchestration: after suspend/ban/delete, invalidate local user cache / refetch list.

---

### 6. Auth hook / routing (`useAuth`, route guards)

| Concern | Instructions |
|---------|--------------|
| Post-login redirect | Switch on `LoginData.status` then `role`: `invited` → accept-invite / blocked; `pending` → pending page; `rejected` → rejected; `suspended\|banned\|deleted` → blocked; `active` → role home. |
| Session restore | If you only stored token, either call `GET /settings/profile` and read `Status`, or persist `status`/`role` at login. Prefer profile refresh on app boot. |
| Guard | Block app shell when `status !== 'active'` (except dedicated pending/blocked routes). |

---

### 7. UI surfaces

| Area | File(s) (typical) | Instructions |
|------|-------------------|--------------|
| Users table | `platform/users` / `users.tsx` | Show `Status` badge (not only Active). Filter by `status` query. Actions: Suspend, Ban, Reinstate, Delete. Hide actions by role (superadmin/admin). |
| User detail | user drawer/page | Display `Status`, `Deleted_at` when present. |
| Login | login route | On ApiError message, route to the correct screen (pending vs banned vs invalid password). |
| Audit log (**new**) | `platform/audit` | Table from `api.platform.audit.list`; row → detail; “Verify” calls `verify`. Optional live feed. |
| Admin live WS | ws helper + audit page | Connect `ADMIN_LIVE` when audit page mounts; on `type === 'audit'`, prepend payload to list (refetch periodically — drops allowed). |

---

### 8. WS helpers (if separate from ENDPOINTS)

| File | Instructions |
|------|--------------|
| `lib/ws` or `integrations/realtime` | Add `connectAdminLive(token, onMessage)`. Handle `audit` in the same switch as conversion/cycle/talent. Document: admin/superadmin only; campaign_manager is **not** allowed. |

---

## Suggested implementation order

1. `#/types` — `UserStatus`, `User` fields, `LoginData`, `AuditLog`
2. `ENDPOINTS` — user lifecycle + audit + `WS.ADMIN_LIVE`
3. `backend.service` — methods
4. `ApiSurface` + `integrations/api` — expose + login store
5. `useAuth` / guards — status-based routing
6. Users UI — badges + actions
7. Audit UI + live WS
8. Error copy for new account_* codes in `api-init`

---

## Do / don’t

| Do | Don’t |
|----|--------|
| Use one `status` for all roles | Add `talent_status` on login |
| Treat DELETE user as soft-delete | Assume hard delete / missing user |
| Soften audit 404 in façade | Let raw 404 bubble into audit UI |
| Refetch REST after WS audit hints | Rely on WS as source of truth |
| Clear session on blocked status mid-flight | Keep showing admin chrome when 403 account_* |

---

## Quick checklist

- [ ] `User.Status` / `Deleted_at` in types
- [ ] `LoginData` includes `user_id`, `role`, `status`, `active`
- [ ] ENDPOINTS for suspend/ban/reinstate/delete + audit + admin live WS
- [ ] `usersService` + `auditService` in backend.service
- [ ] ApiSurface methods + façade wiring
- [ ] Login/guard routing by `status`
- [ ] Users table actions + status filter
- [ ] Audit list/detail/verify page
- [ ] `/ws/admin/live` consumer for `audit` events
- [ ] ApiError handling for `account_banned` / `deleted` / `rejected`
