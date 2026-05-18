package tracking

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/Ze-uus/talent-backend/internal/response"
)

type handler struct {
	svc *TrackingService
}

func newHandler(svc *TrackingService) *handler { return &handler{svc: svc} }

type std_output struct {
	Body response.Response
}

func (h *handler) register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "track_event",
		Method:      http.MethodPost,
		Path:        "/track/{token}",
		Summary:     "Conversion event ingestion",
		Tags:        []string{"tracking"},
	}, func(ctx context.Context, in *struct {
		Token string `path:"token"`
		Body  struct {
			Event_type      string `json:"event_type"`
			KPB_type        string `json:"kpb_type,omitempty"`
			Idempotency_key string `json:"idempotency_key"`
		}
	}) (*std_output, error) {
		if err := h.svc.LogEvent(ctx, in.Token, in.Body.Event_type, in.Body.KPB_type, in.Body.Idempotency_key); err != nil {
			return &std_output{Body: response.Fail(err.Error())}, nil
		}
		return &std_output{Body: response.Ok(nil, "event_logged")}, nil
	})
}
