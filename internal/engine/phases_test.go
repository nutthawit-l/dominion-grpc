package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCleanupAndEndTurn_MovesInPlayAndHandToDiscard(t *testing.T) {
	gs := newTestState(2)
	gs.StartingPlayer = 0
	gs.CurrentPlayer = 0
	gs.Phase = PhaseCleanup
	gs.Players[0].Hand = []CardID{"copper", "estate"}
	gs.Players[0].InPlay = []CardID{"silver"}
	// give the next player a drawable deck so DrawCards does not panic
	fillDeck(gs, 1, "copper", 10)
	fillDeck(gs, 0, "copper", 10)

	events := cleanupAndEndTurn(gs)

	require.Len(t, gs.Players[0].Hand, 5)
	require.Len(t, gs.Players[0].InPlay, 0)
	require.Contains(t, gs.Players[0].Discard, CardID("estate"))
	require.Contains(t, gs.Players[0].Discard, CardID("silver"))
	require.Equal(t, PlayerIdx(1), gs.CurrentPlayer)
	require.Equal(t, PhaseAction, gs.Phase)
	require.Equal(t, 1, gs.Players[1].Actions)
	require.Equal(t, 1, gs.Players[1].Buys)
	require.Equal(t, 0, gs.Players[1].Coins)
	require.NotEmpty(t, events)
}

func TestCleanupAndEndTurn_WrapsToFirstPlayerAndIncrementsTurn(t *testing.T) {
	gs := newTestState(2)
	gs.StartingPlayer = 0
	gs.CurrentPlayer = 1
	gs.Phase = PhaseCleanup
	gs.Turn = 1
	fillDeck(gs, 0, "copper", 10)
	fillDeck(gs, 1, "copper", 10)

	cleanupAndEndTurn(gs)

	require.Equal(t, PlayerIdx(0), gs.CurrentPlayer)
	require.Equal(t, 2, gs.Turn)
}

func TestCleanup_FlushesSetAsideToDiscard(t *testing.T) {
	gs := newTestState(2)
	gs.CurrentPlayer = 0
	gs.StartingPlayer = 0
	gs.Phase = PhaseCleanup
	gs.Players[0].SetAside = []CardID{"copper", "estate"}
	gs.Players[0].Deck = []CardID{"silver", "silver", "silver", "silver", "silver"}

	cleanupAndEndTurn(gs)

	require.Empty(t, gs.Players[0].SetAside, "SetAside must be empty after cleanup")
	// Discard now contains the original SetAside cards.
	discardSet := map[CardID]int{}
	for _, c := range gs.Players[0].Discard {
		discardSet[c]++
	}
	require.Equal(t, 1, discardSet["copper"])
	require.Equal(t, 1, discardSet["estate"])
}

func TestCleanup_ResetsMerchantFields(t *testing.T) {
	gs := newTestState(2)
	gs.CurrentPlayer = 0
	gs.StartingPlayer = 0
	gs.Phase = PhaseCleanup
	// Pre-load the per-turn Merchant trigger fields for the active player.
	gs.Players[0].MerchantBonusCharges = 3
	gs.Players[0].FirstSilverPlayedThisTurn = true
	fillDeck(gs, 0, "copper", 5)
	fillDeck(gs, 1, "copper", 5)

	cleanupAndEndTurn(gs)

	require.Equal(t, 0, gs.Players[0].MerchantBonusCharges,
		"MerchantBonusCharges must reset to 0 at end of turn")
	require.False(t, gs.Players[0].FirstSilverPlayedThisTurn,
		"FirstSilverPlayedThisTurn must reset to false at end of turn")
}
