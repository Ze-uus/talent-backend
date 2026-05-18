package tracking

import (
	"context"
	"errors"
	"time"

	"github.com/Ze-uus/talent-backend/internal/store"
)

type TrackingService struct {
	st store.Store
}

func New(s store.Store) *TrackingService { return &TrackingService{st: s} }

// LogEvent validates the token and records a conversion event.
func (s *TrackingService) LogEvent(ctx context.Context, token, event_type, kpb_type, idempotency_key string) error {
	link, err := s.st.GetTrackingLinkByToken(ctx, token)
	if err != nil {
		return errors.New("invalid_token")
	}
	if !link.Active {
		return errors.New("token_inactive")
	}
	cycle, err := s.st.GetCycleByID(ctx, link.Cycle_id)
	if err != nil {
		return err
	}
	return s.st.LogConversionEvent(ctx, store.ConversionEvent{
		Link_token:      token,
		Talent_id:       link.Talent_id,
		Campaign_id:     link.Campaign_id,
		Cycle_id:        link.Cycle_id,
		Pipeline_type:   store.Campaign_type(cycle.Campaign_type),
		Event_type:      event_type,
		KPB_type:        kpb_type,
		Idempotency_key: idempotency_key,
		Occurred_at:     time.Now().UTC(),
	})
}
