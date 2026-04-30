package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// totalCards returns the sum of all cards in the game across every
// zone. It should be invariant under any legal action.
func totalCards(gs *GameState) int {
	total := len(gs.Trash)
	for _, p := range gs.Players {
		total += len(p.Hand) + len(p.Deck) + len(p.Discard) + len(p.InPlay) + len(p.SetAside)
	}
	for _, n := range gs.Supply.Piles {
		total += n
	}
	return total
}

// randomLegalAction picks a simple legal action for the current player.
// It prefers playing treasures in buy phase, buying a silver if possible,
// and otherwise ending the phase.
func randomLegalAction(gs *GameState) Action {
	me := gs.CurrentPlayer
	switch gs.Phase {
	case PhaseAction:
		return EndPhase{PlayerIdx: me}
	case PhaseBuy:
		for _, c := range gs.Players[me].Hand {
			if c == "copper" || c == "silver" || c == "gold" {
				return PlayCard{PlayerIdx: me, Card: c}
			}
		}
		if gs.Players[me].Buys <= 0 {
			return EndPhase{PlayerIdx: me}
		}
		if gs.Players[me].Coins >= 8 && gs.Supply.Piles["province"] > 0 {
			return BuyCard{PlayerIdx: me, Card: "province"}
		}
		if gs.Players[me].Coins >= 6 && gs.Supply.Piles["gold"] > 0 {
			return BuyCard{PlayerIdx: me, Card: "gold"}
		}
		if gs.Players[me].Coins >= 3 && gs.Supply.Piles["silver"] > 0 {
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
		gs, err := NewGame("prop", []string{"A", "B"}, nil, i, basicsLookup2)
		require.NoError(t, err)
		start := totalCards(gs)

		for step := 0; step < maxSteps && !gs.Ended; step++ {
			_, _, err := Apply(gs, randomLegalAction(gs), basicsLookup2)
			require.NoErrorf(t, err, "seed=%d step=%d", i, step)
		}
		require.Truef(t, gs.Ended, "seed=%d did not terminate within %d steps", i, maxSteps)
		require.Equalf(t, start, totalCards(gs), "seed=%d card count drifted", i)
	}
}
