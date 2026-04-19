package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestFestival_OnPlay_StatChanges(t *testing.T) {
	s := newTestStateForCards(1)
	s.Players[0].Actions = 0
	s.Players[0].Buys = 1
	s.Players[0].Coins = 0

	events := Festival.OnPlay(s, 0)

	require.Equal(t, 2, s.Players[0].Actions)
	require.Equal(t, 2, s.Players[0].Buys)
	require.Equal(t, 2, s.Players[0].Coins)
	require.Len(t, events, 3)
}

func TestFestival_OnPlay_StacksWithExistingResources(t *testing.T) {
	s := newTestStateForCards(1)
	s.Players[0].Actions = 3
	s.Players[0].Buys = 2
	s.Players[0].Coins = 5

	_ = Festival.OnPlay(s, 0)

	require.Equal(t, 5, s.Players[0].Actions)
	require.Equal(t, 3, s.Players[0].Buys)
	require.Equal(t, 7, s.Players[0].Coins)
}

func TestFestival_Metadata(t *testing.T) {
	require.Equal(t, "Festival", Festival.Name)
	require.Equal(t, 5, Festival.Cost)
	require.True(t, Festival.HasType(engine.TypeAction))
}
