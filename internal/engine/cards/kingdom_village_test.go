package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestVillage_OnPlay_DrawsOneAndAddsTwoActions(t *testing.T) {
	s := newTestStateForCards(1)
	s.Players[0].Deck = []engine.CardID{"copper"}

	events := Village.OnPlay(s, 0)

	require.Equal(t, 2, s.Players[0].Actions)
	require.Len(t, s.Players[0].Hand, 1)
	require.Equal(t, engine.CardID("copper"), s.Players[0].Hand[0])
	require.Len(t, events, 2)
}

func TestVillage_OnPlay_EmptyDeckAndDiscard_StillGrantsActions(t *testing.T) {
	s := newTestStateForCards(1)
	events := Village.OnPlay(s, 0)

	require.Equal(t, 2, s.Players[0].Actions, "actions always granted even when draw fails")
	require.Empty(t, s.Players[0].Hand)
	require.Len(t, events, 1)
}

func TestVillage_Metadata(t *testing.T) {
	require.Equal(t, "Village", Village.Name)
	require.Equal(t, 3, Village.Cost)
	require.True(t, Village.HasType(engine.TypeAction))
}
