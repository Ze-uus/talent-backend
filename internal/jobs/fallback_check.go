package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/Ze-uus/talent-backend/internal/store"
)

// FallbackCheckJob runs hourly and flags conversion events for direct_traffic
// cycles where the 24-hour downstream confirmation window has passed.
// Story 24: fallback_flagged is set here; LockFallbackConversions is called
// by the FinalisePayouts admin action.
type FallbackCheckJob struct {
	st  store.Store
	log *slog.Logger
}

func NewFallbackCheckJob(s store.Store, log *slog.Logger) *FallbackCheckJob {
	return &FallbackCheckJob{st: s, log: log}
}

func (j *FallbackCheckJob) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cycles, err := j.st.GetActiveCycles(ctx)
	if err != nil {
		j.log.Error("fallback_check: get active cycles failed", "err", err)
		return
	}

	now := time.Now().UTC()
	for _, cycle := range cycles {
		if cycle.Campaign_type != store.Type_direct_traffic {
			continue
		}
		// Check each completed day in the cycle. A day is eligible for flagging
		// once 48h have elapsed (cycle day end + 24h confirmation window).
		for day := cycle.Start_date.UTC().Truncate(24 * time.Hour); day.Before(now); day = day.AddDate(0, 0, 1) {
			deadline := day.AddDate(0, 0, 2) // day + 48h
			if now.After(deadline) {
				if err := j.st.FlagFallbackConversions(ctx, cycle.ID, day); err != nil {
					j.log.Error("fallback_check: flag failed",
						"cycle_id", cycle.ID, "day", day.Format("2006-01-02"), "err", err)
				}
			}
		}
	}
}
