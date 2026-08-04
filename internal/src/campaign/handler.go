package campaign

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
	"github.com/Ze-uus/talent-backend/internal/response"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type handler struct {
	svc *CampaignService
}

func newHandler(svc *CampaignService) *handler { return &handler{svc: svc} }

type std_output struct {
	Status int `json:"-"`
	Body   response.Response
}

func (h *handler) register(api huma.API) {
	h.registerCampaigns(api)
	h.registerCycles(api)
}

func (h *handler) registerCampaigns(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "campaigns_create",
		Method:      http.MethodPost,
		Path:        "/admin/campaigns",
		Summary:     "Create campaign",
		Tags:        []string{"campaigns"},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Brand_id      string  `json:"brand_id"`
			Name          string  `json:"name"`
			Campaign_type string  `json:"campaign_type"`
			Total_budget  float64 `json:"total_budget"`
			Target_cpa    float64 `json:"target_cpa"`
			Max_cpa       float64 `json:"max_cpa"`
			Audience      string  `json:"audience"`
			Cycle_length  int     `json:"cycle_length"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		c := store.Campaign{
			Brand_id:      in.Body.Brand_id,
			Name:          in.Body.Name,
			Campaign_type: store.Campaign_type(in.Body.Campaign_type),
			Total_budget:  in.Body.Total_budget,
			Target_cpa:    in.Body.Target_cpa,
			Max_cpa:       in.Body.Max_cpa,
			Audience:      in.Body.Audience,
			Cycle_length:  in.Body.Cycle_length,
		}
		created, err := h.svc.Create(ctx, c)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(created, "campaign_created")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "campaigns_list",
		Method:      http.MethodGet,
		Path:        "/admin/campaigns",
		Summary:     "List campaigns",
		Tags:        []string{"campaigns"},
	}, func(ctx context.Context, in *struct {
		Status   string `query:"status"`
		Brand_id string `query:"brand_id"`
		Limit    int    `query:"limit"`
		Offset   int    `query:"offset"`
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		if u.Role == store.Role_campaign_manager {
			campaigns, err := h.svc.st.GetCampaignsByManagerID(ctx, u.ID)
			if err != nil {
				return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
			}
			return &std_output{Status: http.StatusOK, Body: response.Ok(campaigns, "ok")}, nil
		}
		if !isAdminRole(u.Role) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		campaigns, err := h.svc.List(ctx, store.CampaignFilter{Status: in.Status, Brand_id: in.Brand_id, Limit: in.Limit, Offset: in.Offset})
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(campaigns, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "campaigns_get",
		Method:      http.MethodGet,
		Path:        "/admin/campaigns/{id}",
		Summary:     "Get campaign",
		Tags:        []string{"campaigns"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		c, err := h.svc.Get(ctx, in.ID)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		if u.Role == store.Role_campaign_manager {
			if !managerOwnsCampaign(ctx, h.svc.st, u.ID, in.ID) {
				return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
			}
		} else if !isAdminRole(u.Role) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(c, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "campaigns_patch",
		Method:      http.MethodPatch,
		Path:        "/admin/campaigns/{id}",
		Summary:     "Update campaign",
		Tags:        []string{"campaigns"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Body struct {
			Name             *string `json:"name,omitempty"`
			Status           *string `json:"status,omitempty"`
			Urgency_level    *string `json:"urgency_level,omitempty"`
			Cycle_length     *int    `json:"cycle_length,omitempty"`
			Creators_allowed *bool   `json:"creators_allowed,omitempty"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		patch := store.CampaignPatch{
			Name:             in.Body.Name,
			Status:           in.Body.Status,
			Urgency_level:    in.Body.Urgency_level,
			Cycle_length:     in.Body.Cycle_length,
			Creators_allowed: in.Body.Creators_allowed,
		}
		if err := h.svc.Patch(ctx, in.ID, patch); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "campaign_updated")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "campaigns_archive",
		Method:      http.MethodPost,
		Path:        "/admin/campaigns/{id}/archive",
		Summary:     "Archive campaign",
		Tags:        []string{"campaigns"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.Archive(ctx, in.ID); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "campaign_archived")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "campaigns_assign_manager",
		Method:      http.MethodPost,
		Path:        "/admin/campaigns/{id}/managers",
		Summary:     "Assign campaign manager",
		Tags:        []string{"campaigns"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Body struct {
			Manager_id string `json:"manager_id"`
		}
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || !isAdminRole(u.Role) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.AssignManager(ctx, in.Body.Manager_id, in.ID, u.ID); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "manager_assigned")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "campaigns_unassign_manager",
		Method:      http.MethodDelete,
		Path:        "/admin/campaigns/{id}/managers/{mid}",
		Summary:     "Unassign campaign manager",
		Tags:        []string{"campaigns"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Mid string `path:"mid"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.UnassignManager(ctx, in.Mid, in.ID); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "manager_unassigned")}, nil
	})
}

func (h *handler) registerCycles(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "cycles_create",
		Method:      http.MethodPost,
		Path:        "/admin/campaigns/{id}/cycles",
		Summary:     "Create cycle — runs waterfall algorithm",
		Tags:        []string{"cycles"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Body struct {
			Cycle_budget    float64 `json:"cycle_budget"`
			Cycle_objective string  `json:"cycle_objective"`
			End_date        *string `json:"end_date,omitempty"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		c := store.Cycle{
			Campaign_id:     in.ID,
			Cycle_budget:    in.Body.Cycle_budget,
			Cycle_objective: in.Body.Cycle_objective,
		}
		created, err := h.svc.CreateCycle(ctx, c)
		if err != nil {
			switch err.Error() {
			case "invalid_campaign_cpa", "target_cpa_exceeds_max_cpa", "cycle_budget_below_target_cpa":
				return &std_output{Status: http.StatusUnprocessableEntity, Body: response.Fail(err.Error())}, nil
			}
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(created, "cycle_created")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "cycles_list",
		Method:      http.MethodGet,
		Path:        "/admin/campaigns/{id}/cycles",
		Summary:     "List cycles for campaign",
		Tags:        []string{"cycles"},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id"`
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		if u.Role == store.Role_campaign_manager && !managerOwnsCampaign(ctx, h.svc.st, u.ID, in.ID) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		} else if !isAdminRole(u.Role) && u.Role != store.Role_campaign_manager {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		cycles, err := h.svc.ListCycles(ctx, in.ID)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(cycles, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "cycles_get",
		Method:      http.MethodGet,
		Path:        "/admin/campaigns/{id}/cycles/{cid}",
		Summary:     "Get cycle",
		Tags:        []string{"cycles"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Cid string `path:"cid"`
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok {
			return &std_output{Status: 401, Body: response.Fail("unauthenticated")}, nil
		}
		if u.Role == store.Role_campaign_manager && !managerOwnsCampaign(ctx, h.svc.st, u.ID, in.ID) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		} else if !isAdminRole(u.Role) && u.Role != store.Role_campaign_manager {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		cycle, err := h.svc.GetCycle(ctx, in.Cid)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(cycle, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "cycles_patch",
		Method:      http.MethodPatch,
		Path:        "/admin/campaigns/{id}/cycles/{cid}",
		Summary:     "Update cycle",
		Tags:        []string{"cycles"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Cid  string `path:"cid"`
		Body struct {
			Cycle_budget    *float64 `json:"cycle_budget,omitempty"`
			Cycle_objective *string  `json:"cycle_objective,omitempty"`
			Z_factor        *float64 `json:"z_factor,omitempty"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		patch := store.CyclePatch{
			Cycle_budget:    in.Body.Cycle_budget,
			Cycle_objective: in.Body.Cycle_objective,
			Z_factor:        in.Body.Z_factor,
		}
		if err := h.svc.PatchCycle(ctx, in.Cid, patch); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "cycle_updated")}, nil
	})

	for _, op := range []struct {
		id      string
		path    string
		summary string
		action  func(context.Context, string) error
		msg     string
	}{
		{"cycles_activate", "/admin/campaigns/{id}/cycles/{cid}/activate", "Activate cycle", h.svc.ActivateCycle, "cycle_activated"},
		{"cycles_pause", "/admin/campaigns/{id}/cycles/{cid}/pause", "Pause cycle", h.svc.PauseCycle, "cycle_paused"},
		{"cycles_close", "/admin/campaigns/{id}/cycles/{cid}/close", "Close cycle", h.svc.CloseCycle, "cycle_closed"},
	} {
		op := op
		huma.Register(api, huma.Operation{
			OperationID: op.id,
			Method:      http.MethodPost,
			Path:        op.path,
			Summary:     op.summary,
			Tags:        []string{"cycles"},
		}, func(ctx context.Context, in *struct {
			ID  string `path:"id"`
			Cid string `path:"cid"`
		}) (*std_output, error) {
			if !isAdminOrAbove(ctx) {
				return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
			}
			if err := op.action(ctx, in.Cid); err != nil {
				return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
			}
			return &std_output{Status: http.StatusOK, Body: response.Ok(nil, op.msg)}, nil
		})
	}

	huma.Register(api, huma.Operation{
		OperationID: "cycles_finalise_payouts",
		Method:      http.MethodPost,
		Path:        "/admin/campaigns/{id}/cycles/{cid}/finalise-payouts",
		Summary:     "Finalise payouts — Story 24: lock fallback conversions and compute payouts",
		Tags:        []string{"cycles"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Cid string `path:"cid"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.FinalisePayouts(ctx, in.Cid); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "payouts_finalised")}, nil
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

func managerOwnsCampaign(ctx context.Context, st store.Store, manager_id, campaign_id string) bool {
	campaigns, err := st.GetCampaignsByManagerID(ctx, manager_id)
	if err != nil {
		return false
	}
	for _, c := range campaigns {
		if c.ID == campaign_id {
			return true
		}
	}
	return false
}
