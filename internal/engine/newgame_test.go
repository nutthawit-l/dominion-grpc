package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewGame_TwoPlayer_InitialDeckAndHand(t *testing.T) {
	names := []string{"Alice", "Bob"}
	lookup := func(id CardID) (*Card, bool) { return basicsLookup[id], true }

	s, err := NewGame("game-1", names, nil, 42, lookup)
	require.NoError(t, err)

	require.Len(t, s.Players, 2)
	for i, p := range s.Players {
		require.Equalf(t, 5, len(p.Hand), "player %d should start with hand of 5", i)
		require.Equalf(t, 5, len(p.Deck), "player %d should have 5 left in deck", i)
		// 7 Coppers + 3 Estates = 10 total.
		require.Equal(t, 10, len(p.Hand)+len(p.Deck))
		countIn := func(zone []CardID, id CardID) int {
			n := 0
			for _, c := range zone {
				if c == id {
					n++
				}
			}
			return n
		}
		total := func(id CardID) int {
			return countIn(p.Hand, id) + countIn(p.Deck, id)
		}
		require.Equal(t, 7, total("copper"))
		require.Equal(t, 3, total("estate"))
	}
	cp := s.CurrentPlayer
	require.Equal(t, PhaseAction, s.Phase)
	require.Contains(t, []int{0, 1}, s.CurrentPlayer)
	require.Equal(t, 1, s.Turn)
	require.Equal(t, 1, s.Players[cp].Actions)
	require.Equal(t, 1, s.Players[cp].Buys)
}

func TestNewGame_TwoPlayer_SupplyCounts(t *testing.T) {
	lookup := func(id CardID) (*Card, bool) { return basicsLookup[id], true }
	s, err := NewGame("game-2", []string{"A", "B"}, nil, 1, lookup)
	require.NoError(t, err)

	// 2-player Base set counts:
	require.Equal(t, 46, s.Supply.Piles["copper"]) // 60 - 7*2
	require.Equal(t, 40, s.Supply.Piles["silver"])
	require.Equal(t, 30, s.Supply.Piles["gold"])
	require.Equal(t, 8, s.Supply.Piles["estate"]) // 14 - 3*2 = 8
	require.Equal(t, 8, s.Supply.Piles["duchy"])
	require.Equal(t, 8, s.Supply.Piles["province"])
	require.Equal(t, 10, s.Supply.Piles["curse"])
}

func TestIsKingdom_ActionCardsQualify(t *testing.T) {
	action := &Card{ID: "x", Types: []CardType{TypeAction}}
	treasure := &Card{ID: "y", Types: []CardType{TypeTreasure}}
	victory := &Card{ID: "z", Types: []CardType{TypeVictory}}
	curse := &Card{ID: "w", Types: []CardType{TypeCurse}}
	actionAttack := &Card{ID: "v", Types: []CardType{TypeAction, TypeAttack}}
	actionReaction := &Card{ID: "u", Types: []CardType{TypeAction, TypeReaction}}

	require.True(t, action.IsKingdom())
	require.False(t, treasure.IsKingdom())
	require.False(t, victory.IsKingdom())
	require.False(t, curse.IsKingdom())
	require.True(t, actionAttack.IsKingdom())
	require.True(t, actionReaction.IsKingdom())
}

func TestNewGame_EmptyKingdom_UsesAllRegisteredActions(t *testing.T) {
	action := &Card{ID: "alpha", Name: "Alpha", Cost: 3, Types: []CardType{TypeAction},
		OnPlay: func(*GameState, int) []Event { return nil }}
	action2 := &Card{ID: "beta", Name: "Beta", Cost: 4, Types: []CardType{TypeAction},
		OnPlay: func(*GameState, int) []Event { return nil }}
	basics := basicCards()
	lookup := combineLookups(basics, []*Card{action, action2})

	s, err := NewGame("g", []string{"p0", "p1"}, nil, 42, lookup)
	require.NoError(t, err)
	require.Equal(t, 10, s.Supply.Piles["alpha"])
	require.Equal(t, 10, s.Supply.Piles["beta"])
}

func TestNewGame_ExplicitKingdom_OnlyRequestedCards(t *testing.T) {
	action := &Card{ID: "alpha", Name: "Alpha", Cost: 3, Types: []CardType{TypeAction},
		OnPlay: func(*GameState, int) []Event { return nil }}
	action2 := &Card{ID: "beta", Name: "Beta", Cost: 4, Types: []CardType{TypeAction},
		OnPlay: func(*GameState, int) []Event { return nil }}
	lookup := combineLookups(basicCards(), []*Card{action, action2})

	s, err := NewGame("g", []string{"p0", "p1"}, []CardID{"alpha"}, 42, lookup)
	require.NoError(t, err)
	require.Equal(t, 10, s.Supply.Piles["alpha"])
	_, hasBeta := s.Supply.Piles["beta"]
	require.False(t, hasBeta, "beta should not be in supply when not requested")
}

func TestNewGame_UnknownKingdomCard_ReturnsError(t *testing.T) {
	lookup := combineLookups(basicCards(), nil)
	_, err := NewGame("g", []string{"p0", "p1"}, []CardID{"mystery"}, 42, lookup)
	require.Error(t, err)
}

func TestNewGame_DuplicateKingdomCard_SinglePile(t *testing.T) {
	action := &Card{ID: "alpha", Name: "Alpha", Cost: 3, Types: []CardType{TypeAction},
		OnPlay: func(*GameState, int) []Event { return nil }}
	lookup := combineLookups(basicCards(), []*Card{action})

	s, err := NewGame("g", []string{"p0", "p1"}, []CardID{"alpha", "alpha"}, 42, lookup)
	require.NoError(t, err)
	require.Equal(t, 10, s.Supply.Piles["alpha"])
}

// Minimal card table used only for these tests. The cards package
// registry is used in integration tests; here we want a dep-free table.
var basicsLookup = map[CardID]*Card{
	"copper":   {ID: "copper", Name: "Copper", Cost: 0, Types: []CardType{TypeTreasure}},
	"silver":   {ID: "silver", Name: "Silver", Cost: 3, Types: []CardType{TypeTreasure}},
	"gold":     {ID: "gold", Name: "Gold", Cost: 6, Types: []CardType{TypeTreasure}},
	"estate":   {ID: "estate", Name: "Estate", Cost: 2, Types: []CardType{TypeVictory}},
	"duchy":    {ID: "duchy", Name: "Duchy", Cost: 5, Types: []CardType{TypeVictory}},
	"province": {ID: "province", Name: "Province", Cost: 8, Types: []CardType{TypeVictory}},
	"curse":    {ID: "curse", Name: "Curse", Cost: 0, Types: []CardType{TypeCurse}},
}

func basicCards() []*Card {
	return []*Card{
		{ID: "copper", Name: "Copper", Cost: 0, Types: []CardType{TypeTreasure}},
		{ID: "silver", Name: "Silver", Cost: 3, Types: []CardType{TypeTreasure}},
		{ID: "gold", Name: "Gold", Cost: 6, Types: []CardType{TypeTreasure}},
		{ID: "estate", Name: "Estate", Cost: 2, Types: []CardType{TypeVictory}},
		{ID: "duchy", Name: "Duchy", Cost: 5, Types: []CardType{TypeVictory}},
		{ID: "province", Name: "Province", Cost: 8, Types: []CardType{TypeVictory}},
		{ID: "curse", Name: "Curse", Cost: 0, Types: []CardType{TypeCurse}},
	}
}

func combineLookups(base, extra []*Card) CardLookup {
	all := map[CardID]*Card{}
	for _, c := range base {
		all[c.ID] = c
	}
	for _, c := range extra {
		all[c.ID] = c
	}
	RegisterKingdomLister(func() []*Card {
		out := make([]*Card, 0, len(all))
		for _, c := range all {
			out = append(out, c)
		}
		return out
	})
	return func(id CardID) (*Card, bool) {
		c, ok := all[id]
		return c, ok
	}
}
