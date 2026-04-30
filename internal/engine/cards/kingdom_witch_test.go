package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestWitch_Metadata(t *testing.T) {
	require.Equal(t, "Witch", Witch.Name)
	require.Equal(t, 5, Witch.Cost)
	require.True(t, Witch.HasType(engine.TypeAction))
	require.True(t, Witch.HasType(engine.TypeAttack))
}

func TestWitch_OnPlay_DrawsTwo_EachOpponentGainsCurse(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"witch"}
	gs.Players[0].Deck = []engine.CardID{"copper", "copper"}
	gs.Supply.Piles["curse"] = 10

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "witch"}, testLookup)
	require.NoError(t, err)
	require.Len(t, gs.Players[0].Hand, 2, "attacker drew 2")
	require.Contains(t, gs.Players[1].Discard, engine.CardID("curse"))
	require.Equal(t, 9, gs.Supply.Piles["curse"])
}

func TestWitch_OnPlay_MoatBlocksCurseGain(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"witch"}
	gs.Players[1].Hand = []engine.CardID{"moat"}
	gs.Supply.Piles["curse"] = 10

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "witch"}, testLookup)
	require.NoError(t, err)
	require.NotContains(t, gs.Players[1].Discard, engine.CardID("curse"),
		"Moat-holding opponent must not receive a Curse")
	require.Equal(t, 10, gs.Supply.Piles["curse"])
}

func TestWitch_OnPlay_EmptyCursePile_NoEffect(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"witch"}
	gs.Supply.Piles["curse"] = 0

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "witch"}, testLookup)
	require.NoError(t, err)
	require.NotContains(t, gs.Players[1].Discard, engine.CardID("curse"))
}

func TestWitch_OnPlay_ThreePlayer_BothOpponentsGainCurses(t *testing.T) {
	gs := newTestStateForCards(3)
	gs.Phase = engine.PhaseAction
	gs.CurrentPlayer = 0
	gs.Players[0].Actions = 1
	gs.Players[0].Hand = []engine.CardID{"witch"}
	gs.Supply.Piles["curse"] = 10

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "witch"}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[1].Discard, engine.CardID("curse"))
	require.Contains(t, gs.Players[2].Discard, engine.CardID("curse"))
	require.Equal(t, 8, gs.Supply.Piles["curse"])
}
