package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeAttack and fakeReaction are helpers for wiring a lookup that the
// tests in this file can control without registering real cards.
type fakeAttack struct{ id CardID }

func fakeLookupWith(cards ...*Card) CardLookup {
	byID := map[CardID]*Card{}
	for _, c := range cards {
		byID[c.ID] = c
	}
	return func(id CardID) (*Card, bool) {
		c, ok := byID[id]
		return c, ok
	}
}

func TestResolveAttackVictims_TwoPlayerNoReactions(t *testing.T) {
	gs := NewTestStateWithRNG("t", 1, []PlayerState{
		{Name: "a"}, {Name: "b", Hand: []CardID{"copper", "estate"}},
	})
	lookup := fakeLookupWith(
		&Card{ID: "copper", Name: "Copper"},
		&Card{ID: "estate", Name: "Estate"},
	)

	victims, events := ResolveAttackVictims(gs, 0, "witch", lookup)

	require.Equal(t, []PlayerIdx{1}, victims)
	require.Len(t, events, 1)
	require.Equal(t, EventAttackPlayed, events[0].Kind)
	require.Equal(t, PlayerIdx(0), events[0].PlayerIdx)
	require.Equal(t, CardID("witch"), events[0].CardID)
}

func TestResolveAttackVictims_FourPlayerIterationOrder(t *testing.T) {
	gs := NewTestStateWithRNG("t", 1, []PlayerState{
		{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"},
	})
	lookup := fakeLookupWith()

	victims, _ := ResolveAttackVictims(gs, 1, "witch", lookup) // attacker = seat 1

	require.Equal(t, []PlayerIdx{2, 3, 0}, victims,
		"iteration must start at (attacker+1) and wrap")
}

func TestResolveAttackVictims_MoatLikeBlockerSkipsVictim(t *testing.T) {
	blocker := &Card{
		ID: "blocker", Name: "Blocker",
		OnReaction: func(gs *GameState, victim PlayerIdx, trigger Trigger) (bool, []Event) {
			return true, []Event{{Kind: EventReactionTriggered, PlayerIdx: victim, CardID: "blocker"}}
		},
	}
	gs := NewTestStateWithRNG("t", 1, []PlayerState{
		{Name: "a"},
		{Name: "b", Hand: []CardID{"blocker", "copper"}},
		{Name: "c"},
	})
	lookup := fakeLookupWith(blocker, &Card{ID: "copper", Name: "Copper"})

	victims, events := ResolveAttackVictims(gs, 0, "witch", lookup)

	require.Equal(t, []PlayerIdx{2}, victims, "blocked victim (seat 1) must be excluded")
	require.Contains(t, events, Event{Kind: EventReactionTriggered, PlayerIdx: 1, CardID: "blocker"})
}

func TestResolveAttackVictims_AllBlocked_EmptyVictimList(t *testing.T) {
	blocker := &Card{
		ID: "blocker", Name: "Blocker",
		OnReaction: func(gs *GameState, victim PlayerIdx, trigger Trigger) (bool, []Event) {
			return true, nil
		},
	}
	gs := NewTestStateWithRNG("t", 1, []PlayerState{
		{Name: "a"},
		{Name: "b", Hand: []CardID{"blocker"}},
	})
	lookup := fakeLookupWith(blocker)

	victims, _ := ResolveAttackVictims(gs, 0, "witch", lookup)
	require.Empty(t, victims)
}

func TestResolveAttackVictims_NonReactionTriggerIgnored(t *testing.T) {
	nonBlocker := &Card{
		ID: "non_blocker", Name: "X",
		OnReaction: func(gs *GameState, victim PlayerIdx, trigger Trigger) (bool, []Event) {
			return false, nil // defines OnReaction but chooses not to block
		},
	}
	gs := NewTestStateWithRNG("t", 1, []PlayerState{
		{Name: "a"},
		{Name: "b", Hand: []CardID{"non_blocker"}},
	})
	lookup := fakeLookupWith(nonBlocker)

	victims, _ := ResolveAttackVictims(gs, 0, "witch", lookup)
	require.Equal(t, []PlayerIdx{1}, victims)
}

// Silence unused-struct warning on fakeAttack since it's part of the shared
// helper surface for upcoming card tests.
var _ = fakeAttack{}
