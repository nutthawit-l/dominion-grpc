package engine

import (
	"fmt"
	"math/rand"
)

// CardLookup is how engine code fetches card definitions without
// importing the cards package (which would create an import cycle).
type CardLookup func(CardID) (*Card, bool)

// KingdomLister is an optional interface: if a CardLookup also
// satisfies this signature (via an attached lister), resolveKingdom
// can enumerate the registry. Production code (cards.DefaultRegistry)
// supplies one via NewLookupAndLister; test helpers supply one too.
type KingdomLister func() []*Card

var kingdomLister KingdomLister

// RegisterKingdomLister installs the function used by NewGame to
// enumerate all kingdom cards when an empty kingdom list is passed.
// Call this once from the cards package init(), and once per test
// that uses a custom registry.
func RegisterKingdomLister(f KingdomLister) {
	kingdomLister = f
}

// NewGame builds the initial state for a 2-player Base game.
//
// `kingdom` selects which kingdom cards appear in the supply. If empty
// or nil, every kingdom card discoverable through `lookup` (every card
// tagged TypeAction) is added with 10 copies. Duplicate IDs collapse
// into a single pile. Unknown IDs return an error.
func NewGame(gameID string, playerNames []string, kingdom []CardID, seed int64, lookup CardLookup) (*GameState, error) {
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

	kingdomIDs, err := resolveKingdom(kingdom, lookup)
	if err != nil {
		return nil, err
	}
	for _, id := range kingdomIDs {
		s.Supply.Piles[id] = 10
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

	return s, nil
}

// resolveKingdom returns the deduplicated list of kingdom card IDs to
// place in the supply. When `requested` is empty, it returns every
// kingdom card reachable through `lookup`.
func resolveKingdom(requested []CardID, lookup CardLookup) ([]CardID, error) {
	if len(requested) == 0 {
		return allKingdomCards(lookup), nil
	}
	seen := map[CardID]struct{}{}
	out := make([]CardID, 0, len(requested))
	for _, id := range requested {
		c, ok := lookup(id)
		if !ok {
			return nil, fmt.Errorf("newgame: unknown card %q", id)
		}
		if !c.IsKingdom() {
			return nil, fmt.Errorf("newgame: %q is not a kingdom card", id)
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func allKingdomCards(lookup CardLookup) []CardID {
	return discoverKingdom(lookup)
}

func discoverKingdom(lookup CardLookup) []CardID {
	if kingdomLister == nil {
		return nil
	}
	var ids []CardID
	for _, c := range kingdomLister() {
		if c.IsKingdom() {
			ids = append(ids, c.ID)
		}
	}
	return ids
}
