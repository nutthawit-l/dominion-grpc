package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDrawCards_FromDeck(t *testing.T) {
	s := newTestState(1)
	fillDeck(s, 0, "copper", 5)

	events := DrawCards(s, 0, 3)

	require.Len(t, s.Players[0].Hand, 3)
	require.Len(t, s.Players[0].Deck, 2)
	require.Len(t, events, 3)
	require.Equal(t, EventCardDrawn, events[0].Kind)
}

func TestDrawCards_ReshufflesDiscardWhenDeckEmpty(t *testing.T) {
	s := newTestState(1)
	s.Players[0].Discard = []CardID{"copper", "estate", "silver"}

	events := DrawCards(s, 0, 2)

	require.Len(t, s.Players[0].Hand, 2)
	require.Len(t, s.Players[0].Discard, 0)
	require.Len(t, s.Players[0].Deck, 1)
	require.Len(t, events, 2)
}

func TestDrawCards_StopsWhenBothEmpty(t *testing.T) {
	s := newTestState(1)
	s.Players[0].Deck = []CardID{"copper"}

	events := DrawCards(s, 0, 5)

	require.Len(t, s.Players[0].Hand, 1)
	require.Len(t, events, 1)
}

func TestDiscardFromHand_MovesCardsToDiscard(t *testing.T) {
	s := newTestState(1)
	s.Players[0].Hand = []CardID{"copper", "estate", "silver"}

	events := DiscardFromHand(s, 0, []CardID{"estate", "copper"})

	require.ElementsMatch(t, []CardID{"silver"}, s.Players[0].Hand)
	require.ElementsMatch(t, []CardID{"estate", "copper"}, s.Players[0].Discard)
	require.Len(t, events, 2)
	require.Equal(t, EventCardDiscarded, events[0].Kind)
}

func TestDiscardFromHand_IgnoresCardsNotInHand(t *testing.T) {
	s := newTestState(1)
	s.Players[0].Hand = []CardID{"copper"}

	events := DiscardFromHand(s, 0, []CardID{"silver"})

	require.ElementsMatch(t, []CardID{"copper"}, s.Players[0].Hand)
	require.Len(t, s.Players[0].Discard, 0)
	require.Len(t, events, 0)
}
