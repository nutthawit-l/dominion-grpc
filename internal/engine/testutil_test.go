package engine

import (
	"math/rand"
)

// newTestState builds a minimal GameState with the given per-player
// deck/discard/hand sizes. All cards are "copper" for simplicity;
// tests can overwrite fields after construction if they need more.
func newTestState(numPlayers int) *GameState {
	players := make([]PlayerState, numPlayers)
	for i := range players {
		players[i] = PlayerState{Name: "p" + string(rune('0'+i))}
	}
	return &GameState{
		GameID:        "test",
		Seed:          1,
		rng:           rand.New(rand.NewSource(1)),
		Players:       players,
		CurrentPlayer: 0,
		Phase:         PhaseAction,
		Supply:        Supply{Piles: map[CardID]int{}},
	}
}

// fillDeck pushes n copies of card onto player p's deck.
func fillDeck(s *GameState, p int, card CardID, n int) {
	for i := 0; i < n; i++ {
		s.Players[p].Deck = append(s.Players[p].Deck, card)
	}
}
