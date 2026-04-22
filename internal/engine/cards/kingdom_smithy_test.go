package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestSmithy_OnPlay_DrawsThree(t *testing.T) {
	gs := newTestStateForCards(1)
	gs.Players[0].Deck = []engine.CardID{"copper", "silver", "gold"}

	events := Smithy.OnPlay(gs, 0)

	require.Len(t, gs.Players[0].Hand, 3)
	require.Len(t, events, 3)
}

func TestSmithy_OnPlay_PartialDraw_OnShortSupply(t *testing.T) {
	gs := newTestStateForCards(1)
	gs.Players[0].Deck = []engine.CardID{"copper"}
	gs.Players[0].Discard = []engine.CardID{"silver"}

	events := Smithy.OnPlay(gs, 0)

	require.Len(t, gs.Players[0].Hand, 2)
	require.Len(t, events, 2)
}

func TestSmithy_OnPlay_EmptyDeckAndDiscard_DrawsZero(t *testing.T) {
	gs := newTestStateForCards(1)
	events := Smithy.OnPlay(gs, 0)
	require.Empty(t, gs.Players[0].Hand)
	require.Empty(t, events)
}

func TestSmithy_Metadata(t *testing.T) {
	require.Equal(t, "Smithy", Smithy.Name)
	require.Equal(t, 4, Smithy.Cost)
	require.True(t, Smithy.HasType(engine.TypeAction))
}
