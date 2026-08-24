package middleware

import (
	"net/http"
	"strings"

	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/Ze-uus/talent-backend/internal/audit"
	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
)

// RequestMeta copies chi request ID plus client IP/UA into ctxkeys for audit.
func RequestMeta(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := ctxkeys.WithRequestMeta(r.Context(), chimw.GetReqID(r.Context()), r.RemoteAddr, r.UserAgent())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuditMutations records successful mutating HTTP requests to the append-only audit log.
func AuditMutations(rec *audit.Recorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rec == nil || !isMutating(r.Method) || shouldSkipAudit(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			if status >= 200 && status < 400 {
				_ = rec.Record(r.Context(), audit.Entry{
					Action:      "http." + strings.ToLower(r.Method),
					Entity_type: "route",
					Entity_id:   r.URL.Path,
					After: map[string]any{
						"status":       status,
						"content_type": r.Header.Get("Content-Type"),
						"bytes":        ww.BytesWritten(),
					},
				})
			}
		})
	}
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func shouldSkipAudit(path string) bool {
	if strings.HasPrefix(path, "/health") || strings.HasPrefix(path, "/stream") {
		return true
	}
	if strings.HasPrefix(path, "/ws/") {
		return true
	}
	if strings.HasPrefix(path, "/docs") || strings.HasPrefix(path, "/openapi") || strings.HasPrefix(path, "/schemas") {
		return true
	}
	return false
}
