package engine

import (
	"fmt"
	"math/rand"
)

// CardLookup is how engine code fetches card definitions without
// importing the cards package (which would create an import cycle).
type CardLookup func(CardID) (*Card, bool)

// NewGame builds the initial state for a 2-player Tier 0 Base game.
// Each player gets 7 Coppers + 3 Estates shuffled into deck+hand,
// draws 5, and the supply is filled to Base-set 2-player counts.
func NewGame(gameID string, playerNames []string, seed int64, lookup CardLookup) (*GameState, error) {
	if len(playerNames) != 2 {
		return nil, fmt.Errorf("newgame: expected 2 players, got %d", len(playerNames))
	}
	s := &GameState{
		GameID:        gameID,
		Seed:          seed,
		rng:           rand.New(rand.NewSource(seed)),
		CurrentPlayer: 0,
		Phase:         PhaseAction,
		Turn:          1,
		Supply:        Supply{Piles: map[CardID]int{}},
	}
	s.Players = make([]PlayerState, len(playerNames))
	for i, name := range playerNames {
		s.Players[i] = PlayerState{Name: name}
		// 7 coppers + 3 estates into deck, shuffle, draw 5.
		for k := 0; k < 7; k++ {
			s.Players[i].Deck = append(s.Players[i].Deck, "copper")
		}
		for k := 0; k < 3; k++ {
			s.Players[i].Deck = append(s.Players[i].Deck, "estate")
		}
		shuffleCards(s.rng, s.Players[i].Deck)
	}

	// 2-player supply counts (Base 2nd edition).
	s.Supply.Piles = map[CardID]int{
		"copper":   60 - 7*2,
		"silver":   40,
		"gold":     30,
		"estate":   14 - 3*2,
		"duchy":    8,
		"province": 8,
		"curse":    10,
	}

	// Draw each player's opening hand.
	for i := range s.Players {
		DrawCards(s, i, 5)
	}

	// Pick starting player after all shuffles/draws so existing
	// deterministic shuffle results with a given seed are preserved.
	s.CurrentPlayer = s.rng.Intn(len(playerNames))
	s.StartingPlayer = s.CurrentPlayer

	// First player's Action phase resources.
	s.Players[s.CurrentPlayer].Actions = 1
	s.Players[s.CurrentPlayer].Buys = 1
	s.Players[s.CurrentPlayer].Coins = 0

	// lookup is unused in Tier 0's initial setup but is threaded through
	// so Tier 1+ card-selection logic can use it later.
	_ = lookup
	return s, nil
}
