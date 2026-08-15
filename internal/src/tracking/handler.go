package tracking

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Ze-uus/talent-backend/internal/response"
)

type handler struct {
	svc *TrackingService
}

func newHandler(svc *TrackingService) *handler { return &handler{svc: svc} }

type std_output struct {
	Status int `json:"-"`
	Body   response.Response
}

func (h *handler) register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "tracking_presentation",
		Method:      http.MethodGet,
		Path:        "/t/{token}",
		Summary:     "Public campaign presentation",
		Tags:        []string{"tracking"},
		Security:    []map[string][]string{},
	}, func(ctx context.Context, in *struct {
		Token string `path:"token"`
	}) (*std_output, error) {
		presentation, err := h.svc.GetPresentation(ctx, in.Token)
		if err != nil {
			if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrPresentationUnavailable) {
				return &std_output{Status: http.StatusNotFound, Body: response.Fail(response.ErrNotFound)}, nil
			}
			return &std_output{Status: http.StatusInternalServerError, Body: response.Fail(response.ErrInternal)}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(presentation, "ok")}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "track_event",
		Method:      http.MethodPost,
		Path:        "/t/{token}",
		Summary:     "Conversion event ingestion",
		Tags:        []string{"tracking"},
		Security:    []map[string][]string{},
	}, func(ctx context.Context, in *struct {
		Token string `path:"token"`
		Body  struct {
			Event_type      string `json:"event_type"`
			KPB_type        string `json:"kpb_type,omitempty"`
			Idempotency_key string `json:"idempotency_key"`
		}
	}) (*std_output, error) {
		if err := h.svc.LogEvent(ctx, in.Token, in.Body.Event_type, in.Body.KPB_type, in.Body.Idempotency_key); err != nil {
			if errors.Is(err, ErrDuplicateEvent) {
				return &std_output{Status: http.StatusNoContent, Body: response.Ok(nil, "already_logged")}, nil
			}
			if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrPresentationUnavailable) {
				return &std_output{Status: http.StatusNotFound, Body: response.Fail(response.ErrNotFound)}, nil
			}
			if err.Error() == "event_type_required" || err.Error() == "idempotency_key_required" {
				return &std_output{Status: http.StatusUnprocessableEntity, Body: response.Fail(err.Error())}, nil
			}
			return &std_output{Status: 500, Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Status: http.StatusOK, Body: response.Ok(nil, "event_logged")}, nil
	})
}
