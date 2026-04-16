package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCleanupAndEndTurn_MovesInPlayAndHandToDiscard(t *testing.T) {
	s := newTestState(2)
	s.StartingPlayer = 0
	s.CurrentPlayer = 0
	s.Phase = PhaseCleanup
	s.Players[0].Hand = []CardID{"copper", "estate"}
	s.Players[0].InPlay = []CardID{"silver"}
	// give the next player a drawable deck so DrawCards does not panic
	fillDeck(s, 1, "copper", 10)
	fillDeck(s, 0, "copper", 10)

	events := cleanupAndEndTurn(s)

	require.Len(t, s.Players[0].Hand, 5)
	require.Len(t, s.Players[0].InPlay, 0)
	require.Contains(t, s.Players[0].Discard, CardID("estate"))
	require.Contains(t, s.Players[0].Discard, CardID("silver"))
	require.Equal(t, 1, s.CurrentPlayer)
	require.Equal(t, PhaseAction, s.Phase)
	require.Equal(t, 1, s.Players[1].Actions)
	require.Equal(t, 1, s.Players[1].Buys)
	require.Equal(t, 0, s.Players[1].Coins)
	require.NotEmpty(t, events)
}

func TestCleanupAndEndTurn_WrapsToFirstPlayerAndIncrementsTurn(t *testing.T) {
	s := newTestState(2)
	s.StartingPlayer = 0
	s.CurrentPlayer = 1
	s.Phase = PhaseCleanup
	s.Turn = 1
	fillDeck(s, 0, "copper", 10)
	fillDeck(s, 1, "copper", 10)

	cleanupAndEndTurn(s)

	require.Equal(t, 0, s.CurrentPlayer)
	require.Equal(t, 2, s.Turn)
}
