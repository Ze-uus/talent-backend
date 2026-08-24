package talent_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Ze-uus/talent-backend/internal/src/talent"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type talentStore struct {
	store.Store
	assignment store.TalentAssignment
	cycle      store.Cycle
	campaign   store.Campaign
	brand      store.Brand
	managers   []store.User
	links      []store.TrackingLink
}

func (s *talentStore) ListAssignmentsByTalent(_ context.Context, talentID string) ([]store.TalentAssignment, error) {
	if s.assignment.Talent_id != talentID {
		return []store.TalentAssignment{}, nil
	}
	return []store.TalentAssignment{s.assignment}, nil
}

func (s *talentStore) GetAssignment(_ context.Context, talentID, cycleID string) (store.TalentAssignment, error) {
	if s.assignment.Talent_id != talentID || s.assignment.Cycle_id != cycleID {
		return store.TalentAssignment{}, pgx.ErrNoRows
	}
	return s.assignment, nil
}

func (s *talentStore) GetCycleByID(_ context.Context, id string) (store.Cycle, error) {
	if s.cycle.ID != id {
		return store.Cycle{}, pgx.ErrNoRows
	}
	return s.cycle, nil
}

func (s *talentStore) GetCampaignByID(_ context.Context, id string) (store.Campaign, error) {
	if s.campaign.ID != id {
		return store.Campaign{}, pgx.ErrNoRows
	}
	return s.campaign, nil
}

func (s *talentStore) GetBrandByID(_ context.Context, id string) (store.Brand, error) {
	if s.brand.ID != id {
		return store.Brand{}, pgx.ErrNoRows
	}
	return s.brand, nil
}

func (s *talentStore) ListManagersByCampaignID(_ context.Context, campaignID string) ([]store.User, error) {
	if s.campaign.ID != campaignID {
		return nil, pgx.ErrNoRows
	}
	return s.managers, nil
}

func (s *talentStore) ListTrackingLinksByCycle(_ context.Context, cycleID string) ([]store.TrackingLink, error) {
	if s.cycle.ID != cycleID {
		return nil, pgx.ErrNoRows
	}
	return s.links, nil
}

func (s *talentStore) GetTalentConversions(_ context.Context, talentID, cycleID string) (float64, error) {
	if s.assignment.Talent_id != talentID || s.assignment.Cycle_id != cycleID {
		return 0, pgx.ErrNoRows
	}
	return 4, nil
}

func (s *talentStore) GetTotalConversions(_ context.Context, cycleID string) (float64, error) {
	if s.cycle.ID != cycleID {
		return 0, pgx.ErrNoRows
	}
	return 10, nil
}

func testStore() *talentStore {
	now := time.Now().UTC()
	return &talentStore{
		assignment: store.TalentAssignment{
			Talent_id: "talent-1", Campaign_id: "campaign-1", Cycle_id: "cycle-1",
			Slot_id: "slot-1", Role_label: "advocate", Status: "active",
			Assignment_source: store.Source_algorithm, Effective_tier: 300000, Assigned_at: now,
		},
		cycle: store.Cycle{
			ID: "cycle-1", Human_id: "CMP-26-01-C1", Campaign_id: "campaign-1",
			Cycle_number: 1, Status: store.Cycle_active, Cycle_objective: "traffic",
			Campaign_type: store.Type_direct_traffic, Start_date: now, End_date: now.Add(7 * 24 * time.Hour),
		},
		campaign: store.Campaign{
			ID: "campaign-1", Human_id: "CMP-26-01", Brand_id: "brand-1",
			Name: "Launch", Status: store.Campaign_active,
			Campaign_type: store.Type_direct_traffic, Audience: "students",
			Target_cpa: 150000, Max_cpa: 300000,
			Content: []store.ContentItem{{ID: "brief", Images: []string{}, Links: []store.ContentLink{}}},
		},
		brand:    store.Brand{ID: "brand-1", Name: "Acme", Logo_url: "https://example.com/logo.png"},
		managers: []store.User{{ID: "manager-1", Full_name: "Campaign Owner", Email: "owner@example.com"}},
		links: []store.TrackingLink{
			{Talent_id: "other-talent", Cycle_id: "cycle-1", Token: "wrong", Active: true},
			{Talent_id: "talent-1", Cycle_id: "cycle-1", Token: "right", Active: true},
		},
	}
}

func TestListMyCyclesReturnsEnrichedTalentSafeShape(t *testing.T) {
	svc := talent.New(testStore(), nil)

	views, err := svc.ListMyCycles(context.Background(), "talent-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 {
		t.Fatalf("expected one cycle, got %d", len(views))
	}
	view := views[0]
	if view.Campaign.Name != "Launch" || view.Brand.Name != "Acme" || view.Cycle.ID != "cycle-1" {
		t.Fatalf("view was not enriched: %+v", view)
	}
	if len(view.Cycle.Content) != 1 || view.Cycle.Content[0].ID != "brief" {
		t.Fatalf("expected effective campaign content, got %+v", view.Cycle.Content)
	}
	if view.TrackingLink == nil || view.TrackingLink.Token != "right" {
		t.Fatalf("expected talent-owned tracking link, got %+v", view.TrackingLink)
	}
	if view.TrackingLink.EventEndpoint != "/t/right" {
		t.Fatalf("unexpected tracking endpoint %q", view.TrackingLink.EventEndpoint)
	}
	if view.Stats.MyConversions != 4 || view.Stats.CycleTotal != 10 {
		t.Fatalf("unexpected stats: %+v", view.Stats)
	}

	body, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	jsonBody := string(body)
	if !strings.Contains(jsonBody, `"talent_id"`) || strings.Contains(jsonBody, `"Talent_id"`) {
		t.Fatalf("expected lowercase snake_case JSON, got %s", jsonBody)
	}
}

func TestGetMyCycleRejectsUnassignedTalent(t *testing.T) {
	svc := talent.New(testStore(), nil)

	_, err := svc.GetMyCycle(context.Background(), "other-talent", "cycle-1")
	if !errors.Is(err, talent.ErrCycleNotAssigned) {
		t.Fatalf("expected cycle_not_assigned, got %v", err)
	}
}

func TestGetCycleStatsRejectsUnassignedTalent(t *testing.T) {
	svc := talent.New(testStore(), nil)

	_, err := svc.GetCycleStats(context.Background(), "other-talent", "cycle-1")
	if !errors.Is(err, talent.ErrCycleNotAssigned) {
		t.Fatalf("expected cycle_not_assigned, got %v", err)
	}
}
