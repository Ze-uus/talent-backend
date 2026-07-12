package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Ze-uus/talent-backend/internal/domain"
)

func registerTestClient(t *testing.T, hub *Hub, cycleID, talentID string, isAdmin bool) *client {
	t.Helper()
	c := &client{
		hub:       hub,
		send:      make(chan []byte, 8),
		cycle_id:  cycleID,
		talent_id: talentID,
		is_admin:  isAdmin,
	}
	hub.register <- c
	time.Sleep(10 * time.Millisecond)
	return c
}

func readWSMessage(t *testing.T, ch <-chan []byte) WSMessage {
	t.Helper()
	select {
	case raw := <-ch:
		var msg WSMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unmarshal ws message: %v", err)
		}
		return msg
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ws message")
		return WSMessage{}
	}
}

func readOptional(t *testing.T, ch <-chan []byte) bool {
	t.Helper()
	select {
	case <-ch:
		return true
	case <-time.After(50 * time.Millisecond):
		return false
	}
}

func TestHub_RoutesConversionToAdminAndTalent(t *testing.T) {
	anomaly_ch := make(chan domain.AnomalyEvent, 4)
	conversion_ch := make(chan domain.ConversionEvent, 4)
	cycle_ch := make(chan domain.CycleUpdateEvent, 4)
	talent_ch := make(chan domain.TalentUpdateEvent, 4)

	hub := NewHub(anomaly_ch, conversion_ch, cycle_ch, talent_ch, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	admin := registerTestClient(t, hub, "cycle-1", "", true)
	talent := registerTestClient(t, hub, "cycle-1", "talent-1", false)
	other := registerTestClient(t, hub, "cycle-1", "talent-2", false)

	conversion_ch <- domain.ConversionEvent{
		Update_type: "created",
		Talent_id:   "talent-1",
		Cycle_id:    "cycle-1",
		Event_type:  "click",
		Occurred_at: time.Now().UTC(),
	}

	adminMsg := readWSMessage(t, admin.send)
	if adminMsg.Type != msg_conversion {
		t.Fatalf("admin expected conversion, got %q", adminMsg.Type)
	}

	talentMsg := readWSMessage(t, talent.send)
	if talentMsg.Type != msg_conversion {
		t.Fatalf("talent expected conversion, got %q", talentMsg.Type)
	}

	if readOptional(t, other.send) {
		t.Fatal("other talent should not receive conversion event")
	}
}

func TestHub_RoutesAnomalyToAdminOnly(t *testing.T) {
	anomaly_ch := make(chan domain.AnomalyEvent, 4)
	conversion_ch := make(chan domain.ConversionEvent, 4)
	cycle_ch := make(chan domain.CycleUpdateEvent, 4)
	talent_ch := make(chan domain.TalentUpdateEvent, 4)

	hub := NewHub(anomaly_ch, conversion_ch, cycle_ch, talent_ch, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	admin := registerTestClient(t, hub, "cycle-1", "", true)
	talent := registerTestClient(t, hub, "cycle-1", "talent-1", false)

	anomaly_ch <- domain.AnomalyEvent{
		Talent_id: "talent-1",
		Cycle_id:  "cycle-1",
		Timestamp: time.Now().UTC(),
		Shard:     domain.AnomalyShard{Anomaly_type: "collapse", Value: 0.4},
	}

	msg := readWSMessage(t, admin.send)
	if msg.Type != msg_anomaly {
		t.Fatalf("admin expected anomaly, got %q", msg.Type)
	}
	if readOptional(t, talent.send) {
		t.Fatal("talent should not receive anomaly event")
	}
}

func TestHub_RoutesCycleUpdateToAdminOnly(t *testing.T) {
	anomaly_ch := make(chan domain.AnomalyEvent, 4)
	conversion_ch := make(chan domain.ConversionEvent, 4)
	cycle_ch := make(chan domain.CycleUpdateEvent, 4)
	talent_ch := make(chan domain.TalentUpdateEvent, 4)

	hub := NewHub(anomaly_ch, conversion_ch, cycle_ch, talent_ch, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	admin := registerTestClient(t, hub, "cycle-1", "", true)
	talent := registerTestClient(t, hub, "cycle-1", "talent-1", false)

	cycle_ch <- domain.CycleUpdateEvent{
		Cycle_id:    "cycle-1",
		Campaign_id: "camp-1",
		Update_type: "closed",
		Shard:       domain.CycleUpdateShard{Status: "closed"},
	}

	msg := readWSMessage(t, admin.send)
	if msg.Type != msg_cycle_update {
		t.Fatalf("admin expected cycle_update, got %q", msg.Type)
	}
	if readOptional(t, talent.send) {
		t.Fatal("talent should not receive cycle_update event")
	}
}

func TestHub_RoutesTalentUpdateWithoutCycleToAllTalentConnections(t *testing.T) {
	anomaly_ch := make(chan domain.AnomalyEvent, 4)
	conversion_ch := make(chan domain.ConversionEvent, 4)
	cycle_ch := make(chan domain.CycleUpdateEvent, 4)
	talent_ch := make(chan domain.TalentUpdateEvent, 4)

	hub := NewHub(anomaly_ch, conversion_ch, cycle_ch, talent_ch, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	registerTestClient(t, hub, "cycle-1", "talent-1", false)
	conn2 := registerTestClient(t, hub, "cycle-2", "talent-1", false)

	talent_ch <- domain.TalentUpdateEvent{
		Talent_id:   "talent-1",
		Update_type: "approved",
		Shard:       domain.TalentUpdateShard{Status: "active"},
	}

	msg1 := readWSMessage(t, conn2.send)
	if msg1.Type != msg_talent_update {
		t.Fatalf("expected talent_update, got %q", msg1.Type)
	}
}
