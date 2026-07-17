package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"

	"github.com/Ze-uus/talent-backend/cmd/config"
	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/jobs"
	mw "github.com/Ze-uus/talent-backend/internal/middleware"
	postgres "github.com/Ze-uus/talent-backend/internal/store/postgres"
	adminsvc "github.com/Ze-uus/talent-backend/internal/src/admin"
	assignmentsvc "github.com/Ze-uus/talent-backend/internal/src/assignment"
	authsvc "github.com/Ze-uus/talent-backend/internal/src/auth"
	brandsvc "github.com/Ze-uus/talent-backend/internal/src/brand"
	campaignsvc "github.com/Ze-uus/talent-backend/internal/src/campaign"
	"github.com/Ze-uus/talent-backend/internal/src/health"
	payoutsvc "github.com/Ze-uus/talent-backend/internal/src/payout"
	settingssvc "github.com/Ze-uus/talent-backend/internal/src/settings"
	talentsvc "github.com/Ze-uus/talent-backend/internal/src/talent"
	trackingsvc "github.com/Ze-uus/talent-backend/internal/src/tracking"
	ws "github.com/Ze-uus/talent-backend/internal/websocket"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	_ = godotenv.Load(".env.local")
	cfg := config.Load()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if version != "dev" {
		cfg.App_version = version
	}

	db, err := postgres.NewStore(cfg.Database_url)
	if err != nil {
		log.Error("db connect failed", "err", err)
		os.Exit(1)
	}

	// ─── Real-time channels ──────────────────────────────────────────────────────

	anomaly_ch := make(chan domain.AnomalyEvent, 256)
	conversion_ch := make(chan domain.ConversionEvent, 512)
	cycle_update_ch := make(chan domain.CycleUpdateEvent, 64)
	talent_update_ch := make(chan domain.TalentUpdateEvent, 64)

	hub := ws.NewHub(anomaly_ch, conversion_ch, cycle_update_ch, talent_update_ch, db, cfg.Allowed_origins, log)
	sse := ws.NewSSEHandler(db, 5*time.Second)

	authSvc := authsvc.NewAuthService(db, "Scaloo")

	// ─── Services (channels wired after hub creation) ───────────────────────────

	payout := payoutsvc.New(db)
	campaign := campaignsvc.New(db, payout, cycle_update_ch, log)
	brand := brandsvc.New(db)
	talent := talentsvc.New(db)
	assignment := assignmentsvc.New(db, talent_update_ch, log)
	admin := adminsvc.New(db, talent_update_ch, log)
	settings := settingssvc.New(db, authSvc)
	tracking := trackingsvc.New(db, conversion_ch, log)

	// ─── Router + global middleware ──────────────────────────────────────────────

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(mw.CORS(cfg.Allowed_origins))
	r.Use(mw.RateLimit(cfg.Rate_limit_rps))
	// Resolve Bearer sessions for Huma handlers that call UserFromContext.
	// Public routes (login/register/etc.) work without a token; invalid tokens still 401.
	r.Use(mw.AuthenticateOptional(authSvc))

	// ─── Huma API (single instance, single /docs) ────────────────────────────────

	apiCfg := huma.DefaultConfig("Scaloo API", cfg.App_version)
	apiCfg.DocsPath = ""
	if apiCfg.Components == nil {
		apiCfg.Components = &huma.Components{}
	}
	apiCfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearerAuth": {
			Type:        "http",
			Scheme:      "bearer",
			Description: "Session token from POST /auth/login — paste without the 'Bearer ' prefix",
		},
	}
	apiCfg.Security = []map[string][]string{{"bearerAuth": {}}}
	apiCfg.Info.Description = `
## Overview

**Scaloo** is a performance-based talent-budget allocation platform. Brands run
time-boxed campaigns; the algorithm assigns talent to budget slots, tracks
conversions in real time, and calculates payouts at cycle close.

---

## Core concepts

| Entity | Description |
|--------|-------------|
| **Brand** | Client company. Each brand has a 3-letter shortcode (e.g. ` + "`CRD`" + `) used to generate campaign IDs. |
| **Campaign** | Top-level engagement. Human ID format: ` + "`CRD-26-01`" + `. Holds the total budget and CPA targets. |
| **Cycle** | Atomic execution unit within a campaign (5, 7, or 10 days). Budget is drawn down per cycle. |
| **Talent** | Registered performer assigned to budget slots. Categorised as ` + "`student`" + `, ` + "`micro`" + `, or ` + "`community`" + `. |
| **Assignment** | A talent's allocation to a specific cycle slot, with match scores and PDC values. |
| **Payout record** | Immutable audit record created at cycle close — gross, KPB bonuses, commission, final net. |
| **Tracking link** | Short-lived token URL. Every click/conversion is logged against it. |

---

## Authentication

All protected endpoints require a **Bearer token** in the ` + "`Authorization`" + ` header:

` + "```" + `
Authorization: Bearer <token>
` + "```" + `

### Getting a token

1. **Register** ` + "`POST /auth/register`" + ` — creates a talent account (email + password).
2. **Login** ` + "`POST /auth/login`" + ` — returns a session token.
3. **Google OAuth** ` + "`GET /auth/google`" + ` — redirect flow for talent sign-in.
4. **Invite flow** ` + "`POST /auth/invite/verify`" + ` — for admin / manager accounts created by superadmin invite.

### Roles

| Role | Access |
|------|--------|
| ` + "`superadmin`" + ` | Full access including role management and invites |
| ` + "`admin`" + ` | Full access except role changes |
| ` + "`campaign_manager`" + ` | Assigned campaigns only |
| ` + "`talent`" + ` | Own profile, reports, and payout records |
| ` + "`viewer`" + ` | Read-only campaign dashboard (token + password) |

---

## Request flow (happy path)

` + "```" + `
1. Brand created          POST /brands
2. Campaign created       POST /campaigns
3. Cycle opened           POST /campaigns/:id/cycles
4. Talents assigned       POST /assignments
5. Tracking links issued  POST /assignments/:id/links
6. Conversions logged     GET  /t/:token  (public tracking endpoint)
7. Cycle closed           POST /cycles/:id/close
8. Payout calculated      GET  /payouts/cycle/:id
` + "```" + `

---

## Rate limiting

` + "`" + `${cfg.Rate_limit_rps}` + "`" + ` requests/second per IP. Exceeding this returns ` + "`429 Too Many Requests`" + `.

---

## Response envelope

All endpoints return the same envelope:

` + "```json" + `
{
  "status": "ok" | "error",
  "message": "human readable string",
  "data": { ... } | null
}
` + "```"
	api := humachi.New(r, apiCfg)

	// ─── Docs UI ─────────────────────────────────────────────────────────────────

	r.Get("/docs", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no" />
  <title>Scaloo API</title>
  <script src="https://unpkg.com/@stoplight/elements/web-components.min.js"></script>
  <link rel="stylesheet" href="https://unpkg.com/@stoplight/elements/styles.min.css" />
</head>
<body style="margin:0">
  <elements-api
    apiDescriptionUrl="/openapi.json"
    router="hash"
    layout="sidebar"
    tryItCredentialsPolicy="include"
  />
</body>
</html>`))
	})

	// ─── Swagger UI (alternative to /docs) ───────────────────────────────────────

	r.Get("/swagger", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
  <title>Scaloo API — Swagger UI</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
<script>
window.onload = function() {
  SwaggerUIBundle({
    url: "/openapi.json",
    dom_id: "#swagger-ui",
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
    layout: "StandaloneLayout",
    deepLinking: true,
    tryItOutEnabled: true,
  })
}
</script>
</body>
</html>`))
	})

	// ─── Routes ──────────────────────────────────────────────────────────────────
	// AuthenticateOptional (above) injects the user when Authorization: Bearer is set.
	// Handlers that call UserFromContext still return 401 if no session is present.

	health.Mount(api, cfg.App_version, db)
	authsvc.Mount(api, authSvc)
	trackingsvc.Mount(api, tracking)

	// Generic campaign-viewer SSE stream (token validated inside handler)
	r.Get("/stream/{viewer_token}", sse.ServeHTTP)

	// Brand contact SSE stream (cookie auth)
	r.Group(func(r chi.Router) {
		r.Use(mw.AuthenticateBrandContact(authSvc))
		r.Get("/brand/view/{viewer_token}/live", sse.ServeBrandContact)
	})

	brandsvc.Mount(api, brand)
	campaignsvc.Mount(api, campaign)
	assignmentsvc.Mount(api, assignment)
	payoutsvc.Mount(api, payout)
	adminsvc.Mount(api, admin)
	talentsvc.Mount(api, talent)
	settingssvc.Mount(api, settings)

	// WS upgrades require a valid Bearer session at the chi level.
	r.Group(func(r chi.Router) {
		r.Use(mw.Authenticate(authSvc))
		r.Get("/ws/admin/{cycle_id}", hub.ServeAdmin)
		r.Get("/ws/talent/{cycle_id}", hub.ServeTalent)
	})

	// ─── Background jobs ─────────────────────────────────────────────────────────

	c := cron.New()
	if _, err := c.AddJob("0 0 * * *", jobs.NewDailyComputeJob(db, anomaly_ch, log)); err != nil {
		log.Error("failed to register DailyComputeJob", "err", err)
	}
	if _, err := c.AddJob("0 1 * * *", jobs.NewNightlyLearningJob(db, cfg.Delta_lt, log)); err != nil {
		log.Error("failed to register NightlyLearningJob", "err", err)
	}
	if _, err := c.AddJob("0 * * * *", jobs.NewFallbackCheckJob(db, log)); err != nil {
		log.Error("failed to register FallbackCheckJob", "err", err)
	}
	c.Start()
	defer c.Stop()

	// ─── WS hub goroutine ────────────────────────────────────────────────────────

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// ─── HTTP server ─────────────────────────────────────────────────────────────

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info("server starting",
			"port", cfg.Port,
			"version", cfg.App_version,
			"env", cfg.App_env,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")
	cancel()

	sd_ctx, sd_cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer sd_cancel()
	_ = srv.Shutdown(sd_ctx)
	log.Info("stopped")
}
