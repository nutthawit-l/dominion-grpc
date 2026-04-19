package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestCouncilRoom_OnPlay_TwoPlayer(t *testing.T) {
	s := newTestStateForCards(2)
	s.Players[0].Deck = []engine.CardID{"copper", "copper", "copper", "copper"}
	s.Players[1].Deck = []engine.CardID{"estate"}
	s.Players[0].Buys = 1

	events := CouncilRoom.OnPlay(s, 0)

	require.Len(t, s.Players[0].Hand, 4)
	require.Equal(t, 2, s.Players[0].Buys)
	require.Len(t, s.Players[1].Hand, 1)
	require.Equal(t, engine.CardID("estate"), s.Players[1].Hand[0])
	// 4 self draws + 1 BuysAdded + 1 other-player draw = 6 total.
	require.Len(t, events, 6)
}

func TestCouncilRoom_OnPlay_ThreePlayer_EachOtherDrawsOne(t *testing.T) {
	s := newTestStateForCards(3)
	s.Players[0].Deck = []engine.CardID{"copper", "copper", "copper", "copper"}
	s.Players[1].Deck = []engine.CardID{"estate"}
	s.Players[2].Deck = []engine.CardID{"silver"}

	_ = CouncilRoom.OnPlay(s, 0)

	require.Len(t, s.Players[0].Hand, 4)
	require.Len(t, s.Players[1].Hand, 1)
	require.Len(t, s.Players[2].Hand, 1)
}

func TestCouncilRoom_OnPlay_OtherPlayerEmptyDeckStillOK(t *testing.T) {
	s := newTestStateForCards(2)
	s.Players[0].Deck = []engine.CardID{"copper", "copper", "copper", "copper"}
	// Player 1 has no cards anywhere.

	events := CouncilRoom.OnPlay(s, 0)

	require.Len(t, s.Players[0].Hand, 4)
	require.Empty(t, s.Players[1].Hand)
	// 4 self draws + 1 BuysAdded + 0 other-player draws = 5.
	require.Len(t, events, 5)
}

func TestCouncilRoom_Metadata(t *testing.T) {
	require.Equal(t, "Council Room", CouncilRoom.Name)
	require.Equal(t, 5, CouncilRoom.Cost)
	require.True(t, CouncilRoom.HasType(engine.TypeAction))
}
