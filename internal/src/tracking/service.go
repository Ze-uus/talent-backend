package tracking

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/response"
	"github.com/Ze-uus/talent-backend/internal/store"
	ws "github.com/Ze-uus/talent-backend/internal/websocket"
)

var (
	ErrDuplicateEvent          = errors.New("duplicate_event")
	ErrInvalidToken            = errors.New("invalid_token")
	ErrPresentationUnavailable = errors.New("presentation_unavailable")
)

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
	event_type = strings.TrimSpace(event_type)
	idempotency_key = strings.TrimSpace(idempotency_key)
	if event_type == "" {
		return response.Validation("event_type_required")
	}
	if idempotency_key == "" {
		return response.Validation("idempotency_key_required")
	}
	link, cycle, _, err := s.activeTrackingContext(ctx, token)
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

func (s *TrackingService) GetPresentation(ctx context.Context, token string) (PresentationView, error) {
	_, cycle, campaign, err := s.activeTrackingContext(ctx, token)
	if err != nil {
		return PresentationView{}, err
	}
	brand, err := s.st.GetBrandByID(ctx, campaign.Brand_id)
	if err != nil {
		return PresentationView{}, err
	}
	kpbLabels := make([]string, 0, len(cycle.KPB_config))
	for _, definition := range cycle.KPB_config {
		kpbLabels = append(kpbLabels, definition.Label)
	}
	return PresentationView{
		Brand: PresentationBrand{
			Name: brand.Name, Industry: brand.Industry, Description: brand.Description,
			Website: brand.Website, LogoURL: brand.Logo_url,
		},
		Campaign: PresentationCampaign{
			HumanID: campaign.Human_id, Name: campaign.Name,
			CampaignType: campaign.Campaign_type, Audience: campaign.Audience,
		},
		Cycle: PresentationCycle{
			HumanID: cycle.Human_id, CycleNumber: cycle.Cycle_number,
			CycleObjective: cycle.Cycle_objective, CampaignType: cycle.Campaign_type,
			KPBLabels: kpbLabels, StartDate: cycle.Start_date, EndDate: cycle.End_date,
		},
		Content: store.EffectiveContent(campaign, cycle),
	}, nil
}

func (s *TrackingService) activeTrackingContext(ctx context.Context, token string) (store.TrackingLink, store.Cycle, store.Campaign, error) {
	link, err := s.st.GetTrackingLinkByToken(ctx, strings.TrimSpace(token))
	if err != nil || !link.Active {
		return store.TrackingLink{}, store.Cycle{}, store.Campaign{}, ErrInvalidToken
	}
	cycle, err := s.st.GetCycleByID(ctx, link.Cycle_id)
	if err != nil {
		return store.TrackingLink{}, store.Cycle{}, store.Campaign{}, err
	}
	campaign, err := s.st.GetCampaignByID(ctx, link.Campaign_id)
	if err != nil {
		return store.TrackingLink{}, store.Cycle{}, store.Campaign{}, err
	}
	if cycle.Campaign_id != campaign.ID ||
		cycle.Status != store.Cycle_active ||
		campaign.Status != store.Campaign_active {
		return store.TrackingLink{}, store.Cycle{}, store.Campaign{}, ErrPresentationUnavailable
	}
	return link, cycle, campaign, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
