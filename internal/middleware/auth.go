package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
	"github.com/Ze-uus/talent-backend/internal/response"
	authsvc "github.com/Ze-uus/talent-backend/internal/src/auth"
	"github.com/Ze-uus/talent-backend/internal/store"
)

// Authenticate validates the session token from Authorization: Bearer header.
// Missing or invalid tokens are rejected with 401.
func Authenticate(svc *authsvc.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				response.WriteJSON(w, http.StatusUnauthorized, response.Fail("missing_token"))
				return
			}
			user, err := svc.ValidateSession(r.Context(), token)
			if err != nil {
				response.WriteJSON(w, http.StatusUnauthorized, response.Fail(err.Error()))
				return
			}
			ctx := ctxkeys.WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthenticateOptional populates the user in context when a Bearer token is present.
// Requests without a token continue unauthenticated so public routes still work.
// Invalid tokens are rejected with 401.
func AuthenticateOptional(svc *authsvc.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			user, err := svc.ValidateSession(r.Context(), token)
			if err != nil {
				response.WriteJSON(w, http.StatusUnauthorized, response.Fail(err.Error()))
				return
			}
			ctx := ctxkeys.WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole enforces that the authenticated user holds one of the given roles.
// Convention:
//   - Superadmin only:           RequireRole(store.Role_superadmin)
//   - Admin or above:            RequireRole(store.Role_superadmin, store.Role_admin)
//   - Campaign manager or above: RequireRole(store.Role_superadmin, store.Role_admin, store.Role_campaign_manager)
func RequireRole(roles ...store.User_role) func(http.Handler) http.Handler {
	allowed := make(map[store.User_role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := ctxkeys.UserFromContext(r.Context())
			if !ok || !allowed[user.Role] {
				response.WriteJSON(w, http.StatusForbidden, response.Fail("insufficient_role"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AuthenticateBrandContact validates the brand contact session cookie.
// Cookie value is the viewer_token; active status is re-checked on every request
// without re-validating the password.
func AuthenticateBrandContact(svc *authsvc.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("brand_contact_session")
			if err != nil {
				response.WriteJSON(w, http.StatusUnauthorized, response.Fail("missing_session"))
				return
			}
			contact, err := svc.GetActiveBrandContact(r.Context(), cookie.Value)
			if err != nil {
				response.WriteJSON(w, http.StatusUnauthorized, response.Fail("invalid_session"))
				return
			}
			ctx := ctxkeys.WithBrandContact(r.Context(), contact)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) (store.User, bool) {
	return ctxkeys.UserFromContext(ctx)
}

func BrandContactFromContext(ctx context.Context) (store.BrandContact, bool) {
	return ctxkeys.BrandContactFromContext(ctx)
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
