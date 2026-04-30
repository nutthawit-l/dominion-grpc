package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestCouncilRoom_OnPlay_TwoPlayer(t *testing.T) {
	gs := newTestStateForCards(2)
	gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper", "copper"}
	gs.Players[1].Deck = []engine.CardID{"estate"}
	gs.Players[0].Buys = 1

	events := CouncilRoom.OnPlay(gs, 0)

	require.Len(t, gs.Players[0].Hand, 4)
	require.Equal(t, 2, gs.Players[0].Buys)
	require.Len(t, gs.Players[1].Hand, 1)
	require.Equal(t, engine.CardID("estate"), gs.Players[1].Hand[0])
	// 4 self draws + 1 BuysAdded + 1 other-player draw = 6 total.
	require.Len(t, events, 6)
}

func TestCouncilRoom_OnPlay_ThreePlayer_EachOtherDrawsOne(t *testing.T) {
	gs := newTestStateForCards(3)
	gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper", "copper"}
	gs.Players[1].Deck = []engine.CardID{"estate"}
	gs.Players[2].Deck = []engine.CardID{"silver"}

	_ = CouncilRoom.OnPlay(gs, 0)

	require.Len(t, gs.Players[0].Hand, 4)
	require.Len(t, gs.Players[1].Hand, 1)
	require.Len(t, gs.Players[2].Hand, 1)
}

func TestCouncilRoom_OnPlay_OtherPlayerEmptyDeckStillOK(t *testing.T) {
	gs := newTestStateForCards(2)
	gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper", "copper"}
	// Player 1 has no cards anywhere.

	events := CouncilRoom.OnPlay(gs, 0)

	require.Len(t, gs.Players[0].Hand, 4)
	require.Empty(t, gs.Players[1].Hand)
	// 4 self draws + 1 BuysAdded + 0 other-player draws = 5.
	require.Len(t, events, 5)
}

func TestCouncilRoom_Metadata(t *testing.T) {
	require.Equal(t, "Council Room", CouncilRoom.Name)
	require.Equal(t, 5, CouncilRoom.Cost)
	require.True(t, CouncilRoom.HasType(engine.TypeAction))
}
