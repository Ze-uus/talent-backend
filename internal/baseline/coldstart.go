package baseline

import (
	"context"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/store"
)

// EnsureColdStart seeds talent_baselines from the category median when missing.
// Existing baselines are left unchanged so learned LTEC is preserved.
func EnsureColdStart(ctx context.Context, st store.Store, talent store.Talent, delta_lt float64) error {
	if _, err := st.GetTalentBaseline(ctx, talent.ID); err == nil {
		return nil
	}
	cat, err := st.GetCategoryBaseline(ctx, string(talent.Category))
	if err != nil {
		return err
	}
	b, err := algo.BootstrapBaseline(cat, delta_lt)
	if err != nil {
		return err
	}
	return st.UpsertTalentBaseline(ctx, store.TalentBaseline{
		Talent_id:  talent.ID,
		Alpha_lt:   b.Alpha_lt,
		Beta_lt:    b.Beta_lt,
		Lambda_lt:  b.Lambda_lt,
		Sigma_hist: b.Sigma_hist,
		Delta_lt:   b.Delta_lt,
	})
}
