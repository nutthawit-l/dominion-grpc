package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestWorkshop_OnPlay_SetsGainPrompt(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"workshop"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "workshop"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)

	prompt, ok := gs.PendingDecision.Prompt.(engine.GainFromSupplyPrompt)
	require.True(t, ok)
	require.Equal(t, 4, prompt.MaxCost)
	require.Equal(t, engine.GainToDiscard, prompt.Dest)
}

func TestWorkshop_OnResolve_GainsCard(t *testing.T) {
	gs := newActionPhaseState()
	gs.Supply.Piles["silver"] = 10

	engine.RequestDecision(gs, 0, "workshop", 0,
		engine.GainFromSupplyPrompt{MaxCost: 4, Dest: engine.GainToDiscard}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{Card: "silver"},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[0].Discard, engine.CardID("silver"))
	require.Equal(t, 9, gs.Supply.Piles["silver"])
}
