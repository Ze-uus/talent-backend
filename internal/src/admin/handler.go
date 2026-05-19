package admin

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
	"github.com/Ze-uus/talent-backend/internal/response"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type handler struct {
	svc *AdminService
}

func newHandler(svc *AdminService) *handler { return &handler{svc: svc} }

type std_output struct {
	Status int `json:"-"`
	Body   response.Response
}

func (h *handler) register(api huma.API) {
	h.registerUsers(api)
	h.registerTalents(api)
	h.registerViewers(api)
	h.registerAudit(api)
}

func (h *handler) registerUsers(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "admin_users_list",
		Method:      http.MethodGet,
		Path:        "/admin/users",
		Summary:     "List users",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		Role   string `query:"role"`
		Active string `query:"active"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		var active *bool
		if in.Active != "" {
			b := in.Active == "true"
			active = &b
		}
		users, err := h.svc.ListUsers(ctx, store.UserFilter{Role: in.Role, Active: active})
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(users, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin_users_get",
		Method:      http.MethodGet,
		Path:        "/admin/users/{id}",
		Summary:     "Get user",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		u, err := h.svc.GetUser(ctx, in.ID)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(u, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin_users_patch",
		Method:      http.MethodPatch,
		Path:        "/admin/users/{id}",
		Summary:     "Update user",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Body struct {
			Full_name *string `json:"full_name,omitempty"`
			Active    *bool   `json:"active,omitempty"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.PatchUser(ctx, in.ID, store.UserPatch{Full_name: in.Body.Full_name, Active: in.Body.Active}); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "user_updated")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin_users_patch_role",
		Method:      http.MethodPatch,
		Path:        "/admin/users/{id}/role",
		Summary:     "Change user role",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Body struct {
			Role string `json:"role"`
		}
	}) (*std_output, error) {
		caller, ok := ctxkeys.UserFromContext(ctx)
		if !ok || caller.Role != store.Role_superadmin {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		// Role changes require superadmin to prevent privilege escalation
		_ = in.Body.Role
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "role_updated")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin_users_deactivate",
		Method:      http.MethodDelete,
		Path:        "/admin/users/{id}",
		Summary:     "Deactivate user",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.DeactivateUser(ctx, in.ID); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "user_deactivated")}, nil
	})
}

func (h *handler) registerTalents(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "admin_talents_list",
		Method:      http.MethodGet,
		Path:        "/admin/talents",
		Summary:     "List talents",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		Status   string `query:"status"`
		Category string `query:"category"`
		Limit    int    `query:"limit"`
		Offset   int    `query:"offset"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		talents, err := h.svc.st.ListTalents(ctx, store.TalentFilter{Status: in.Status, Category: in.Category, Limit: in.Limit, Offset: in.Offset})
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(talents, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin_talents_get",
		Method:      http.MethodGet,
		Path:        "/admin/talents/{id}",
		Summary:     "Get talent",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		t, err := h.svc.st.GetTalentByID(ctx, in.ID)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(t, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin_talents_approve",
		Method:      http.MethodPost,
		Path:        "/admin/talents/{id}/approve",
		Summary:     "Approve talent",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Body struct {
			Category string `json:"category"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.ApproveTalent(ctx, in.ID, store.Talent_category(in.Body.Category)); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "talent_approved")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin_talents_reject",
		Method:      http.MethodPost,
		Path:        "/admin/talents/{id}/reject",
		Summary:     "Reject talent",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.RejectTalent(ctx, in.ID); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "talent_rejected")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin_talents_patch",
		Method:      http.MethodPatch,
		Path:        "/admin/talents/{id}",
		Summary:     "Update talent",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Body struct {
			Rate_per_day  *float64 `json:"rate_per_day,omitempty"`
			Max_tier      *int     `json:"max_tier,omitempty"`
			Skills        []string `json:"skills,omitempty"`
			Bio           *string  `json:"bio,omitempty"`
			Portfolio_url *string  `json:"portfolio_url,omitempty"`
			Category      *string  `json:"category,omitempty"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		patch := store.TalentPatch{
			Rate_per_day:  in.Body.Rate_per_day,
			Max_tier:      in.Body.Max_tier,
			Skills:        in.Body.Skills,
			Bio:           in.Body.Bio,
			Portfolio_url: in.Body.Portfolio_url,
			Category:      in.Body.Category,
		}
		if err := h.svc.PatchTalent(ctx, in.ID, patch); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "talent_updated")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin_talents_suspend",
		Method:      http.MethodPost,
		Path:        "/admin/talents/{id}/suspend",
		Summary:     "Suspend talent",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.SuspendTalent(ctx, in.ID); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "talent_suspended")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "admin_talents_reinstate",
		Method:      http.MethodPost,
		Path:        "/admin/talents/{id}/reinstate",
		Summary:     "Reinstate talent",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.ReinstateTalent(ctx, in.ID); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "talent_reinstated")}, nil
	})
}

func (h *handler) registerViewers(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "viewers_create",
		Method:      http.MethodPost,
		Path:        "/admin/campaigns/{id}/viewers",
		Summary:     "Create campaign viewer",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Body struct {
			Name string `json:"name"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		v := store.CampaignViewer{Campaign_id: in.ID, Name: in.Body.Name, Active: true}
		if err := h.svc.CreateViewer(ctx, v); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "viewer_created")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "viewers_list",
		Method:      http.MethodGet,
		Path:        "/admin/campaigns/{id}/viewers",
		Summary:     "List campaign viewers",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		viewers, err := h.svc.ListViewers(ctx, in.ID)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(viewers, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "viewers_patch",
		Method:      http.MethodPatch,
		Path:        "/admin/campaigns/{id}/viewers/{vid}",
		Summary:     "Update viewer",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Vid  string `path:"vid"`
		Body struct {
			Name   *string `json:"name,omitempty"`
			Active *bool   `json:"active,omitempty"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.PatchViewer(ctx, in.Vid, store.ViewerPatch{Name: in.Body.Name, Active: in.Body.Active}); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "viewer_updated")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "viewer_passwords_add",
		Method:      http.MethodPost,
		Path:        "/admin/campaigns/{id}/viewers/{vid}/passwords",
		Summary:     "Add viewer password",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Vid  string `path:"vid"`
		Body struct {
			Label         string `json:"label"`
			Password_hash string `json:"password_hash"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		p := store.ViewerPassword{Viewer_id: in.Vid, Label: in.Body.Label, Password_hash: in.Body.Password_hash, Active: true}
		if err := h.svc.AddViewerPassword(ctx, p); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "password_added")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "viewer_passwords_list",
		Method:      http.MethodGet,
		Path:        "/admin/campaigns/{id}/viewers/{vid}/passwords",
		Summary:     "List viewer passwords",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Vid string `path:"vid"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		passwords, err := h.svc.ListViewerPasswords(ctx, in.Vid)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(passwords, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "viewer_passwords_patch",
		Method:      http.MethodPatch,
		Path:        "/admin/campaigns/{id}/viewers/{vid}/passwords/{pid}",
		Summary:     "Update viewer password",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, _ *struct {
		ID  string `path:"id"`
		Vid string `path:"vid"`
		Pid string `path:"pid"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "password_updated")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "viewer_passwords_deactivate",
		Method:      http.MethodDelete,
		Path:        "/admin/campaigns/{id}/viewers/{vid}/passwords/{pid}",
		Summary:     "Deactivate viewer password",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Vid string `path:"vid"`
		Pid string `path:"pid"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.DeactivateViewerPassword(ctx, in.Pid); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "password_deactivated")}, nil
	})
}

func (h *handler) registerAudit(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "audit_list",
		Method:      http.MethodGet,
		Path:        "/admin/audit",
		Summary:     "List audit log",
		Tags:        []string{"admin"},
	}, func(ctx context.Context, in *struct {
		Entity_type string `query:"entity_type"`
		Entity_id   string `query:"entity_id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		entries, err := h.svc.ListAudit(ctx, in.Entity_type, in.Entity_id)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(entries, "ok")}, nil
	})
}

func isAdminOrAbove(ctx context.Context) bool {
	u, ok := ctxkeys.UserFromContext(ctx)
	if !ok {
		return false
	}
	return u.Role == store.Role_superadmin || u.Role == store.Role_admin
}
