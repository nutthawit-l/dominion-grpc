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

func TestAddCoins(t *testing.T) {
	s := newTestState(1)
	events := AddCoins(s, 0, 3)
	require.Equal(t, 3, s.Players[0].Coins)
	require.Len(t, events, 1)
	require.Equal(t, EventCoinsAdded, events[0].Kind)
	require.Equal(t, 3, events[0].Count)
}

func TestAddBuys(t *testing.T) {
	s := newTestState(1)
	s.Players[0].Buys = 1
	events := AddBuys(s, 0, 2)
	require.Equal(t, 3, s.Players[0].Buys)
	require.Len(t, events, 1)
	require.Equal(t, EventBuysAdded, events[0].Kind)
}

func TestAddActions(t *testing.T) {
	s := newTestState(1)
	events := AddActions(s, 0, 2)
	require.Equal(t, 2, s.Players[0].Actions)
	require.Len(t, events, 1)
	require.Equal(t, EventActionsAdded, events[0].Kind)
}

func TestGainCard_ToDiscard(t *testing.T) {
	s := newTestState(1)
	s.Supply.Piles["silver"] = 40

	events := GainCard(s, 0, "silver", GainToDiscard)

	require.Equal(t, 39, s.Supply.Piles["silver"])
	require.Equal(t, []CardID{"silver"}, s.Players[0].Discard)
	require.Len(t, events, 1)
	require.Equal(t, EventCardGained, events[0].Kind)
}

func TestGainCard_EmptyPileReturnsNoEvent(t *testing.T) {
	s := newTestState(1)
	s.Supply.Piles["province"] = 0

	events := GainCard(s, 0, "province", GainToDiscard)

	require.Empty(t, s.Players[0].Discard)
	require.Empty(t, events)
	require.Equal(t, 0, s.Supply.Piles["province"])
}

func TestGainCard_ToHand(t *testing.T) {
	s := newTestState(1)
	s.Supply.Piles["gold"] = 30

	events := GainCard(s, 0, "gold", GainToHand)

	require.Equal(t, []CardID{"gold"}, s.Players[0].Hand)
	require.Len(t, events, 1)
}

func TestEachOtherPlayer_IterationOrderAndEventOrdering(t *testing.T) {
	s := newTestState(4)
	var visited []int
	events := EachOtherPlayer(s, s.CurrentPlayer, func(idx int) []Event {
		visited = append(visited, idx)
		return []Event{{Kind: EventCardDrawn, PlayerIdx: idx, Count: 1}}
	})
	// Starting from next seat after CurrentPlayer (1), wrapping: 2, 3, 0.
	require.Equal(t, []int{2, 3, 0}, visited)
	// Events are concatenated in visit order.
	require.Len(t, events, 3)
	require.Equal(t, 2, events[0].PlayerIdx)
	require.Equal(t, 3, events[1].PlayerIdx)
	require.Equal(t, 0, events[2].PlayerIdx)
}

func TestEachOtherPlayer_TwoPlayers(t *testing.T) {
	s := newTestState(2)
	var visited []int
	_ = EachOtherPlayer(s, 0, func(idx int) []Event {
		visited = append(visited, idx)
		return nil
	})
	require.Equal(t, []int{1}, visited)
}

func TestEachOtherPlayer_NilCallbackEvents(t *testing.T) {
	s := newTestState(3)
	events := EachOtherPlayer(s, 0, func(idx int) []Event { return nil })
	require.Nil(t, events)
}
