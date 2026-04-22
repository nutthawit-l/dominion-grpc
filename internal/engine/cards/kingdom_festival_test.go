package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestFestival_OnPlay_StatChanges(t *testing.T) {
	gs := newTestStateForCards(1)
	gs.Players[0].Actions = 0
	gs.Players[0].Buys = 1
	gs.Players[0].Coins = 0

	events := Festival.OnPlay(gs, 0)

	require.Equal(t, 2, gs.Players[0].Actions)
	require.Equal(t, 2, gs.Players[0].Buys)
	require.Equal(t, 2, gs.Players[0].Coins)
	require.Len(t, events, 3)
}

func TestFestival_OnPlay_StacksWithExistingResources(t *testing.T) {
	gs := newTestStateForCards(1)
	gs.Players[0].Actions = 3
	gs.Players[0].Buys = 2
	gs.Players[0].Coins = 5

	_ = Festival.OnPlay(gs, 0)

	require.Equal(t, 5, gs.Players[0].Actions)
	require.Equal(t, 3, gs.Players[0].Buys)
	require.Equal(t, 7, gs.Players[0].Coins)
}

func TestFestival_Metadata(t *testing.T) {
	require.Equal(t, "Festival", Festival.Name)
	require.Equal(t, 5, Festival.Cost)
	require.True(t, Festival.HasType(engine.TypeAction))
}
