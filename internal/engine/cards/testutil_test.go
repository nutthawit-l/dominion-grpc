package cards

import (
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func newTestStateForCards(numPlayers int) *engine.GameState {
	return newTestStateForCardsWithRNG(numPlayers, 1)
}

func newTestStateForCardsWithRNG(numPlayers int, seed int64) *engine.GameState {
	players := make([]engine.PlayerState, numPlayers)
	for i := range players {
		players[i] = engine.PlayerState{Name: "p" + string(rune('0'+i))}
	}
	gs := engine.NewTestStateWithRNG("test", seed, players)
	gs.Phase = engine.PhaseBuy
	return gs
}

// newActionPhaseState creates a test state where player 0 is in the Action
// phase with 1 action, for testing card OnPlay.
func newActionPhaseState() *engine.GameState {
	gs := newTestStateForCards(2)
	gs.Phase = engine.PhaseAction
	gs.CurrentPlayer = 0
	gs.Players[0].Actions = 1
	gs.Players[0].Buys = 1
	return gs
}

// testLookup returns a CardLookup that finds cards in the DefaultRegistry.
func testLookup(id engine.CardID) (*engine.Card, bool) {
	return DefaultRegistry.Lookup(id)
}
