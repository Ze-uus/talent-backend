package domain

import "time"

// AnomalyEvent is pushed to the anomaly channel by the daily compute job
// and broadcast to admin WebSocket clients for real-time dashboards.
type AnomalyEvent struct {
	Talent_id    string
	Cycle_id     string
	Anomaly_type string // "collapse" | "breakout"
	Value        float64
	Timestamp    time.Time
}

// ConversionEvent is a lightweight projection of store.ConversionEvent
// pushed to the conversion channel by the tracking handler and broadcast
// to admin and talent WebSocket clients.
type ConversionEvent struct {
	Talent_id   string
	Cycle_id    string
	Campaign_id string
	Event_type  string
	Occurred_at time.Time
}
