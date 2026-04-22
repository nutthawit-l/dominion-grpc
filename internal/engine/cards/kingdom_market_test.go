package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestMarket_OnPlay_GrantsAllFourStats(t *testing.T) {
	gs := newTestStateForCards(1)
	gs.Players[0].Deck = []engine.CardID{"copper"}
	gs.Players[0].Actions = 0
	gs.Players[0].Buys = 1
	gs.Players[0].Coins = 0

	events := Market.OnPlay(gs, 0)

	require.Len(t, gs.Players[0].Hand, 1)
	require.Equal(t, 1, gs.Players[0].Actions)
	require.Equal(t, 2, gs.Players[0].Buys)
	require.Equal(t, 1, gs.Players[0].Coins)
	require.Len(t, events, 4)
}

func TestMarket_Metadata(t *testing.T) {
	require.Equal(t, "Market", Market.Name)
	require.Equal(t, 5, Market.Cost)
	require.True(t, Market.HasType(engine.TypeAction))
}
