package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/store"
)

// DailyComputeJob runs at 00:00 UTC and performs per-talent-per-cycle computation:
// STEC analysis, breakout detection, and baseline update.
type DailyComputeJob struct {
	st         store.Store
	anomaly_ch chan<- domain.AnomalyEvent
	log        *slog.Logger
}

func NewDailyComputeJob(s store.Store, anomaly_ch chan<- domain.AnomalyEvent, log *slog.Logger) *DailyComputeJob {
	return &DailyComputeJob{st: s, anomaly_ch: anomaly_ch, log: log}
}

func (j *DailyComputeJob) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cycles, err := j.st.GetActiveCycles(ctx)
	if err != nil {
		j.log.Error("daily_compute: get active cycles failed", "err", err)
		return
	}

	for _, cycle := range cycles {
		campaign, err := j.st.GetCampaignByID(ctx, cycle.Campaign_id)
		if err != nil {
			j.log.Error("daily_compute: get campaign failed", "cycle_id", cycle.ID, "err", err)
			continue
		}

		assignments, err := j.st.ListAssignedTalents(ctx, cycle.ID)
		if err != nil {
			j.log.Error("daily_compute: list assignments failed", "cycle_id", cycle.ID, "err", err)
			continue
		}

		all_tiers := algo.ValidTiers(algo.DefaultConfig())

		for _, a := range assignments {
			if a.Status != "active" {
				continue
			}
			j.processTalent(ctx, a, cycle, campaign, all_tiers)
		}
	}
}

func (j *DailyComputeJob) processTalent(
	ctx context.Context,
	a store.TalentAssignment,
	cycle store.Cycle,
	campaign store.Campaign,
	all_tiers []int,
) {
	today_output, err := j.st.GetTalentTodayOutput(ctx, a.Talent_id, time.Now().UTC())
	if err != nil {
		j.log.Warn("daily_compute: get today output failed", "talent_id", a.Talent_id, "err", err)
		today_output = 0
	}

	daily_outputs, err := j.st.GetDailyOutputs(ctx, a.Talent_id, cycle.ID)
	if err != nil || len(daily_outputs) == 0 {
		// Not enough data to run STEC yet; still update baseline
		j.updateBaseline(ctx, a.Talent_id, today_output)
		return
	}

	stec, err := algo.ComputeSTEC(algo.DailyOutputs{
		Talent_id:     a.Talent_id,
		Cycle_id:      cycle.ID,
		Outputs:       daily_outputs,
		PDC_allocated: float64(a.Effective_tier),
	}, cycle.Z_factor)
	if err != nil {
		j.log.Warn("daily_compute: STEC failed", "talent_id", a.Talent_id, "err", err)
		j.updateBaseline(ctx, a.Talent_id, today_output)
		return
	}

	total_conversions, _ := j.st.GetTalentConversions(ctx, a.Talent_id, cycle.ID)
	cycle_days := int(time.Since(cycle.Start_date).Hours()/24) + 1
	if cycle_days < 1 {
		cycle_days = 1
	}

	breakout := algo.DetectBreakout(algo.BreakoutInput{
		Talent_id:         a.Talent_id,
		Cycle_id:          cycle.ID,
		PDC_allocated:     float64(a.Effective_tier),
		Total_conversions: total_conversions,
		Cycle_days:        cycle_days,
		Starting_tier:     a.Effective_tier,
		All_tiers:         all_tiers,
		Slot_target_cost:  campaign.Target_cpa,
		Slot_max_cost:     campaign.Max_cpa,
		ZDR:               stec.ZDR,
		REL:               stec.REL,
		Match_dimensions: algo.MatchDimensions{
			DM: a.Match_dm,
			GP: a.Match_gp,
			OH: a.Match_oh,
		},
	}, stec.Mu_st, algo.DefaultBreakoutThresholds())

	if breakout.Is_breakout {
		ev := domain.AnomalyEvent{
			Talent_id:    a.Talent_id,
			Cycle_id:     cycle.ID,
			Anomaly_type: "breakout",
			Value:        breakout.PDC_override,
			Timestamp:    time.Now().UTC(),
		}
		select {
		case j.anomaly_ch <- ev:
		default:
			j.log.Warn("daily_compute: anomaly_ch full, event dropped", "talent_id", a.Talent_id)
		}
	} else if stec.Pattern == "collapse" {
		ev := domain.AnomalyEvent{
			Talent_id:    a.Talent_id,
			Cycle_id:     cycle.ID,
			Anomaly_type: "collapse",
			Value:        stec.ZDR,
			Timestamp:    time.Now().UTC(),
		}
		select {
		case j.anomaly_ch <- ev:
		default:
			j.log.Warn("daily_compute: anomaly_ch full, collapse event dropped", "talent_id", a.Talent_id)
		}
	}

	j.updateBaseline(ctx, a.Talent_id, today_output)
}

func (j *DailyComputeJob) updateBaseline(ctx context.Context, talent_id string, today_output float64) {
	sb, err := j.st.GetTalentBaseline(ctx, talent_id)
	if err != nil {
		return
	}
	b := algo.TalentBaseline{
		Talent_id:  sb.Talent_id,
		Alpha_lt:   sb.Alpha_lt,
		Beta_lt:    sb.Beta_lt,
		Lambda_lt:  sb.Lambda_lt,
		Sigma_hist: sb.Sigma_hist,
		Delta_lt:   sb.Delta_lt,
	}
	updated, err := algo.UpdateBaseline(b, today_output)
	if err != nil {
		j.log.Warn("daily_compute: UpdateBaseline failed", "talent_id", talent_id, "err", err)
		return
	}
	su := store.TalentBaseline{
		Talent_id:  updated.Talent_id,
		Alpha_lt:   updated.Alpha_lt,
		Beta_lt:    updated.Beta_lt,
		Lambda_lt:  updated.Lambda_lt,
		Sigma_hist: updated.Sigma_hist,
		Delta_lt:   updated.Delta_lt,
	}
	if err := j.st.UpsertTalentBaseline(ctx, su); err != nil {
		j.log.Error("daily_compute: UpsertTalentBaseline failed", "talent_id", talent_id, "err", err)
	}
}
