package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestMoat_Metadata(t *testing.T) {
	require.Equal(t, "Moat", Moat.Name)
	require.Equal(t, 2, Moat.Cost)
	require.True(t, Moat.HasType(engine.TypeAction))
	require.True(t, Moat.HasType(engine.TypeReaction))
}

func TestMoat_OnPlay_DrawsTwo(t *testing.T) {
	gs := newTestStateForCards(1)
	gs.Players[0].Deck = []engine.CardID{"copper", "silver", "gold"}

	events := Moat.OnPlay(gs, 0)

	require.Len(t, gs.Players[0].Hand, 2)
	require.Len(t, events, 2)
}

func TestMoat_OnReaction_AttackTrigger_Blocks(t *testing.T) {
	gs := newTestStateForCards(2)
	tr := engine.Trigger{Kind: engine.TriggerAttackPlayed, Attacker: 0, CardID: "witch"}

	blocks, events := Moat.OnReaction(gs, 1, tr)

	require.True(t, blocks)
	require.Len(t, events, 1)
	require.Equal(t, engine.EventReactionTriggered, events[0].Kind)
	require.Equal(t, engine.PlayerIdx(1), events[0].PlayerIdx)
	require.Equal(t, engine.CardID("moat"), events[0].CardID)
}

func TestMoat_OnReaction_UnknownTrigger_DoesNotBlock(t *testing.T) {
	gs := newTestStateForCards(2)
	tr := engine.Trigger{Kind: engine.TriggerUnknown, Attacker: 0, CardID: "x"}

	blocks, events := Moat.OnReaction(gs, 1, tr)

	require.False(t, blocks)
	require.Empty(t, events)
}
