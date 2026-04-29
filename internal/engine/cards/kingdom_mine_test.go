package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestMine_OnPlay_NoTreasureInHand_NoDecision(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"mine", "estate"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "mine"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
}

func TestMine_OnPlay_TreasureInHand_SetsPrompt(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"mine", "copper", "silver"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "mine"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)

	prompt, ok := gs.PendingDecision.Prompt.(engine.TrashFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 0, prompt.Min)
	require.Equal(t, 1, prompt.Max)
	require.Equal(t, []engine.CardType{engine.TypeTreasure}, prompt.TypeFilter)
}

func TestMine_Step0_TrashCopper_Step1_GainSilverToHand(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"copper"}
	gs.Supply.Piles["silver"] = 10

	engine.RequestDecision(gs, 0, "mine", 0,
		engine.TrashFromHandPrompt{Min: 0, Max: 1, TypeFilter: []engine.CardType{engine.TypeTreasure}}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: []engine.CardID{"copper"}},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Trash, engine.CardID("copper"))
	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, 1, gs.PendingDecision.Step)

	prompt, ok := gs.PendingDecision.Prompt.(engine.GainFromSupplyPrompt)
	require.True(t, ok)
	require.Equal(t, 3, prompt.MaxCost) // copper costs 0 + 3 = 3
	require.Equal(t, engine.GainToHand, prompt.Dest)
	require.Equal(t, []engine.CardType{engine.TypeTreasure}, prompt.TypeFilter)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{Card: "silver"},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[0].Hand, engine.CardID("silver"))
	require.Nil(t, gs.PendingDecision)
}

func TestMine_Step0_Decline_NoEffect(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"copper"}

	engine.RequestDecision(gs, 0, "mine", 0,
		engine.TrashFromHandPrompt{Min: 0, Max: 1, TypeFilter: []engine.CardType{engine.TypeTreasure}}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: nil},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[0].Hand, engine.CardID("copper"))
	require.Nil(t, gs.PendingDecision)
}
