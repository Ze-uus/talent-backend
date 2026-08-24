package payout

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
	"github.com/Ze-uus/talent-backend/internal/response"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type handler struct {
	svc *PayoutService
}

func newHandler(svc *PayoutService) *handler { return &handler{svc: svc} }

type std_output struct {
	Status int `json:"-"`
	Body   response.Response
}

func (h *handler) register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "payouts_list",
		Method:      http.MethodGet,
		Path:        "/admin/campaigns/{id}/cycles/{cid}/payouts",
		Summary:     "List payouts for a cycle",
		Tags:        []string{"payouts"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Cid string `path:"cid"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		payouts, err := h.svc.st.ListPayoutsByCycle(ctx, in.Cid)
		if err != nil {
			return &std_output{Status: response.ErrorStatus(err), Body: response.Fail(response.ErrorCode(err))}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(payouts, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "payouts_get",
		Method:      http.MethodGet,
		Path:        "/admin/campaigns/{id}/cycles/{cid}/payouts/{tid}",
		Summary:     "Get payout record",
		Tags:        []string{"payouts"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Cid string `path:"cid"`
		Tid string `path:"tid"`
	}) (*std_output, error) {
		if !isAdminOrAbove(ctx) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		record, err := h.svc.st.GetPayoutRecord(ctx, in.Tid, in.Cid)
		if err != nil {
			return &std_output{Status: response.ErrorStatus(err), Body: response.Fail(response.ErrorCode(err))}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(record, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "payouts_approve",
		Method:      http.MethodPost,
		Path:        "/admin/campaigns/{id}/cycles/{cid}/payouts/{tid}/approve",
		Summary:     "Approve payout",
		Tags:        []string{"payouts"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Cid string `path:"cid"`
		Tid string `path:"tid"`
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || !isAdminRole(u.Role) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.ApprovePayout(ctx, in.Tid, in.Cid, u.ID); err != nil {
			return &std_output{Status: response.ErrorStatus(err), Body: response.Fail(response.ErrorCode(err))}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "payout_approved")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "payouts_mark_paid",
		Method:      http.MethodPost,
		Path:        "/admin/campaigns/{id}/cycles/{cid}/payouts/{tid}/pay",
		Summary:     "Mark payout as paid",
		Tags:        []string{"payouts"},
	}, func(ctx context.Context, in *struct {
		ID  string `path:"id"`
		Cid string `path:"cid"`
		Tid string `path:"tid"`
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || !isAdminRole(u.Role) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.MarkPayoutPaid(ctx, in.Tid, in.Cid, u.ID); err != nil {
			return &std_output{Status: response.ErrorStatus(err), Body: response.Fail(response.ErrorCode(err))}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "payout_paid")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "payouts_flag",
		Method:      http.MethodPost,
		Path:        "/admin/campaigns/{id}/cycles/{cid}/payouts/{tid}/flag",
		Summary:     "Flag payout for review",
		Tags:        []string{"payouts"},
	}, func(ctx context.Context, in *struct {
		ID   string `path:"id"`
		Cid  string `path:"cid"`
		Tid  string `path:"tid"`
		Body struct {
			Reason string `json:"reason"`
		}
	}) (*std_output, error) {
		u, ok := ctxkeys.UserFromContext(ctx)
		if !ok || !isAdminRole(u.Role) {
			return &std_output{Status: 403, Body: response.Fail("insufficient_role")}, nil
		}
		if err := h.svc.FlagPayout(ctx, in.Tid, in.Cid, in.Body.Reason, u.ID); err != nil {
			return &std_output{Status: response.ErrorStatus(err), Body: response.Fail(response.ErrorCode(err))}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "payout_flagged")}, nil
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
