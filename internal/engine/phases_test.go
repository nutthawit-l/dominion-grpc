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
	require.Equal(t, 1, gs.CurrentPlayer)
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

	require.Equal(t, 0, gs.CurrentPlayer)
	require.Equal(t, 2, gs.Turn)
}
