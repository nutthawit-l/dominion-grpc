package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// totalCards returns the sum of all cards in the game across every
// zone. It should be invariant under any legal action.
func totalCards(s *GameState) int {
	total := len(s.Trash)
	for _, p := range s.Players {
		total += len(p.Hand) + len(p.Deck) + len(p.Discard) + len(p.InPlay)
	}
	for _, n := range s.Supply.Piles {
		total += n
	}
	return total
}

// randomLegalAction picks a simple legal action for the current player.
// It prefers playing treasures in buy phase, buying a silver if possible,
// and otherwise ending the phase.
func randomLegalAction(s *GameState) Action {
	me := s.CurrentPlayer
	switch s.Phase {
	case PhaseAction:
		return EndPhase{PlayerIdx: me}
	case PhaseBuy:
		for _, c := range s.Players[me].Hand {
			if c == "copper" || c == "silver" || c == "gold" {
				return PlayCard{PlayerIdx: me, Card: c}
			}
		}
		if s.Players[me].Buys <= 0 {
			return EndPhase{PlayerIdx: me}
		}
		if s.Players[me].Coins >= 8 && s.Supply.Piles["province"] > 0 {
			return BuyCard{PlayerIdx: me, Card: "province"}
		}
		if s.Players[me].Coins >= 6 && s.Supply.Piles["gold"] > 0 {
			return BuyCard{PlayerIdx: me, Card: "gold"}
		}
		if s.Players[me].Coins >= 3 && s.Supply.Piles["silver"] > 0 {
			return BuyCard{PlayerIdx: me, Card: "silver"}
		}
		return EndPhase{PlayerIdx: me}
	}
	return EndPhase{PlayerIdx: me}
}

func TestProperty_CardConservation(t *testing.T) {
	if testing.Short() {
		t.Skip("property sweep is skipped in short mode")
	}
	const seeds = 500
	const maxSteps = 5000
	for i := int64(0); i < seeds; i++ {
		s, err := NewGame("prop", []string{"A", "B"}, i, basicsLookup2)
		require.NoError(t, err)
		start := totalCards(s)

		for step := 0; step < maxSteps && !s.Ended; step++ {
			_, _, err := Apply(s, randomLegalAction(s), basicsLookup2)
			require.NoErrorf(t, err, "seed=%d step=%d", i, step)
		}
		require.Truef(t, s.Ended, "seed=%d did not terminate within %d steps", i, maxSteps)
		require.Equalf(t, start, totalCards(s), "seed=%d card count drifted", i)
	}
}
