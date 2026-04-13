# Scaloo Backend

Talent–budget allocation platform. Matches talents to campaign budget
slots using the Hungarian Algorithm, tracks conversion performance in real
time across three role-scoped views, and calculates performance-based
payouts at cycle close.

---

## How it works

Campaigns run in **cycles** (5, 7, or 10 days). Each cycle has its own
budget, talent pool, and slot structure. The allocation algorithm runs
fresh per cycle — talents from previous cycles compete alongside new
applicants each time.

**Execution flow:**

1. Admin creates a campaign (type: Direct Traffic or Lead Validation)
2. Admin creates Cycle 1 with a cycle budget → Waterfall Algorithm generates tier slots
3. Admin runs the Assignment Solver → Hungarian Algorithm returns optimal talent→slot matching
4. Admin reviews and confirms assignments (manual overrides are audit-logged)
5. Assigned talents receive a unique tracking link and campaign brief
6. Talent shares the link — every click/conversion auto-logs via `POST /track/:token`
7. The Daily Compute Job (00:00 UTC) classifies each talent's utilization pattern and updates PDC_next
8. The Nightly Learning Job (01:00 UTC) updates each talent's long-term Bayesian baseline
9. At cycle close, payouts are calculated, scored against KPIs, and queued for admin approval
10. Remaining campaign budget is carried forward to Cycle 2

**Three real-time views:**
- **Admin** — birds-eye cycle dashboard over WebSocket (`/ws/admin/{cycle_id}`)
- **Talent** — own conversion metrics over WebSocket (`/ws/talent/{cycle_id}`)
- **Campaign client** — read-only trend feed over SSE (`/stream/{viewer_token}`)

**Auth model (Option D):**
- Admin: invited via email → verifies invite → sets password → sets up TOTP → TOTP required every login
- Talent: registers → awaits approval → sets up TOTP during onboarding → TOTP re-checked every 72h
- Campaign viewer: no account — token URL + password → short-lived cookie → SSE access

---

## Stack

| Layer | Technology |
|---|---|
| Language | Go 1.22+ |
| Router | `go-chi/chi v5` |
| API docs | `danielgtaylor/huma v2` (OpenAPI 3.1, auto-generated) |
| WebSocket | `coder/websocket` |
| Database | PostgreSQL (NeonDB) via `jackc/pgx v5` |
| Query gen | `sqlc` |
| Migrations | `pressly/goose v3` |
| Scheduler | `robfig/cron v3` |
| 2FA | `pquerna/otp` (TOTP, RFC 6238) |
| OAuth | `golang.org/x/oauth2` (Google) |
| Containers | Docker + Docker Compose |

---

## Getting started

### Prerequisites

```bash
make install-tools   # installs golangci-lint, goose, air, sqlc
```

Requires: Go 1.22+, Docker, Docker Compose.

### First-time setup

```bash
git clone https://github.com/your-org/talent-backend.git
cd talent-backend
cp .env.local.example .env.local
# Fill in DATABASE_URL and APP_KEY

make setup           # installs tools, lints, migrates, tests
make dev             # starts docker + server
```

API: `http://localhost:8080`
Docs: `http://localhost:8080/docs`
Spec: `http://localhost:8080/openapi.json`

### Daily development

```bash
make dev             # docker-up + migrate-up + run
make test            # run all tests
make lint            # lint
make all             # full cycle: lint → test → migrate → build → docker
```

---

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | Postgres connection string |
| `APP_KEY` | Yes | — | Application secret (min 32 chars) |
| `PORT` | No | `8080` | HTTP listen port |
| `APP_ENV` | No | `development` | `development` \| `production` |
| `ALLOWED_ORIGINS` | No | `http://localhost:3000` | CORS origins (comma-separated) |
| `RATE_LIMIT_RPS` | No | `100` | Requests/sec per IP |
| `DELTA_LT` | No | `0.97` | Long-term Bayesian decay rate |
| `GOOGLE_CLIENT_ID` | OAuth | — | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | OAuth | — | Google OAuth client secret |
| `GOOGLE_REDIRECT_URL` | OAuth | — | OAuth callback URL |

---

## Project structure

```
scaloo/
├── cmd/
│   └── api/
│       ├── main.go       ← entry point, wires all layers
│       └── config.go     ← env-based config
├── docs/                 ← generated OpenAPI spec
├── internal/
│   ├── algo/             ← pure math, zero dependencies
│   │   ├── waterfall.go  ← cycle slot generation
│   │   ├── qualifier.go  ← tier eligibility + mobility rules
│   │   ├── hungarian.go  ← cost matrix + solver
│   │   ├── stec.go       ← daily utilization + pattern classification
│   │   ├── ltl-bayesian.go ← long-term Bayesian baseline
│   │   └── payout.go     ← cycle payout calculation
│   ├── jobs/
│   │   ├── daily_compute.go    ← 00:00 UTC cron
│   │   └── nightly_learning.go ← 01:00 UTC cron
│   ├── middleware/
│   │   ├── auth.go       ← session + role enforcement
│   │   ├── cors.go
│   │   └── rate-limit.go
│   ├── src/              ← all route packages (same structure each)
│   │   ├── auth/         ← service.go, handler.go, router.go
│   │   ├── assignment/   ← solver, confirm, expand
│   │   ├── campaign/     ← campaign + cycle CRUD
│   │   └── talent/       ← talent CRUD, approval, self-service
│   ├── store/
│   │   ├── store.go      ← Store interface + domain types
│   │   └── postgres/     ← pgx implementation
│   └── websocket/
│       ├── ws.go         ← hub: admin + talent WS
│       └── sse.go        ← campaign viewer SSE
└── migrations/           ← goose SQL files (001–008)
```

---

## Auth flows

### Admin
```
POST /admin/talents/invite     → invite email sent
POST /auth/admin/verify-invite → set password
POST /auth/totp/enroll         → get QR URI
POST /auth/totp/verify         → activate 2FA
POST /auth/login               → email + password + TOTP code → session token
```

### Talent
```
POST /auth/register            → create account (pending approval)
[admin approves]
POST /auth/login               → email + password [+ TOTP after grace period]
POST /auth/totp/enroll         → prompted at onboarding
POST /auth/totp/verify         → activate 2FA
# TOTP re-checked every 72h
```

### Campaign viewer
```
GET  /view/{token}             → enter password (no account needed)
POST /view/{token}/access      → validates password → sets viewer cookie
GET  /stream/{token}           → SSE feed (cookie required)
```

---

## Contributing

### Branch model

```
feature/*  →  dev   (PR required, CI must pass)
dev        →  main  (PR required, CI must pass, source must be dev)
main       →  protected, no direct push, deploy triggers here
```

**No feature branch PRs directly to `main`.**

### PR checklist

- [ ] `make ci` passes locally (`make lint` + `make test-coverage` + `make build`)
- [ ] Tests written for new service logic
- [ ] New endpoints registered in Huma (visible in `/docs`)
- [ ] PATCH inputs use pointer fields only
- [ ] No business logic in handlers (service layer only)
- [ ] Migration added if schema changed
- [ ] Audit log written for admin override actions
- [ ] `.env.local.example` updated if new env vars added

### Opening a PR

```bash
git checkout -b feature/your-feature dev
# implement + test
make ci
git push origin feature/your-feature
# open PR against dev
```

Squash-merge preferred to keep `dev` history clean.

---

## Production

```bash
make build                              # binary → bin/scaloo
APP_ENV=production ./bin/scaloo         # with env vars set

# Or Docker
make docker-up-build
```