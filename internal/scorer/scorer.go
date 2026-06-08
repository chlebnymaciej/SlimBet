package scorer

import "tournament-games/internal/model"

func ScoreBet(bet *model.Bet, fixture *model.Fixture, pointsExact, pointsOutcome int) int {
	if fixture.GoalsHome == nil || fixture.GoalsAway == nil {
		return 0
	}

	actualHome := *fixture.GoalsHome
	actualAway := *fixture.GoalsAway

	base := 0
	switch {
	case bet.HomeScore == actualHome && bet.AwayScore == actualAway:
		base += pointsExact
	case outcome(bet.HomeScore, bet.AwayScore) == outcome(actualHome, actualAway):
		base += pointsOutcome
	}

	predictedTotal := bet.HomeScore + bet.AwayScore
	actualTotal := actualHome + actualAway
	diff := predictedTotal - actualTotal
	if diff < 0 {
		diff = -diff
	}
	goalBonus := pointsExact - diff
	if goalBonus < 0 {
		goalBonus = 0
	}

	return base + goalBonus
}

func outcome(home, away int) string {
	switch {
	case home > away:
		return "home"
	case away > home:
		return "away"
	default:
		return "draw"
	}
}
