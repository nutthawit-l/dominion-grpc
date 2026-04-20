package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestArtisan_OnPlay_SetsGainPrompt(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"artisan"}
	s.Players[0].Actions = 1
	s.Supply.Piles["silver"] = 10

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "artisan"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, s.PendingDecision)
	require.Equal(t, 0, s.PendingDecision.Step)

	prompt, ok := s.PendingDecision.Prompt.(engine.GainFromSupplyPrompt)
	require.True(t, ok)
	require.Equal(t, 5, prompt.MaxCost)
	require.Equal(t, engine.GainToHand, prompt.Dest)
}

func TestArtisan_Step0_GainToHand_Step1_PutOnDeck(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"copper"}
	s.Supply.Piles["silver"] = 10

	engine.RequestDecision(s, 0, "artisan", 0,
		engine.GainFromSupplyPrompt{MaxCost: 5, Dest: engine.GainToHand}, nil)

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{Card: "silver"},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, s.Players[0].Hand, engine.CardID("silver"))
	require.NotNil(t, s.PendingDecision)
	require.Equal(t, 1, s.PendingDecision.Step)

	_, ok := s.PendingDecision.Prompt.(engine.PutOnDeckPrompt)
	require.True(t, ok)

	_, _, err = engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{Card: "copper"},
	}, testLookup)
	require.NoError(t, err)
	require.NotContains(t, s.Players[0].Hand, engine.CardID("copper"))
	require.Equal(t, engine.CardID("copper"), s.Players[0].Deck[len(s.Players[0].Deck)-1])
	require.Nil(t, s.PendingDecision)
}
