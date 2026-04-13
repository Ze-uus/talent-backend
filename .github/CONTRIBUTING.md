# Contributing to Scaloo Backend

---

## Getting started

```bash
git clone https://github.com/your-org/talent-backend.git
cd talent-backend
cp .env.local.example .env.local
make setup    # installs tools, lints, migrates, runs tests
make dev      # starts docker + server
```

Docs at `http://localhost:8080/docs`.

---

## Branch model

```
feature/* or feat/*   →   dev    PR required, CI must pass
bugfix/*  or fix/*   →   dev    PR required, CI must pass
dev         →   main   PR required, CI must pass, source must be dev
main                   protected — no direct push, deploy triggers here
```

**Never open a PR from a feature branch directly to `main`.**
All changes land in `dev` first.

### Naming your branch

```
feature/talent-approval-flow
bugfix/waterfall-remainder-calculation
chore/update-go-dependencies
docs/add-cycle-endpoint-examples
```

---

## Workflow

```bash
# 1. Branch from dev
git checkout dev
git pull origin dev
git checkout -b feature/your-feature

# 2. Implement
# Follow the implementation order: algo → store → service → handler → router

# 3. Test locally
make ci       # lint + test-coverage + build — must pass before pushing

# 4. Push + open PR
git push origin feature/your-feature
# Open PR against dev on GitHub
```

---

## Code standards

### Layer rules

| Layer | Allowed imports | Not allowed |
|---|---|---|
| `internal/algo/` | `math`, `sort`, stdlib only | DB, HTTP, framework |
| `internal/store/` | `pgx`, stdlib | HTTP, framework |
| `internal/jobs/` | `algo`, `store`, stdlib | HTTP handlers |
| `internal/src/*/service.go` | `algo`, `store` | HTTP, `chi`, `huma` |
| `internal/src/*/handler.go` | service package, `huma` | `store` directly |
| `internal/middleware/` | `store`, `src/auth` | service packages |

No handler may call the store directly — always through a service.
No service may import another service — only store + algo.

### Naming

All struct fields, JSON keys, DB columns, constants, and error variables
use `snake_case`. No camelCase. No `CamelCase` for constants — use `snake_case` string constants.

```go
// correct
const status_active = "active"
type UserPatch struct { Full_name *string }

// wrong
const StatusActive = "active"
type UserPatch struct { FullName *string }
```

### Response envelope

Every handler returns this shape — no exceptions:

```go
type Response struct {
    Data    any    `json:"data"`
    Message string `json:"message"`
    Status  string `json:"status"`
    Success bool   `json:"success"`
}
```

Error `message` values are machine-readable `snake_case` strings
(e.g. `"invalid_totp_code"`, `"cycle_budget_exceeds_remaining"`).
Never return user-facing prose in the message field — that's the
frontend's job.

### PATCH pattern

PATCH input structs use pointer fields. Only non-nil fields are applied.

```go
type TalentPatch struct {
    Status        *string  `json:"status,omitempty"`
    Rate_per_day  *float64 `json:"rate_per_day,omitempty"`
    Bio           *string  `json:"bio,omitempty"`
}
```

Never accept a full object for a PATCH endpoint.

### Audit logging

Any admin action that modifies an assignment, changes talent status,
or overrides solver output must write to `audit_log` with `before_state`
and `after_state` as JSONB.

```go
_ = s.st.WriteAuditLog(ctx, store.AuditLog{
    Actor_id:    actor_id,
    Action_type: "assignment_override",
    Entity_type: "cycle",
    Entity_id:   cycle_id,
    Before_state: marshalJSON(before),
    After_state:  marshalJSON(after),
})
```

### Error variables

Define errors as lowercase `snake_case` package-level variables:

```go
var (
    err_invalid_credentials = errors.New("invalid_credentials")
    err_totp_required       = errors.New("totp_required")
)
```

---

## Writing tests

### Unit tests (required for all service logic)

Use a mock store. Never hit the real database in unit tests.

```go
// internal/src/campaign/service_test.go
func TestCreateCycle_RunsWaterfall(t *testing.T) {
    ms := &mockStore{...}
    svc := campaign.New(ms)
    cycle, err := svc.CreateCycle(context.Background(), in)
    // assert slots created
}
```

Tag integration tests:

```go
//go:build integration

func TestCreateCycle_Integration(t *testing.T) {
    // uses real DB
}
```

Run unit tests: `make test`
Run integration tests: `make test-integration`
Run algo tests only: `make test-algo`

### Test file naming

`service_test.go` → service unit tests
`handler_test.go` → handler unit tests (httptest, no real DB)
Each file in `internal/algo/` has a corresponding `_test.go`.

---

## Migrations

```bash
# Create
make migrate-create   # prompts for name → creates migrations/NNN_name.sql

# Apply
make migrate-up

# Check
make migrate-status
```

Migration rules:
- Always include `-- +goose Up` and `-- +goose Down`
- `Down` must be a clean reverse of `Up`
- Never modify an applied migration — create a new one
- Migrations run in numeric order — never reorder existing files

---

## Adding a new endpoint

1. Define input/output structs in `handler.go`
2. Implement business logic in `service.go`
3. Write service unit tests in `service_test.go`
4. Register with Huma in `handler.go` via `huma.Register(...)`
5. Mount in `router.go` — verify it appears in `router.go`'s route group
6. Confirm it shows in `/docs` when server runs
7. Write handler test in `handler_test.go` using `httptest`

---

## Commit style

```
feat: add talent approval endpoint
fix: waterfall remainder calculation overflow
chore: update pgx to v5.7
docs: add cycle endpoint examples
test: add STEC collapse pattern test
migration: add talent_reports table
```

Use present tense. Keep the subject under 72 chars.
No emoji in commit messages.

---

## Questions?

Open a discussion on GitHub or drop a message in the team channel.
If something in this guide contradicts the PRD, the PRD wins.