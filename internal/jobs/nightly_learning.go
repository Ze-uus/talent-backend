package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/store"
)

// NightlyLearningJob runs at 01:00 UTC and updates each active talent's
// long-term Bayesian baseline using the 30-day output window.
type NightlyLearningJob struct {
	st       store.Store
	delta_lt float64
	log      *slog.Logger
}

func NewNightlyLearningJob(s store.Store, delta_lt float64, log *slog.Logger) *NightlyLearningJob {
	return &NightlyLearningJob{st: s, delta_lt: delta_lt, log: log}
}

func (j *NightlyLearningJob) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	talents, err := j.st.ListAllTalents(ctx)
	if err != nil {
		j.log.Error("nightly_learning: list talents failed", "err", err)
		return
	}

	for _, t := range talents {
		if t.Status != store.Status_active {
			continue
		}

		sb, err := j.st.GetTalentBaseline(ctx, t.ID)
		if err != nil {
			continue
		}

		outputs, err := j.st.GetTalentOutputWindow(ctx, t.ID, 30)
		if err != nil || len(outputs) == 0 {
			continue
		}

		b := algo.TalentBaseline{
			Talent_id:  sb.Talent_id,
			Alpha_lt:   sb.Alpha_lt,
			Beta_lt:    sb.Beta_lt,
			Lambda_lt:  sb.Lambda_lt,
			Sigma_hist: sb.Sigma_hist,
			Delta_lt:   sb.Delta_lt,
		}
		updated, err := algo.UpdateSigmaHist(b, outputs)
		if err != nil {
			j.log.Warn("nightly_learning: UpdateSigmaHist failed", "talent_id", t.ID, "err", err)
			continue
		}
		updated.Delta_lt = j.delta_lt

		su := store.TalentBaseline{
			Talent_id:  updated.Talent_id,
			Alpha_lt:   updated.Alpha_lt,
			Beta_lt:    updated.Beta_lt,
			Lambda_lt:  updated.Lambda_lt,
			Sigma_hist: updated.Sigma_hist,
			Delta_lt:   updated.Delta_lt,
		}
		if err := j.st.UpsertTalentBaseline(ctx, su); err != nil {
			j.log.Error("nightly_learning: UpsertTalentBaseline failed", "talent_id", t.ID, "err", err)
		}
	}
}
