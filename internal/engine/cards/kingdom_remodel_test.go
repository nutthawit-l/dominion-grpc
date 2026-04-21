package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestRemodel_OnPlay_SetsTrashPrompt(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"remodel", "copper", "estate"}
	s.Players[0].Actions = 1

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "remodel"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, s.PendingDecision)
	require.Equal(t, 0, s.PendingDecision.Step)

	prompt, ok := s.PendingDecision.Prompt.(engine.TrashFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 1, prompt.Min)
	require.Equal(t, 1, prompt.Max)
}

func TestRemodel_OnPlay_EmptyHand_NoDecision(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"remodel"}
	s.Players[0].Actions = 1

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "remodel"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, s.PendingDecision)
}

func TestRemodel_Step0_TrashThenGainPrompt(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"estate", "copper"}
	s.Supply.Piles["silver"] = 10

	engine.RequestDecision(s, 0, "remodel", 0,
		engine.TrashFromHandPrompt{Min: 1, Max: 1}, nil)

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: []engine.CardID{"estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, s.Trash, engine.CardID("estate"))
	require.NotNil(t, s.PendingDecision)
	require.Equal(t, 1, s.PendingDecision.Step)

	prompt, ok := s.PendingDecision.Prompt.(engine.GainFromSupplyPrompt)
	require.True(t, ok)
	require.Equal(t, 4, prompt.MaxCost) // estate costs 2 + 2 = 4
}

func TestRemodel_Step1_GainsCard(t *testing.T) {
	s := newActionPhaseState()
	s.Supply.Piles["silver"] = 10

	engine.RequestDecision(s, 0, "remodel", 1,
		engine.GainFromSupplyPrompt{MaxCost: 4, Dest: engine.GainToDiscard},
		map[engine.ContextKey]any{engine.CtxKeyTrashedCost: 2})

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{Card: "silver"},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, s.Players[0].Discard, engine.CardID("silver"))
	require.Nil(t, s.PendingDecision)
}
