package brand

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
	"github.com/Ze-uus/talent-backend/internal/response"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type handler struct {
	svc *BrandService
}

func newHandler(svc *BrandService) *handler { return &handler{svc: svc} }

type std_output struct {
	Status int `json:"-"`
	Body   response.Response
}

func (h *handler) register(api huma.API) {
	h.registerAdmin(api)
	h.registerBrandView(api)
}

func (h *handler) registerAdmin(api huma.API) {
	// Create + patch (+ optional logo) and dedicated logo upload are Chi routes
	// — see registerHTTP — so JSON and multipart/form-data both work.

	huma.Register(api, huma.Operation{
		OperationID: "brands_list",
		Method:      http.MethodGet,
		Path:        "/admin/brands",
		Summary:     "List brands",
		Tags:        []string{"brands"},
	}, func(ctx context.Context, in *struct {
		Status string `query:"status"`
		Limit  int    `query:"limit"`
		Offset int    `query:"offset"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		brands, err := h.svc.st.ListBrands(ctx, store.BrandFilter{Status: in.Status, Limit: in.Limit, Offset: in.Offset})
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(brands, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "brands_get",
		Method:      http.MethodGet,
		Path:        "/admin/brands/{id}",
		Summary:     "Get brand",
		Tags:        []string{"brands"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		b, err := h.svc.st.GetBrandByID(ctx, in.ID)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(b, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "brands_suspend",
		Method:      http.MethodPost,
		Path:        "/admin/brands/{id}/suspend",
		Summary:     "Suspend brand",
		Tags:        []string{"brands"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		status := "suspended"
		if err := h.svc.st.UpdateBrand(ctx, in.ID, store.BrandPatch{Status: &status}); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "brand_suspended")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "brands_reinstate",
		Method:      http.MethodPost,
		Path:        "/admin/brands/{id}/reinstate",
		Summary:     "Reinstate brand",
		Tags:        []string{"brands"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		status := "active"
		if err := h.svc.st.UpdateBrand(ctx, in.ID, store.BrandPatch{Status: &status}); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "brand_reinstated")}, nil
	})

	// ─── Brand contacts ───────────────────────────────────────────────────────

	huma.Register(api, huma.Operation{
		OperationID: "brand_contacts_list",
		Method:      http.MethodGet,
		Path:        "/admin/brands/{id}/contacts",
		Summary:     "List brand contacts",
		Tags:        []string{"brands"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		contacts, err := h.svc.st.ListBrandContacts(ctx, in.ID)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(contacts, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "brand_contacts_add",
		Method:      http.MethodPost,
		Path:        "/admin/brands/{id}/contacts",
		Summary:     "Add brand contact — returns plain password once",
		Tags:        []string{"brands"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Body struct {
			First_name string `json:"first_name"`
			Last_name  string `json:"last_name"`
			Role       string `json:"role"`
			Email      string `json:"email"`
			Whatsapp   string `json:"whatsapp"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		contact, plain, err := h.svc.AddContact(ctx, in.ID, in.Body.First_name, in.Body.Last_name, in.Body.Role, in.Body.Email, in.Body.Whatsapp)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(map[string]any{
			"contact":        contact,
			"plain_password": plain,
		}, "contact_added")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "brand_contacts_get",
		Method:      http.MethodGet,
		Path:        "/admin/brands/{id}/contacts/{cid}",
		Summary:     "Get brand contact",
		Tags:        []string{"brands"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Cid string `path:"cid"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		c, err := h.svc.st.GetBrandContactByID(ctx, in.Cid)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(c, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "brand_contacts_patch",
		Method:      http.MethodPatch,
		Path:        "/admin/brands/{id}/contacts/{cid}",
		Summary:     "Update brand contact",
		Tags:        []string{"brands"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Cid  string `path:"cid"`
		Body struct {
			First_name      *string `json:"first_name,omitempty"`
			Last_name       *string `json:"last_name,omitempty"`
			Role            *string `json:"role,omitempty"`
			Email           *string `json:"email,omitempty"`
			Whatsapp_number *string `json:"whatsapp_number,omitempty"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		patch := store.BrandContactPatch{
			First_name:      in.Body.First_name,
			Last_name:       in.Body.Last_name,
			Role:            in.Body.Role,
			Email:           in.Body.Email,
			Whatsapp_number: in.Body.Whatsapp_number,
		}
		if err := h.svc.st.UpdateBrandContact(ctx, in.Cid, patch); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "contact_updated")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "brand_contacts_deactivate",
		Method:      http.MethodDelete,
		Path:        "/admin/brands/{id}/contacts/{cid}",
		Summary:     "Deactivate brand contact",
		Tags:        []string{"brands"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Cid string `path:"cid"`
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || !isAdminRole(u.Role) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.RemoveContact(ctx, in.Cid, u.ID); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "contact_deactivated")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "brand_contacts_regen_password",
		Method:      http.MethodPost,
		Path:        "/admin/brands/{id}/contacts/{cid}/regenerate-password",
		Summary:     "Regenerate brand contact password",
		Tags:        []string{"brands"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Cid string `path:"cid"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		plain, err := h.svc.RegeneratePassword(ctx, in.Cid)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(map[string]string{"plain_password": plain}, "password_regenerated")}, nil
	})
}

func (h *handler) registerBrandView(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "brand_view_dashboard",
		Method:      http.MethodGet,
		Path:        "/brand/view/{token}/dashboard",
		Summary:     "Brand contact dashboard",
		Tags:        []string{"brand"},
	}, func(ctx context.Context, in *struct {
		Token string `path:"token"`
	}) (*std_output, error) {
		_, ok := ctxkeys.BrandContactFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		brand, campaigns, err := h.svc.GetDashboard(ctx, in.Token)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(map[string]any{
			"brand":     brand,
			"campaigns": campaigns,
		}, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "brand_view_live_metrics",
		Method:      http.MethodGet,
		Path:        "/brand/view/{token}/campaigns/{cid}/live",
		Summary:     "Brand contact live campaign metrics",
		Tags:        []string{"brand"},
	}, func(ctx context.Context, in *struct {
		Token string `path:"token"`
		Cid   string `path:"cid"`
	}) (*std_output, error) {
		_, ok := ctxkeys.BrandContactFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		metrics, err := h.svc.GetLiveMetrics(ctx, in.Cid)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(metrics, "ok")}, nil
	})
}

func isAdminOrAbove(ctx context.Context) bool {
	u, ok := ctxkeys.UserFromContext(ctx)
	if !ok {
		return false
	}
	return isAdminRole(u.Role)
}

func isAdminRole(role store.User_role) bool {
	return role == store.Role_superadmin || role == store.Role_admin
}
