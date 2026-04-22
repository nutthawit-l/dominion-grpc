package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDrawCards_FromDeck(t *testing.T) {
	gs := newTestState(1)
	fillDeck(gs, 0, "copper", 5)

	events := DrawCards(gs, 0, 3)

	require.Len(t, gs.Players[0].Hand, 3)
	require.Len(t, gs.Players[0].Deck, 2)
	require.Len(t, events, 3)
	require.Equal(t, EventCardDrawn, events[0].Kind)
}

func TestDrawCards_ReshufflesDiscardWhenDeckEmpty(t *testing.T) {
	gs := newTestState(1)
	gs.Players[0].Discard = []CardID{"copper", "estate", "silver"}

	events := DrawCards(gs, 0, 2)

	require.Len(t, gs.Players[0].Hand, 2)
	require.Len(t, gs.Players[0].Discard, 0)
	require.Len(t, gs.Players[0].Deck, 1)
	require.Len(t, events, 2)
}

func TestDrawCards_StopsWhenBothEmpty(t *testing.T) {
	gs := newTestState(1)
	gs.Players[0].Deck = []CardID{"copper"}

	events := DrawCards(gs, 0, 5)

	require.Len(t, gs.Players[0].Hand, 1)
	require.Len(t, events, 1)
}

func TestDiscardFromHand_MovesCardsToDiscard(t *testing.T) {
	gs := newTestState(1)
	gs.Players[0].Hand = []CardID{"copper", "estate", "silver"}

	events := DiscardFromHand(gs, 0, []CardID{"estate", "copper"})

	require.ElementsMatch(t, []CardID{"silver"}, gs.Players[0].Hand)
	require.ElementsMatch(t, []CardID{"estate", "copper"}, gs.Players[0].Discard)
	require.Len(t, events, 2)
	require.Equal(t, EventCardDiscarded, events[0].Kind)
}

func TestDiscardFromHand_IgnoresCardsNotInHand(t *testing.T) {
	gs := newTestState(1)
	gs.Players[0].Hand = []CardID{"copper"}

	events := DiscardFromHand(gs, 0, []CardID{"silver"})

	require.ElementsMatch(t, []CardID{"copper"}, gs.Players[0].Hand)
	require.Len(t, gs.Players[0].Discard, 0)
	require.Len(t, events, 0)
}

func TestAddCoins(t *testing.T) {
	gs := newTestState(1)
	events := AddCoins(gs, 0, 3)
	require.Equal(t, 3, gs.Players[0].Coins)
	require.Len(t, events, 1)
	require.Equal(t, EventCoinsAdded, events[0].Kind)
	require.Equal(t, 3, events[0].Count)
}

func TestAddBuys(t *testing.T) {
	gs := newTestState(1)
	gs.Players[0].Buys = 1
	events := AddBuys(gs, 0, 2)
	require.Equal(t, 3, gs.Players[0].Buys)
	require.Len(t, events, 1)
	require.Equal(t, EventBuysAdded, events[0].Kind)
}

func TestAddActions(t *testing.T) {
	gs := newTestState(1)
	events := AddActions(gs, 0, 2)
	require.Equal(t, 2, gs.Players[0].Actions)
	require.Len(t, events, 1)
	require.Equal(t, EventActionsAdded, events[0].Kind)
}

func TestGainCard_ToDiscard(t *testing.T) {
	gs := newTestState(1)
	gs.Supply.Piles["silver"] = 40

	events := GainCard(gs, 0, "silver", GainToDiscard)

	require.Equal(t, 39, gs.Supply.Piles["silver"])
	require.Equal(t, []CardID{"silver"}, gs.Players[0].Discard)
	require.Len(t, events, 1)
	require.Equal(t, EventCardGained, events[0].Kind)
}

func TestGainCard_EmptyPileReturnsNoEvent(t *testing.T) {
	gs := newTestState(1)
	gs.Supply.Piles["province"] = 0

	events := GainCard(gs, 0, "province", GainToDiscard)

	require.Empty(t, gs.Players[0].Discard)
	require.Empty(t, events)
	require.Equal(t, 0, gs.Supply.Piles["province"])
}

func TestGainCard_ToHand(t *testing.T) {
	gs := newTestState(1)
	gs.Supply.Piles["gold"] = 30

	events := GainCard(gs, 0, "gold", GainToHand)

	require.Equal(t, []CardID{"gold"}, gs.Players[0].Hand)
	require.Len(t, events, 1)
}

func TestEachOtherPlayer_IterationOrderAndEventOrdering(t *testing.T) {
	gs := newTestState(4)
	var visited []PlayerIdx
	events := EachOtherPlayer(gs, gs.CurrentPlayer, func(idx PlayerIdx) []Event {
		visited = append(visited, idx)
		return []Event{{Kind: EventCardDrawn, PlayerIdx: idx, Count: 1}}
	})
	// Starting from next seat after CurrentPlayer (1), wrapping: 2, 3, 0.
	require.Equal(t, []PlayerIdx{1, 2, 3}, visited)
	// Events are concatenated in visit order.
	require.Len(t, events, 3)
	require.Equal(t, PlayerIdx(1), events[0].PlayerIdx)
	require.Equal(t, PlayerIdx(2), events[1].PlayerIdx)
	require.Equal(t, PlayerIdx(3), events[2].PlayerIdx)
}

func TestEachOtherPlayer_TwoPlayers(t *testing.T) {
	gs := newTestState(2)
	var visited []PlayerIdx
	_ = EachOtherPlayer(gs, gs.CurrentPlayer, func(idx PlayerIdx) []Event {
		visited = append(visited, idx)
		return nil
	})
	require.Equal(t, []PlayerIdx{1}, visited)
}

func TestEachOtherPlayer_NilCallbackEvents(t *testing.T) {
	gs := newTestState(3)
	events := EachOtherPlayer(gs, gs.CurrentPlayer, func(idx PlayerIdx) []Event { return nil })
	require.Nil(t, events)
}

func TestTrashFromHand_MovesCardsToTrash(t *testing.T) {
	gs := newTestState(2)
	gs.Players[0].Hand = []CardID{"estate", "copper", "estate"}

	events := TrashFromHand(gs, 0, []CardID{"estate", "copper"})

	require.Equal(t, []CardID{"estate"}, gs.Players[0].Hand)
	require.Contains(t, gs.Trash, CardID("estate"))
	require.Contains(t, gs.Trash, CardID("copper"))
	require.Len(t, events, 2)
	require.Equal(t, EventCardTrashed, events[0].Kind)
}

func TestTrashFromHand_SkipsMissingCards(t *testing.T) {
	gs := newTestState(2)
	gs.Players[0].Hand = []CardID{"copper"}

	events := TrashFromHand(gs, 0, []CardID{"silver"})

	require.Equal(t, []CardID{"copper"}, gs.Players[0].Hand)
	require.Empty(t, gs.Trash)
	require.Empty(t, events)
}

func TestPutOnDeck_MovesFromHandToTopOfDeck(t *testing.T) {
	gs := newTestState(2)
	gs.Players[0].Hand = []CardID{"copper", "silver", "gold"}
	gs.Players[0].Deck = []CardID{"estate"}

	events := PutOnDeck(gs, 0, []CardID{"silver"})

	require.Equal(t, []CardID{"copper", "gold"}, gs.Players[0].Hand)
	require.Equal(t, []CardID{"estate", "silver"}, gs.Players[0].Deck)
	require.Len(t, events, 1)
	require.Equal(t, EventCardPutOnDeck, events[0].Kind)
}

func TestPutOnDeck_SkipsMissingCards(t *testing.T) {
	gs := newTestState(2)
	gs.Players[0].Hand = []CardID{"copper"}

	events := PutOnDeck(gs, 0, []CardID{"silver"})

	require.Equal(t, []CardID{"copper"}, gs.Players[0].Hand)
	require.Empty(t, events)
}

func TestRevealAndDiscardFromDeck_MovesToDiscard(t *testing.T) {
	gs := newTestState(2)
	gs.Players[0].Deck = []CardID{"estate", "copper", "silver"}

	revealed, events := RevealAndDiscardFromDeck(gs, 0, 1)

	require.Equal(t, []CardID{"silver"}, revealed)
	require.Contains(t, gs.Players[0].Discard, CardID("silver"))
	require.Equal(t, []CardID{"estate", "copper"}, gs.Players[0].Deck)
	require.Len(t, events, 1)
	require.Equal(t, EventCardDiscarded, events[0].Kind)
}

func TestRevealAndDiscardFromDeck_EmptyDeckShufflesDiscard(t *testing.T) {
	gs := newTestState(2)
	gs.Players[0].Deck = nil
	gs.Players[0].Discard = []CardID{"gold", "silver"}

	revealed, events := RevealAndDiscardFromDeck(gs, 0, 1)

	require.Len(t, revealed, 1)
	require.Len(t, events, 1)
}

func TestRevealAndDiscardFromDeck_EmptyBoth_ReturnsEmpty(t *testing.T) {
	gs := newTestState(2)

	revealed, events := RevealAndDiscardFromDeck(gs, 0, 1)

	require.Empty(t, revealed)
	require.Empty(t, events)
}

func TestPlayCardFromZone_Hand(t *testing.T) {
	gs := newTestState(2)
	gs.Players[0].Hand = []CardID{"smithy"}
	called := false
	lookup := func(id CardID) (*Card, bool) {
		if id == "smithy" {
			return &Card{
				ID: "smithy", Types: []CardType{TypeAction},
				OnPlay: func(gs *GameState, px PlayerIdx) []Event {
					called = true
					return []Event{{Kind: EventCardDrawn, PlayerIdx: px, Count: 3}}
				},
			}, true
		}
		return nil, false
	}

	events, err := PlayCardFromZone(gs, 0, "smithy", ZoneHand, lookup)

	require.NoError(t, err)
	require.True(t, called)
	require.Empty(t, gs.Players[0].Hand)
	require.Contains(t, gs.Players[0].InPlay, CardID("smithy"))
	require.GreaterOrEqual(t, len(events), 1)
	require.Equal(t, EventCardPlayed, events[0].Kind)
}

func TestPlayCardFromZone_Discard(t *testing.T) {
	gs := newTestState(2)
	gs.Players[0].Discard = []CardID{"village"}
	lookup := func(id CardID) (*Card, bool) {
		if id == "village" {
			return &Card{
				ID: "village", Types: []CardType{TypeAction},
				OnPlay: func(gs *GameState, px PlayerIdx) []Event { return nil },
			}, true
		}
		return nil, false
	}

	events, err := PlayCardFromZone(gs, 0, "village", ZoneDiscard, lookup)

	require.NoError(t, err)
	require.Empty(t, gs.Players[0].Discard)
	require.Contains(t, gs.Players[0].InPlay, CardID("village"))
	require.Equal(t, EventCardPlayed, events[0].Kind)
}

func TestPlayCardFromZone_CardNotInZone_Error(t *testing.T) {
	gs := newTestState(2)
	lookup := func(id CardID) (*Card, bool) {
		return &Card{ID: "smithy"}, true
	}

	_, err := PlayCardFromZone(gs, 0, "smithy", ZoneHand, lookup)
	require.ErrorIs(t, err, ErrCardNotInHand)

	_, err = PlayCardFromZone(gs, 0, "smithy", ZoneDiscard, lookup)
	require.ErrorIs(t, err, ErrCardNotInDiscard)
}
