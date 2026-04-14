package cards

import (
	"github.com/tie/dominion-grpc/internal/engine"
)

func newTestStateForCards(numPlayers int) *engine.GameState {
	players := make([]engine.PlayerState, numPlayers)
	return &engine.GameState{
		GameID:        "test",
		Seed:          1,
		Players:       players,
		CurrentPlayer: 0,
		Phase:         engine.PhaseBuy,
		Supply:        engine.Supply{Piles: map[engine.CardID]int{}},
	}
}
