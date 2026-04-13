package algo

import (
	"errors"
	"fmt"
	"math"
)

const Forbidden = math.MaxFloat64

const maxMatrixSize = 10000

type AssignmentInput struct {
	Talents   []TalentProfile `json:"talents"`
	Slots     []BudgetSlot    `json:"slots"`
	All_tiers []int           `json:"all_tiers"`
}

type Assignment struct {
	Talent_id string  `json:"talent_id"`
	Slot_id   string  `json:"slot_id"`
	Cost      float64 `json:"cost"`
}

type AssignmentOutput struct {
	Assignments []Assignment `json:"assignments"`
	Total_cost  float64      `json:"total_cost"`
	Unassigned  []string     `json:"unassigned"`
}

// ComputeCij returns the time cost (days) for talent i assigned to slot j.
// Returns Forbidden if the assignment is ineligible.
func ComputeCij(t TalentProfile, slot BudgetSlot, all_tiers []int) float64 {
	q, err := QualifyTalentForSlot(t, slot, all_tiers)
	if err != nil || !q.Qualified {
		return Forbidden
	}
	bounds, err := TierBoundsFor(slot)
	if err != nil || bounds.Max <= 0 {
		return Forbidden
	}
	shortfall := (float64(bounds.Max) - q.Actual) / float64(bounds.Max)
	if shortfall < 0 {
		shortfall = 0
	}
	penalty_days := math.Floor(shortfall / 0.10)
	result := q.Base_days + penalty_days
	if !isFinite(result) {
		return Forbidden
	}
	return result
}

func Solve(in AssignmentInput) (AssignmentOutput, error) {
	if len(in.Talents) == 0 && len(in.Slots) == 0 {
		return AssignmentOutput{}, nil
	}
	if len(in.Talents) == 0 {
		return AssignmentOutput{}, errors.New("no talents provided")
	}
	if len(in.Slots) == 0 {
		return AssignmentOutput{}, errors.New("no slots provided")
	}
	if len(in.All_tiers) == 0 {
		return AssignmentOutput{}, errors.New("all_tiers must not be empty")
	}

	n := len(in.Talents)
	m := len(in.Slots)
	size := n
	if m > size {
		size = m
	}

	if size > maxMatrixSize {
		return AssignmentOutput{}, fmt.Errorf("matrix size %d exceeds maximum %d", size, maxMatrixSize)
	}

	cost := make([][]float64, size)
	for i := range cost {
		cost[i] = make([]float64, size)
		for j := range cost[i] {
			cost[i][j] = Forbidden
		}
	}
	for i, t := range in.Talents {
		for j, s := range in.Slots {
			cost[i][j] = ComputeCij(t, s, in.All_tiers)
		}
	}

	assignment := hungarianSolve(cost, size)
	out := AssignmentOutput{}
	assigned := make(map[int]bool)

	for i, j := range assignment {
		if i >= n || j >= m {
			continue
		}
		c := cost[i][j]
		if c >= Forbidden {
			out.Unassigned = append(out.Unassigned, in.Talents[i].ID)
			continue
		}
		out.Assignments = append(out.Assignments, Assignment{
			Talent_id: in.Talents[i].ID,
			Slot_id:   in.Slots[j].ID,
			Cost:      c,
		})
		out.Total_cost += c
		assigned[i] = true
	}

	for i := range in.Talents {
		if !assigned[i] {
			found := false
			for _, u := range out.Unassigned {
				if u == in.Talents[i].ID {
					found = true
					break
				}
			}
			if !found {
				out.Unassigned = append(out.Unassigned, in.Talents[i].ID)
			}
		}
	}

	return out, nil
}

func hungarianSolve(cost [][]float64, n int) []int {
	const inf = math.MaxFloat64 / 2
	u := make([]float64, n+1)
	v := make([]float64, n+1)
	p := make([]int, n+1)
	way := make([]int, n+1)

	for i := 1; i <= n; i++ {
		p[0] = i
		j0 := 0
		min_val := make([]float64, n+1)
		used := make([]bool, n+1)
		for j := 0; j <= n; j++ {
			min_val[j] = inf
		}
		for {
			used[j0] = true
			i0 := p[j0]
			delta := inf
			var j1 int
			for j := 1; j <= n; j++ {
				if !used[j] {
					cur := cost[i0-1][j-1] - u[i0] - v[j]
					if cur < min_val[j] {
						min_val[j] = cur
						way[j] = j0
					}
					if min_val[j] < delta {
						delta = min_val[j]
						j1 = j
					}
				}
			}
			for j := 0; j <= n; j++ {
				if used[j] {
					u[p[j]] += delta
					v[j] -= delta
				} else {
					min_val[j] -= delta
				}
			}
			j0 = j1
			if p[j0] == 0 {
				break
			}
		}
		for j0 != 0 {
			p[j0] = p[way[j0]]
			j0 = way[j0]
		}
	}

	result := make([]int, n)
	for j := 1; j <= n; j++ {
		if p[j] != 0 {
			result[p[j]-1] = j - 1
		}
	}
	return result
}
