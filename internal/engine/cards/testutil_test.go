package cards

import (
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func newTestStateForCards(numPlayers int) *engine.GameState {
	players := make([]engine.PlayerState, numPlayers)
	for i := range players {
		players[i] = engine.PlayerState{Name: "p" + string(rune('0'+i))}
	}
	s := engine.NewTestStateWithRNG("test", 1, players)
	s.Phase = engine.PhaseBuy
	return s
}

// newActionPhaseState creates a test state where player 0 is in the Action
// phase with 1 action, for testing card OnPlay.
func newActionPhaseState() *engine.GameState {
	s := newTestStateForCards(2)
	s.Phase = engine.PhaseAction
	s.CurrentPlayer = 0
	s.Players[0].Actions = 1
	s.Players[0].Buys = 1
	return s
}

// testLookup returns a CardLookup that finds cards in the DefaultRegistry.
func testLookup(id engine.CardID) (*engine.Card, bool) {
	return DefaultRegistry.Lookup(id)
}
