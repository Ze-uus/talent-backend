package talent

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
	"github.com/Ze-uus/talent-backend/internal/response"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type handler struct {
	svc *TalentService
}

func newHandler(svc *TalentService) *handler { return &handler{svc: svc} }

type std_output struct {
	Body response.Response
}

func (h *handler) register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "talent_get_me",
		Method:      http.MethodGet,
		Path:        "/talent/me",
		Summary:     "Get talent profile",
		Tags:        []string{"talent"},
	}, func(ctx context.Context, _ *struct{}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || u.Role != store.Role_talent {
			return &std_output{Body: response.Fail("insufficient_role")}, nil
		}
		profile, err := h.svc.GetProfile(ctx, u.ID)
		if err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Body: response.Ok(profile, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "talent_patch_me",
		Method:      http.MethodPatch,
		Path:        "/talent/me",
		Summary:     "Update talent profile",
		Tags:        []string{"talent"},
	}, func(ctx context.Context, in *struct {
		Body struct {
			Bio           *string `json:"bio,omitempty"`
			Portfolio_url *string `json:"portfolio_url,omitempty"`
		}
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || u.Role != store.Role_talent {
			return &std_output{Body: response.Fail("insufficient_role")}, nil
		}
		talent, err := h.svc.GetProfile(ctx, u.ID)
		if err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		patch := store.TalentPatch{
			Bio:           in.Body.Bio,
			Portfolio_url: in.Body.Portfolio_url,
		}
		if err := h.svc.PatchProfile(ctx, talent.ID, patch); err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Body: response.Ok(nil, "profile_updated")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "talent_list_cycles",
		Method:      http.MethodGet,
		Path:        "/talent/me/cycles",
		Summary:     "List assigned cycles",
		Tags:        []string{"talent"},
	}, func(ctx context.Context, _ *struct{}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || u.Role != store.Role_talent {
			return &std_output{Body: response.Fail("insufficient_role")}, nil
		}
		talent, err := h.svc.GetProfile(ctx, u.ID)
		if err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		cycles, err := h.svc.ListMyCycles(ctx, talent.ID)
		if err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Body: response.Ok(cycles, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "talent_get_cycle",
		Method:      http.MethodGet,
		Path:        "/talent/me/cycles/{cid}",
		Summary:     "Get cycle detail and tracking link",
		Tags:        []string{"talent"},
	}, func(ctx context.Context, in *struct {
		Cid string `path:"cid"`
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || u.Role != store.Role_talent {
			return &std_output{Body: response.Fail("insufficient_role")}, nil
		}
		talent, err := h.svc.GetProfile(ctx, u.ID)
		if err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		assignment, link, err := h.svc.GetMyCycle(ctx, talent.ID, in.Cid)
		if err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Body: response.Ok(map[string]any{
			"assignment":    assignment,
			"tracking_link": link,
		}, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "talent_get_cycle_stats",
		Method:      http.MethodGet,
		Path:        "/talent/me/cycles/{cid}/stats",
		Summary:     "Live conversion stats for cycle",
		Tags:        []string{"talent"},
	}, func(ctx context.Context, in *struct {
		Cid string `path:"cid"`
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || u.Role != store.Role_talent {
			return &std_output{Body: response.Fail("insufficient_role")}, nil
		}
		talent, err := h.svc.GetProfile(ctx, u.ID)
		if err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		stats, err := h.svc.GetCycleStats(ctx, talent.ID, in.Cid)
		if err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Body: response.Ok(stats, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "talent_request_expansion",
		Method:      http.MethodPost,
		Path:        "/talent/me/cycles/{cid}/expand",
		Summary:     "Request budget expansion",
		Tags:        []string{"talent"},
	}, func(ctx context.Context, in *struct {
		Cid string `path:"cid"`
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || u.Role != store.Role_talent {
			return &std_output{Body: response.Fail("insufficient_role")}, nil
		}
		talent, err := h.svc.GetProfile(ctx, u.ID)
		if err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		if err := h.svc.RequestExpansion(ctx, talent.ID, in.Cid, u.ID); err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Body: response.Ok(nil, "expansion_requested")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "talent_get_history",
		Method:      http.MethodGet,
		Path:        "/talent/me/history",
		Summary:     "Past cycles and payouts",
		Tags:        []string{"talent"},
	}, func(ctx context.Context, _ *struct{}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || u.Role != store.Role_talent {
			return &std_output{Body: response.Fail("insufficient_role")}, nil
		}
		talent, err := h.svc.GetProfile(ctx, u.ID)
		if err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		history, err := h.svc.GetHistory(ctx, talent.ID)
		if err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Body: response.Ok(history, "ok")}, nil
	})
}
