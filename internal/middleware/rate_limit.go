package middleware

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/Ze-uus/talent-backend/internal/response"
)

type ip_limiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      int
}

func new_ip_limiter(rps int) *ip_limiter {
	return &ip_limiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rps,
	}
}

func (l *ip_limiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists := l.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(l.rps), l.rps)
		l.limiters[ip] = limiter
	}
	return limiter
}

// RateLimit returns a middleware that enforces per-IP request rate limiting.
func RateLimit(rps int) func(http.Handler) http.Handler {
	lim := new_ip_limiter(rps)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			if !lim.get(ip).Allow() {
				response.WriteJSON(w, http.StatusTooManyRequests, response.Fail(response.ErrRateLimited))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
