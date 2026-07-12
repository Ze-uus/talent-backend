package tracking

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/store"
	ws "github.com/Ze-uus/talent-backend/internal/websocket"
)

// ErrDuplicateEvent is returned when the idempotency_key was already recorded.
var ErrDuplicateEvent = errors.New("duplicate_event")

type TrackingService struct {
	st            store.Store
	conversion_ch chan<- domain.ConversionEvent
	log           *slog.Logger
}

func New(s store.Store, conversion_ch chan<- domain.ConversionEvent, log *slog.Logger) *TrackingService {
	return &TrackingService{st: s, conversion_ch: conversion_ch, log: log}
}

// LogEvent validates the token, records a conversion event, and pushes a WS shard.
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

	occurred_at := time.Now().UTC()
	if err := s.st.LogConversionEvent(ctx, store.ConversionEvent{
		Link_token:      token,
		Talent_id:       link.Talent_id,
		Campaign_id:     link.Campaign_id,
		Cycle_id:        link.Cycle_id,
		Pipeline_type:   store.Campaign_type(cycle.Campaign_type),
		Event_type:      event_type,
		KPB_type:        kpb_type,
		Idempotency_key: idempotency_key,
		Occurred_at:     occurred_at,
	}); err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateEvent
		}
		return err
	}

	cycle_total, _ := s.st.GetTotalConversions(ctx, link.Cycle_id)
	talent_total, _ := s.st.GetTalentConversions(ctx, link.Talent_id, link.Cycle_id)

	ws.TryPush(s.conversion_ch, domain.ConversionEvent{
		Update_type:   "created",
		Talent_id:     link.Talent_id,
		Cycle_id:      link.Cycle_id,
		Campaign_id:   link.Campaign_id,
		Event_type:    event_type,
		KPB_type:      kpb_type,
		Pipeline_type: string(cycle.Campaign_type),
		Occurred_at:   occurred_at,
		Shard: &domain.ConversionShard{
			Cycle_total:  cycle_total,
			Talent_total: talent_total,
		},
	}, s.log, "conversion_ch full, real-time event dropped", "talent_id", link.Talent_id)

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
