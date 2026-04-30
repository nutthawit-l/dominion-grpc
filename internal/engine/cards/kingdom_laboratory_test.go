package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestLaboratory_OnPlay_DrawsTwoAndAddsOneAction(t *testing.T) {
	gs := newTestStateForCards(1)
	gs.Players[0].Deck = []engine.CardID{"copper", "silver"}
	gs.Players[0].Actions = 0

	events := Laboratory.OnPlay(gs, 0)

	require.Len(t, gs.Players[0].Hand, 2)
	require.Equal(t, 1, gs.Players[0].Actions)
	require.Len(t, events, 3)
}

func TestLaboratory_OnPlay_EmptyDeck_StillGrantsAction(t *testing.T) {
	gs := newTestStateForCards(1)
	gs.Players[0].Actions = 0

	events := Laboratory.OnPlay(gs, 0)

	require.Equal(t, 1, gs.Players[0].Actions)
	require.Empty(t, gs.Players[0].Hand)
	require.Len(t, events, 1)
}

func TestLaboratory_Metadata(t *testing.T) {
	require.Equal(t, "Laboratory", Laboratory.Name)
	require.Equal(t, 5, Laboratory.Cost)
	require.True(t, Laboratory.HasType(engine.TypeAction))
}
