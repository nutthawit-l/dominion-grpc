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

func TestSilver_OnPlay_NoMerchant_NoBonus(t *testing.T) {
	gs := newTestStateForCards(1)
	Silver.OnPlay(gs, 0)
	require.Equal(t, 2, gs.Players[0].Coins,
		"without any MerchantBonusCharges, Silver pays only 2")
	require.True(t, gs.Players[0].FirstSilverPlayedThisTurn,
		"Silver must flip the per-turn first-Silver flag")
}

func TestSilver_OnPlay_FirstTime_DrainsCharges(t *testing.T) {
	gs := newTestStateForCards(1)
	gs.Players[0].MerchantBonusCharges = 2
	Silver.OnPlay(gs, 0)
	require.Equal(t, 4, gs.Players[0].Coins,
		"first Silver pays 2 + 2 (Merchant bonus) = 4")
	require.True(t, gs.Players[0].FirstSilverPlayedThisTurn)
	// Charges remain stranded on PlayerState; only flag prevents re-fire.
}

func TestSilver_OnPlay_SecondTimeThisTurn_NoBonus(t *testing.T) {
	gs := newTestStateForCards(1)
	gs.Players[0].FirstSilverPlayedThisTurn = true
	gs.Players[0].MerchantBonusCharges = 5
	Silver.OnPlay(gs, 0)
	require.Equal(t, 2, gs.Players[0].Coins,
		"second Silver this turn must pay only +2, no bonus")
	require.True(t, gs.Players[0].FirstSilverPlayedThisTurn,
		"flag stays true (idempotent)")
}
