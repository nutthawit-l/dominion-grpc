package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestCopper_OnPlay_AddsOneCoin(t *testing.T) {
	gs := newTestStateForCards(1)
	events := Copper.OnPlay(gs, 0)
	require.Equal(t, 1, gs.Players[0].Coins)
	require.Len(t, events, 1)
}

func TestSilver_OnPlay_AddsTwoCoins(t *testing.T) {
	gs := newTestStateForCards(1)
	events := Silver.OnPlay(gs, 0)
	require.Equal(t, 2, gs.Players[0].Coins)
	require.Len(t, events, 1)
}

func TestGold_OnPlay_AddsThreeCoins(t *testing.T) {
	gs := newTestStateForCards(1)
	events := Gold.OnPlay(gs, 0)
	require.Equal(t, 3, gs.Players[0].Coins)
	require.Len(t, events, 1)
}

func TestTreasures_HaveCorrectMetadata(t *testing.T) {
	cases := []struct {
		card     *engine.Card
		wantName string
		wantCost int
	}{
		{Copper, "Copper", 0},
		{Silver, "Silver", 3},
		{Gold, "Gold", 6},
	}
	for _, tc := range cases {
		require.Equal(t, tc.wantName, tc.card.Name)
		require.Equal(t, tc.wantCost, tc.card.Cost)
		require.True(t, tc.card.HasType(engine.TypeTreasure))
	}
}
