package settings

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
	"github.com/Ze-uus/talent-backend/internal/response"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type handler struct {
	svc *SettingsService
}

func newHandler(svc *SettingsService) *handler { return &handler{svc: svc} }

type std_output struct {
	Status int `json:"-"`
	Body   response.Response
}

func (h *handler) register(api huma.API) {
	h.registerProfile(api)
	h.registerPassword(api)
	h.registerTOTP(api)
	h.registerSessions(api)
}

func (h *handler) registerProfile(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "settings_get_profile",
		Method:      http.MethodGet,
		Path:        "/settings/profile",
		Summary:     "Get own profile",
		Tags:        []string{"settings"},
	}, func(ctx context.Context, _ *struct{}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		profile, err := h.svc.GetProfile(ctx, u.ID)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(profile, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "settings_patch_profile",
		Method:      http.MethodPatch,
		Path:        "/settings/profile",
		Summary:     "Update name or avatar",
		Tags:        []string{"settings"},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Full_name  *string `json:"full_name,omitempty"`
			Avatar_url *string `json:"avatar_url,omitempty"`
		}
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		patch := store.UserPatch{
			Full_name:  in.Body.Full_name,
			Avatar_url: in.Body.Avatar_url,
		}
		if err := h.svc.PatchProfile(ctx, u.ID, patch); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "profile_updated")}, nil
	})
}

func (h *handler) registerPassword(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "settings_change_password",
		Method:      http.MethodPatch,
		Path:        "/settings/password",
		Summary:     "Change password",
		Tags:        []string{"settings"},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Old_password string `json:"old_password"`
			New_password string `json:"new_password"`
		}
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		if err := h.svc.ChangePassword(ctx, u.ID, in.Body.Old_password, in.Body.New_password); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "password_changed")}, nil
	})
}

func (h *handler) registerTOTP(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "settings_totp_enroll",
		Method:      http.MethodPost,
		Path:        "/settings/totp/enroll",
		Summary:     "Re-enroll TOTP",
		Tags:        []string{"settings"},
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
		}, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "settings_totp_verify",
		Method:      http.MethodPost,
		Path:        "/settings/totp/verify",
		Summary:     "Activate TOTP",
		Tags:        []string{"settings"},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Code string `json:"code"`
		}
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		if err := h.svc.VerifyTOTP(ctx, u.ID, in.Body.Code); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "totp_enabled")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "settings_totp_disable",
		Method:      http.MethodPost,
		Path:        "/settings/totp/disable",
		Summary:     "Disable TOTP — talent only",
		Tags:        []string{"settings"},
	}, func(ctx context.Context, _ *struct{}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		if u.Role != store.Role_talent {
			return &std_output{Status: 403, Body: response.Fail("forbidden")}, nil
		}
		if err := h.svc.DisableTOTP(ctx, u.ID); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "totp_disabled")}, nil
	})
}

func (h *handler) registerSessions(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "settings_list_sessions",
		Method:      http.MethodGet,
		Path:        "/settings/sessions",
		Summary:     "List active sessions",
		Tags:        []string{"settings"},
	}, func(ctx context.Context, _ *struct{}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		sessions, err := h.svc.ListSessions(ctx, u.ID)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(sessions, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "settings_revoke_session",
		Method:      http.MethodDelete,
		Path:        "/settings/sessions/{id}",
		Summary:     "Revoke specific session",
		Tags:        []string{"settings"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		if err := h.svc.RevokeSession(ctx, in.ID, u.ID); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "session_revoked")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "settings_revoke_all_sessions",
		Method:      http.MethodDelete,
		Path:        "/settings/sessions",
		Summary:     "Revoke all sessions",
		Tags:        []string{"settings"},
	}, func(ctx context.Context, _ *struct{}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		if err := h.svc.RevokeAllSessions(ctx, u.ID); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "sessions_revoked")}, nil
	})
}
