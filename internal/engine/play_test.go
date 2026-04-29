package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlayCardInPlace_RunsOnPlay_WithoutZoneMove(t *testing.T) {
	gs := newTestState(2)
	gs.Players[0].InPlay = []CardID{"smithy"}
	gs.Players[0].Deck = []CardID{"copper", "estate", "silver"}

	smithy := &Card{
		ID: "smithy", Name: "Smithy", Cost: 4,
		Types: []CardType{TypeAction},
		OnPlay: func(gs *GameState, px PlayerIdx) []Event {
			return DrawCards(gs, px, 3)
		},
	}
	lookup := func(id CardID) (*Card, bool) {
		if id == "smithy" {
			return smithy, true
		}
		return nil, false
	}

	events, err := PlayCardInPlace(gs, 0, "smithy", lookup)

	require.NoError(t, err)
	require.Len(t, gs.Players[0].Hand, 3, "Smithy should draw 3")
	require.Equal(t, []CardID{"smithy"}, gs.Players[0].InPlay,
		"PlayCardInPlace must not duplicate the card in InPlay")
	// First event must be EventCardPlayed for the card.
	require.Equal(t, EventCardPlayed, events[0].Kind)
	require.Equal(t, CardID("smithy"), events[0].CardID)
}

func TestPlayCardInPlace_UnknownCard_ReturnsError(t *testing.T) {
	gs := newTestState(2)
	lookup := func(id CardID) (*Card, bool) { return nil, false }

	_, err := PlayCardInPlace(gs, 0, "ghost", lookup)
	require.ErrorIs(t, err, ErrUnknownCard)
}

func TestPlayCardInPlace_NoOnPlay_OnlyEmitsPlayed(t *testing.T) {
	gs := newTestState(2)
	gs.Players[0].InPlay = []CardID{"vp_only"}
	card := &Card{ID: "vp_only", Name: "VPOnly", Cost: 0}
	lookup := func(id CardID) (*Card, bool) {
		if id == "vp_only" {
			return card, true
		}
		return nil, false
	}

	events, err := PlayCardInPlace(gs, 0, "vp_only", lookup)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, EventCardPlayed, events[0].Kind)
}
