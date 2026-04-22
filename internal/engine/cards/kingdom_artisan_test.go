package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestArtisan_OnPlay_SetsGainPrompt(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"artisan"}
	gs.Players[0].Actions = 1
	gs.Supply.Piles["silver"] = 10

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "artisan"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, 0, gs.PendingDecision.Step)

	prompt, ok := gs.PendingDecision.Prompt.(engine.GainFromSupplyPrompt)
	require.True(t, ok)
	require.Equal(t, 5, prompt.MaxCost)
	require.Equal(t, engine.GainToHand, prompt.Dest)
}

func TestArtisan_Step0_GainToHand_Step1_PutOnDeck(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"copper"}
	gs.Supply.Piles["silver"] = 10

	engine.RequestDecision(gs, 0, "artisan", 0,
		engine.GainFromSupplyPrompt{MaxCost: 5, Dest: engine.GainToHand}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{Card: "silver"},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[0].Hand, engine.CardID("silver"))
	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, 1, gs.PendingDecision.Step)

	_, ok := gs.PendingDecision.Prompt.(engine.PutOnDeckPrompt)
	require.True(t, ok)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{Card: "copper"},
	}, testLookup)
	require.NoError(t, err)
	require.NotContains(t, gs.Players[0].Hand, engine.CardID("copper"))
	require.Equal(t, engine.CardID("copper"), gs.Players[0].Deck[len(gs.Players[0].Deck)-1])
	require.Nil(t, gs.PendingDecision)
}
