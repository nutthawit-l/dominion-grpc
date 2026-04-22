package engine

// IsGameOver returns true if the Province pile is empty or any three
// supply piles are empty.
func IsGameOver(gs *GameState) bool {
	if gs.Supply.Piles["province"] <= 0 {
		return true
	}
	empty := 0
	for _, n := range gs.Supply.Piles {
		if n <= 0 {
			empty++
		}
	}
	return empty >= 3
}

// ComputeScore totals a player's victory points across every zone.
// Uses the supplied lookup to resolve VP values.
func ComputeScore(ps PlayerState, lookup CardLookup) int {
	total := 0
	add := func(zone []CardID) {
		for _, id := range zone {
			c, ok := lookup(id)
			if !ok || c.VictoryPoints == nil {
				continue
			}
			total += c.VictoryPoints(ps)
		}
	}
	add(ps.Hand)
	add(ps.Deck)
	add(ps.Discard)
	add(ps.InPlay)
	return total
}

// DetermineWinners returns the indices of all players tied for the
// highest score.
func DetermineWinners(scores []int) []PlayerIdx {
	best := scores[0]
	for _, s := range scores[1:] {
		if s > best {
			best = s
		}
	}
	var winners []PlayerIdx
	for i, s := range scores {
		if s == best {
			winners = append(winners, PlayerIdx(i))
		}
	}
	return winners
}
