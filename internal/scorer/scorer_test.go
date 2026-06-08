package scorer

import (
	"testing"

	"tournament-games/internal/model"
)

func intPtr(n int) *int { return &n }

// Formula: base (pointsExact for exact score, pointsOutcome for correct outcome, 0 otherwise)
//        + max(0, pointsExact - |predictedTotal - actualTotal|)
func TestScoreBet(t *testing.T) {
	// Using pointsExact=3, pointsOutcome=1 for all cases.
	// goalBonus = max(0, 3 - diff)
	tests := []struct {
		name          string
		betHome       int
		betAway       int
		actualHome    int
		actualAway    int
		pointsExact   int
		pointsOutcome int
		want          int
		note          string
	}{
		// exact 2-2, actual 2-2: base=3, diff=0, bonus=3 → 6
		{"exact_score", 2, 2, 2, 2, 3, 1, 6, "exact match"},
		// bet 2-1 (home), actual 3-0 (home), same total: base=1, diff=0, bonus=3 → 4
		{"correct_winner_same_total", 2, 1, 3, 0, 3, 1, 4, "correct winner, total equal"},
		// bet 1-1 (draw), actual 0-0 (draw), diff=2: base=1, diff=2, bonus=1 → 2
		{"correct_draw_diff_total", 1, 1, 0, 0, 3, 1, 2, "correct draw, total off by 2"},
		// bet 2-1 (home), actual 0-3 (away), same total: base=0, diff=0, bonus=3 → 3
		{"wrong_outcome_same_total", 2, 1, 0, 3, 3, 1, 3, "wrong outcome, total equal"},
		// bet 3-0 (home), actual 0-0 (draw), diff=3: base=0, diff=3, bonus=0 → 0
		{"wrong_outcome_diff3", 3, 0, 0, 0, 3, 1, 0, "wrong outcome, total off by 3"},
		// bet 1-0 (home), actual 0-6 (away), diff=5: base=0, diff=5, bonus=0 → 0
		{"wrong_outcome_diff5", 1, 0, 0, 6, 3, 1, 0, "wrong outcome, total off by 5"},
		// bet 2-1 (home), actual 2-0 (home), diff=1: base=1, diff=1, bonus=2 → 3
		{"correct_winner_diff1", 2, 1, 2, 0, 3, 1, 3, "correct winner, total off by 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bet := &model.Bet{HomeScore: tt.betHome, AwayScore: tt.betAway}
			fixture := &model.Fixture{GoalsHome: intPtr(tt.actualHome), GoalsAway: intPtr(tt.actualAway)}
			got := ScoreBet(bet, fixture, tt.pointsExact, tt.pointsOutcome)
			if got != tt.want {
				t.Errorf("[%s] ScoreBet(%d-%d vs %d-%d) = %d, want %d",
					tt.note, tt.betHome, tt.betAway, tt.actualHome, tt.actualAway, got, tt.want)
			}
		})
	}
}

func TestScoreBetNilGoals(t *testing.T) {
	bet := &model.Bet{HomeScore: 1, AwayScore: 0}
	fixture := &model.Fixture{GoalsHome: nil, GoalsAway: nil}
	if got := ScoreBet(bet, fixture, 3, 1); got != 0 {
		t.Errorf("expected 0 for nil goals, got %d", got)
	}
}
