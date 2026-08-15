package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
	"github.com/Ze-uus/talent-backend/internal/response"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type handler struct {
	svc *AuthService
}

func newHandler(svc *AuthService) *handler {
	return &handler{svc: svc}
}

// ─── Output types ─────────────────────────────────────────────────────────────

type std_output struct {
	Status int `json:"-"`
	Body   response.Response
}

type brand_access_output struct {
	Status     int    `json:"-"`
	Set_cookie string `header:"Set-Cookie"`
	Body       response.Response
}

// ─── Register ─────────────────────────────────────────────────────────────────

func (h *handler) register(api huma.API) {
	h.registerTalentAuth(api)
	h.registerStaffInvites(api)
	h.registerTOTP(api)
	h.registerBrandAccess(api)
}

// ─── Talent auth ──────────────────────────────────────────────────────────────

func (h *handler) registerTalentAuth(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "auth_register",
		Method:      http.MethodPost,
		Path:        "/auth/register",
		Summary:     "Talent local registration",
		Tags:        []string{"auth"},
		Security:    []map[string][]string{},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Email        string `json:"email"`
			Password     string `json:"password"`
			Full_name    string `json:"full_name"`
			Phone_number string `json:"phone_number,omitempty"`
		}
	}) (*std_output, error) {
		if err := h.svc.RegisterTalent(ctx, in.Body.Email, in.Body.Password, in.Body.Full_name, in.Body.Phone_number); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "registration_pending_approval")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth_login",
		Method:      http.MethodPost,
		Path:        "/auth/login",
		Summary:     "Login for all roles",
		Tags:        []string{"auth"},
		Security:    []map[string][]string{},
	}, func(ctx context.Context, in *struct {
		X_forwarded_for string `header:"X-Forwarded-For"`
		User_agent      string `header:"User-Agent"`
		Body            struct {
			Email     string  `json:"email"`
			Password  string  `json:"password"`
			Totp_code *string `json:"totp_code,omitempty"`
		}
	}) (*std_output, error) {
		var totpCode string
		if in.Body.Totp_code != nil {
			totpCode = *in.Body.Totp_code
		}
		result, err := h.svc.Login(ctx, in.Body.Email, in.Body.Password, totpCode, in.X_forwarded_for, in.User_agent)
		if err != nil {
			status, msg := mapLoginError(err)
			return &std_output{Status: status, Body: response.Fail(msg)}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(map[string]any{
			"token":              result.Token,
			"user_id":            result.User_id,
			"role":               result.Role,
			"status":             result.Status,
			"active":             result.Active,
			"require_totp_setup": result.Require_totp_setup,
			"totp_recheck_due":   result.Totp_recheck_due,
		}, "login_success")}, nil
	})

	// Returns the Google OAuth URL; client redirects the user to auth_url.
	huma.Register(api, huma.Operation{
		OperationID: "auth_google",
		Method:      http.MethodGet,
		Path:        "/auth/google",
		Summary:     "Get Google OAuth consent URL",
		Tags:        []string{"auth"},
		Security:    []map[string][]string{},
	}, func(ctx context.Context, _ *struct{}) (*std_output, error) {
		auth_url, state, err := h.svc.GoogleAuthURL()
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(map[string]string{
			"auth_url": auth_url,
			"state":    state,
		}, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth_google_callback",
		Method:      http.MethodGet,
		Path:        "/auth/google/callback",
		Summary:     "Google OAuth callback — exchange code for session",
		Tags:        []string{"auth"},
		Security:    []map[string][]string{},
	}, func(ctx context.Context, in *struct {
		Code string `query:"code" required:"true"`
	}) (*std_output, error) {
		u, err := h.svc.ExchangeGoogleCode(ctx, in.Code)
		if err != nil {
			status, msg := mapLoginError(err)
			return &std_output{Status: status, Body: response.Fail(msg)}, nil
		}
		result, err := h.svc.issueSession(ctx, u, "", "")
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail("session_create_failed")}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(map[string]any{
			"token":              result.Token,
			"user_id":            result.User_id,
			"role":               result.Role,
			"status":             result.Status,
			"active":             result.Active,
			"require_totp_setup": result.Require_totp_setup,
		}, "login_success")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth_logout",
		Method:      http.MethodPost,
		Path:        "/auth/logout",
		Summary:     "Invalidate session",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, in *struct {
		Authorization string `header:"Authorization"`
	}) (*std_output, error) {
		token := extractBearer(in.Authorization)
		if token == "" {
			return &std_output{Status: 401, Body: response.Fail("missing_token")}, nil
		}
		_ = h.svc.Logout(ctx, token)
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "logged_out")}, nil
	})
}

// ─── Staff invites ────────────────────────────────────────────────────────────

func (h *handler) registerStaffInvites(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "auth_invite_superadmin",
		Method:      http.MethodPost,
		Path:        "/auth/superadmin/invite",
		Summary:     "Invite a new superadmin (superadmin only)",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Email     string `json:"email"`
			Full_name string `json:"full_name"`
		}
	}) (*std_output, error) {
		caller, ok := ctxkeys.UserFromContext(ctx)
		if !ok || caller.Role != store.Role_superadmin {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		token, err := h.svc.InviteSuperAdmin(ctx, in.Body.Email, in.Body.Full_name)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(map[string]string{"invite_token": token}, "invite_sent")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth_invite_admin",
		Method:      http.MethodPost,
		Path:        "/auth/admin/invite",
		Summary:     "Invite a new admin (superadmin only)",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Email     string `json:"email"`
			Full_name string `json:"full_name"`
		}
	}) (*std_output, error) {
		caller, ok := ctxkeys.UserFromContext(ctx)
		if !ok || caller.Role != store.Role_superadmin {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		token, err := h.svc.InviteAdmin(ctx, in.Body.Email, in.Body.Full_name)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(map[string]string{"invite_token": token}, "invite_sent")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth_invite_manager",
		Method:      http.MethodPost,
		Path:        "/auth/manager/invite",
		Summary:     "Invite a new campaign manager (admin or above)",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Email     string `json:"email"`
			Full_name string `json:"full_name"`
		}
	}) (*std_output, error) {
		caller, ok := ctxkeys.UserFromContext(ctx)
		if !ok || (caller.Role != store.Role_superadmin && caller.Role != store.Role_admin) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		token, err := h.svc.InviteCampaignManager(ctx, in.Body.Email, in.Body.Full_name)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(map[string]string{"invite_token": token}, "invite_sent")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth_verify_invite",
		Method:      http.MethodPost,
		Path:        "/auth/verify-invite",
		Summary:     "Set password from invite token (all staff roles)",
		Tags:        []string{"auth"},
		Security:    []map[string][]string{},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Invite_token string `json:"invite_token"`
			Password     string `json:"password"`
		}
	}) (*std_output, error) {
		u, err := h.svc.VerifyInvite(ctx, in.Body.Invite_token, in.Body.Password)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(map[string]string{
			"id":   u.ID,
			"role": string(u.Role),
		}, "account_activated")}, nil
	})
}

// ─── TOTP ─────────────────────────────────────────────────────────────────────

func (h *handler) registerTOTP(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "auth_totp_enroll",
		Method:      http.MethodPost,
		Path:        "/auth/totp/enroll",
		Summary:     "Enroll TOTP — returns secret and QR URI",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, _ *struct{}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		secret, qr_uri, err := h.svc.EnrollTOTP(ctx, u.ID)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(map[string]string{
			"secret": secret,
			"qr_uri": qr_uri,
		}, "totp_enrolled")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "auth_totp_verify",
		Method:      http.MethodPost,
		Path:        "/auth/totp/verify",
		Summary:     "Verify TOTP code — activates 2FA",
		Tags:        []string{"auth"},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Code string `json:"code"`
		}
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		if err := h.svc.VerifyAndEnableTOTP(ctx, u.ID, in.Body.Code); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "totp_activated")}, nil
	})
}

// ─── Brand contact access ─────────────────────────────────────────────────────

func (h *handler) registerBrandAccess(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "brand_contact_access",
		Method:      http.MethodPost,
		Path:        "/brand/view/{token}/access",
		Summary:     "Brand contact login — validates password and sets session cookie",
		Tags:        []string{"brand"},
		Security:    []map[string][]string{},
	}, func(ctx context.Context, in *struct {
		Token string `path:"token"`
		Body  struct {
			Password string `json:"password"`
		}
	}) (*brand_access_output, error) {
		contact, err := h.svc.ValidateBrandContactAccess(ctx, in.Token, in.Body.Password)
		if err != nil {
			return &brand_access_output{Status: 401, Body: response.Fail("invalid_credentials")}, nil
		}
		cookie := fmt.Sprintf(
			"brand_contact_session=%s; HttpOnly; Path=/; Max-Age=%d",
			contact.Viewer_token,
			int(viewer_cookie_ttl.Seconds()),
		)
		return &brand_access_output{
			Status:     http.StatusOK,
			Set_cookie: cookie,
			Body:       response.Ok(nil, "access_granted"),
		}, nil
	})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// mapLoginError maps known auth failures to HTTP status + stable client codes.
// Unexpected errors (e.g. store failures) never leak raw SQL to the client.
func mapLoginError(err error) (status int, message string) {
	switch {
	case errors.Is(err, err_invalid_credentials):
		return http.StatusUnauthorized, err_invalid_credentials.Error()
	case errors.Is(err, err_totp_required):
		return http.StatusUnauthorized, err_totp_required.Error()
	case errors.Is(err, err_invalid_totp):
		return http.StatusUnauthorized, err_invalid_totp.Error()
	case errors.Is(err, err_account_pending):
		return http.StatusForbidden, err_account_pending.Error()
	case errors.Is(err, err_account_suspended):
		return http.StatusForbidden, err_account_suspended.Error()
	case errors.Is(err, err_account_banned):
		return http.StatusForbidden, err_account_banned.Error()
	case errors.Is(err, err_account_deleted):
		return http.StatusForbidden, err_account_deleted.Error()
	case errors.Is(err, err_account_rejected):
		return http.StatusForbidden, err_account_rejected.Error()
	case errors.Is(err, err_account_invited):
		return http.StatusForbidden, err_account_invited.Error()
	default:
		return http.StatusInternalServerError, "login_failed"
	}
}

func extractBearer(authorization string) string {
	if strings.HasPrefix(authorization, "Bearer ") {
		return strings.TrimPrefix(authorization, "Bearer ")
	}
	return ""
}
