package health

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Ze-uus/talent-backend/internal/response"
)

// Pinger tests database connectivity. Nil means no DB is wired yet.
type Pinger interface {
	Ping(ctx context.Context) error
}

type handler struct {
	version    string
	started_at time.Time
	db         Pinger
}

func new_handler(version string, db Pinger) *handler {
	return &handler{
		version:    version,
		started_at: time.Now().UTC(),
	    db:         db,
	}
}

type health_output struct {
	Body response.Response
}

// register wires the four health endpoints onto the Huma API.
func (h *handler) register(api huma.API) {

	huma.Register(api, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health check with uptime and version",
		Tags:        []string{"health"},
	}, func(ctx context.Context, _ *struct{}) (*health_output, error) {
		data := map[string]any{
			"version":   h.version,
			"uptime":    time.Since(h.started_at).String(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		return &health_output{Body: response.Ok(data, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "health_live",
		Method:      http.MethodGet,
		Path:        "/health/live",
		Summary:     "Liveness probe",
		Tags:        []string{"health"},
	}, func(ctx context.Context, _ *struct{}) (*health_output, error) {
		return &health_output{Body: response.Ok(nil, "alive")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "health_ready",
		Method:      http.MethodGet,
		Path:        "/health/ready",
		Summary:     "Readiness probe — checks DB connectivity",
		Tags:        []string{"health"},
	}, func(ctx context.Context, _ *struct{}) (*health_output, error) {
		if h.db == nil {
			return &health_output{Body: response.Fail("db_not_configured")}, nil
		}
		if err := h.db.Ping(ctx); err != nil {
			return &health_output{Body: response.Fail("db_unreachable")}, nil
		}
		return &health_output{Body: response.Ok(nil, "ready")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "health_status",
		Method:      http.MethodGet,
		Path:        "/health/status",
		Summary:     "Service status with dependency health",
		Tags:        []string{"health"},
	}, func(ctx context.Context, _ *struct{}) (*health_output, error) {
		db_status := "unavailable"
		if h.db != nil {
			if err := h.db.Ping(ctx); err == nil {
				db_status = "connected"
			} else {
				db_status = "unreachable"
			}
		}

		data := map[string]any{
			"service":   "scaloo",
			"version":   h.version,
			"uptime":    time.Since(h.started_at).String(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"dependencies": map[string]string{
				"postgres": db_status,
			},
		}
		return &health_output{Body: response.Ok(data, "ok")}, nil
	})
}
