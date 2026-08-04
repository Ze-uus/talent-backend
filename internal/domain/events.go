package domain

import "time"

// ConversionShard carries pre-computed aggregates so clients can patch UI instantly.
type ConversionShard struct {
	Cycle_total  float64 `json:"cycle_total,omitempty"`
	Talent_total float64 `json:"talent_total,omitempty"`
}

// ConversionEvent is pushed by the tracking service after a successful DB write.
type ConversionEvent struct {
	Update_type string           `json:"update_type"` // always "created"
	Talent_id   string           `json:"talent_id"`
	Cycle_id    string           `json:"cycle_id"`
	Campaign_id string           `json:"campaign_id"`
	Event_type  string           `json:"event_type"`
	KPB_type    string           `json:"kpb_type,omitempty"`
	Pipeline_type string         `json:"pipeline_type,omitempty"`
	Occurred_at time.Time        `json:"occurred_at"`
	Shard       *ConversionShard `json:"shard,omitempty"`
}

// AnomalyShard is the delta payload for anomaly alerts.
type AnomalyShard struct {
	Anomaly_type string  `json:"anomaly_type"` // "collapse" | "breakout"
	Value        float64 `json:"value"`
}

// AnomalyEvent is pushed by the daily compute job on collapse/breakout detection.
type AnomalyEvent struct {
	Talent_id string       `json:"talent_id"`
	Cycle_id  string       `json:"cycle_id"`
	Timestamp time.Time    `json:"timestamp"`
	Shard     AnomalyShard `json:"shard"`
}

// CycleUpdateShard carries cycle fields that changed.
type CycleUpdateShard struct {
	Status           string  `json:"status,omitempty"`
	Remaining_budget float64 `json:"remaining_budget,omitempty"`
}

// CycleUpdateEvent is pushed when cycle state changes.
type CycleUpdateEvent struct {
	Cycle_id    string           `json:"cycle_id"`
	Campaign_id string           `json:"campaign_id"`
	Update_type string           `json:"update_type"`
	Shard       CycleUpdateShard `json:"shard"`
}

// TalentUpdateShard carries talent/assignment fields that changed.
type TalentUpdateShard struct {
	Status         string `json:"status,omitempty"`
	Category       string `json:"category,omitempty"`
	Cycle_id       string `json:"cycle_id,omitempty"`
	Effective_tier int    `json:"effective_tier,omitempty"`
	Slot_id        string `json:"slot_id,omitempty"`
	Tracking_token string `json:"tracking_token,omitempty"`
}

// TalentUpdateEvent is pushed when talent profile or assignment changes.
type TalentUpdateEvent struct {
	Talent_id   string            `json:"talent_id"`
	Cycle_id    string            `json:"cycle_id,omitempty"`
	Update_type string            `json:"update_type"`
	Shard       TalentUpdateShard `json:"shard"`
}

// AuditEvent is pushed to admin-live clients after each append-only audit record.
type AuditEvent struct {
	ID          string `json:"id"`
	Actor_id    string `json:"actor_id,omitempty"`
	Action_type string `json:"action_type"`
	Entity_type string `json:"entity_type"`
	Entity_id   string `json:"entity_id"`
	Request_id  string `json:"request_id,omitempty"`
	Seq         int64  `json:"seq"`
	Prev_hash   string `json:"prev_hash"`
	Entry_hash  string `json:"entry_hash"`
	Signature   string `json:"signature"`
	Archive_uri string `json:"archive_uri,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Before      any    `json:"before,omitempty"`
	After       any    `json:"after,omitempty"`
}
