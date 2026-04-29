package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestRemodel_OnPlay_SetsTrashPrompt(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"remodel", "copper", "estate"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "remodel"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, 0, gs.PendingDecision.Step)

	prompt, ok := gs.PendingDecision.Prompt.(engine.TrashFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 1, prompt.Min)
	require.Equal(t, 1, prompt.Max)
}

func TestRemodel_OnPlay_EmptyHand_NoDecision(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"remodel"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "remodel"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
}

func TestRemodel_Step0_TrashThenGainPrompt(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"estate", "copper"}
	gs.Supply.Piles["silver"] = 10

	engine.RequestDecision(gs, 0, "remodel", 0,
		engine.TrashFromHandPrompt{Min: 1, Max: 1}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: []engine.CardID{"estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Trash, engine.CardID("estate"))
	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, 1, gs.PendingDecision.Step)

	prompt, ok := gs.PendingDecision.Prompt.(engine.GainFromSupplyPrompt)
	require.True(t, ok)
	require.Equal(t, 4, prompt.MaxCost) // estate costs 2 + 2 = 4
}

func TestRemodel_Step1_GainsCard(t *testing.T) {
	gs := newActionPhaseState()
	gs.Supply.Piles["silver"] = 10

	engine.RequestDecision(gs, 0, "remodel", 1,
		engine.GainFromSupplyPrompt{MaxCost: 4, Dest: engine.GainToDiscard},
		map[engine.ContextKey]any{engine.CtxKeyTrashedCost: 2})

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{Card: "silver"},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[0].Discard, engine.CardID("silver"))
	require.Nil(t, gs.PendingDecision)
}
