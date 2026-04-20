package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestWorkshop_OnPlay_SetsGainPrompt(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"workshop"}
	s.Players[0].Actions = 1

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "workshop"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, s.PendingDecision)

	prompt, ok := s.PendingDecision.Prompt.(engine.GainFromSupplyPrompt)
	require.True(t, ok)
	require.Equal(t, 4, prompt.MaxCost)
	require.Equal(t, engine.GainToDiscard, prompt.Dest)
}

func TestWorkshop_OnResolve_GainsCard(t *testing.T) {
	s := newActionPhaseState()
	s.Supply.Piles["silver"] = 10

	engine.RequestDecision(s, 0, "workshop", 0,
		engine.GainFromSupplyPrompt{MaxCost: 4, Dest: engine.GainToDiscard}, nil)

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{Card: "silver"},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, s.Players[0].Discard, engine.CardID("silver"))
	require.Equal(t, 9, s.Supply.Piles["silver"])
}
