package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestLaboratory_OnPlay_DrawsTwoAndAddsOneAction(t *testing.T) {
	s := newTestStateForCards(1)
	s.Players[0].Deck = []engine.CardID{"copper", "silver"}
	s.Players[0].Actions = 0

	events := Laboratory.OnPlay(s, 0)

	require.Len(t, s.Players[0].Hand, 2)
	require.Equal(t, 1, s.Players[0].Actions)
	require.Len(t, events, 3)
}

func TestLaboratory_OnPlay_EmptyDeck_StillGrantsAction(t *testing.T) {
	s := newTestStateForCards(1)
	s.Players[0].Actions = 0

	events := Laboratory.OnPlay(s, 0)

	require.Equal(t, 1, s.Players[0].Actions)
	require.Empty(t, s.Players[0].Hand)
	require.Len(t, events, 1)
}

func TestLaboratory_Metadata(t *testing.T) {
	require.Equal(t, "Laboratory", Laboratory.Name)
	require.Equal(t, 5, Laboratory.Cost)
	require.True(t, Laboratory.HasType(engine.TypeAction))
}
