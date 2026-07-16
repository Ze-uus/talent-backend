package assignment

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
	"github.com/Ze-uus/talent-backend/internal/response"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type handler struct {
	svc *AssignmentService
}

func newHandler(svc *AssignmentService) *handler { return &handler{svc: svc} }

type std_output struct {
	Status int `json:"-"`
	Body   response.Response
}

func (h *handler) register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "assignments_solver",
		Method:      http.MethodGet,
		Path:        "/admin/campaigns/{id}/cycles/{cid}/solver",
		Summary:     "Run qualifier + Hungarian solver",
		Tags:        []string{"assignments"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Cid string `path:"cid"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		output, err := h.svc.RunSolver(ctx, in.Cid)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(output, "solver_complete")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "assignments_confirm",
		Method:      http.MethodPost,
		Path:        "/admin/campaigns/{id}/cycles/{cid}/assign",
		Summary:     "Confirm or override assignments",
		Tags:        []string{"assignments"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Cid  string `path:"cid"`
		Body struct {
			Confirmed    []ConfirmedAssignment `json:"confirmed"`
			Solver_output SolverOutput         `json:"solver_output"`
			Is_override  bool                  `json:"is_override"`
		}
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || !isAdminRole(u.Role) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		err := h.svc.ConfirmAssignments(ctx, in.Cid, in.ID, u.ID, in.Body.Confirmed, in.Body.Solver_output, in.Body.Is_override)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "assignments_confirmed")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "assignments_list",
		Method:      http.MethodGet,
		Path:        "/admin/campaigns/{id}/cycles/{cid}/assignments",
		Summary:     "List assignments for a cycle",
		Tags:        []string{"assignments"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Cid string `path:"cid"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		assignments, err := h.svc.st.ListAssignedTalents(ctx, in.Cid)
		if err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(assignments, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "assignments_patch",
		Method:      http.MethodPatch,
		Path:        "/admin/campaigns/{id}/cycles/{cid}/assignments/{tid}",
		Summary:     "Update assignment",
		Tags:        []string{"assignments"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Cid  string `path:"cid"`
		Tid  string `path:"tid"`
		Body struct {
			Status        *string  `json:"status,omitempty"`
			Breakout_flag *bool    `json:"breakout_flag,omitempty"`
			Match_score   *float64 `json:"match_score,omitempty"`
		}
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		patch := store.AssignmentPatch{
			Status:        in.Body.Status,
			Breakout_flag: in.Body.Breakout_flag,
			Match_score:   in.Body.Match_score,
		}
		if err := h.svc.st.UpdateAssignment(ctx, in.Tid, in.Cid, patch); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "assignment_updated")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "assignments_remove",
		Method:      http.MethodDelete,
		Path:        "/admin/campaigns/{id}/cycles/{cid}/assignments/{tid}",
		Summary:     "Remove assignment",
		Tags:        []string{"assignments"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Cid string `path:"cid"`
		Tid string `path:"tid"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		status := "removed_payout"
		if err := h.svc.st.UpdateAssignment(ctx, in.Tid, in.Cid, store.AssignmentPatch{Status: &status}); err != nil {
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "assignment_removed")}, nil
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
