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

	"github.com/Ze-uus/talent-backend/cmd/config"
	mw "github.com/Ze-uus/talent-backend/internal/middleware"
	"github.com/Ze-uus/talent-backend/internal/src/health"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	cfg := config.Load()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if version != "dev" {
		cfg.App_version = version
	}

	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(mw.CORS(cfg.Allowed_origins))
	r.Use(mw.RateLimit(cfg.Rate_limit_rps))

	// Versioned API — all routes live under /v1
	r.Route("/v1", func(r chi.Router) {
		api := humachi.New(r, huma.DefaultConfig("Scaloo API", cfg.App_version))

		// Public: health
		health.Mount(api, cfg.App_version, nil)

		// Future: auth, tracking, SSE, protected routes
	})

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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("stopped")
}
