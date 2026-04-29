package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestMoneylender_OnPlay_NoCopperInHand_NoDecision(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"moneylender", "estate", "silver"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "moneylender"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
}

func TestMoneylender_OnPlay_CopperInHand_SetsPrompt(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"moneylender", "copper", "estate"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "moneylender"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)

	prompt, ok := gs.PendingDecision.Prompt.(engine.TrashFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 0, prompt.Min)
	require.Equal(t, 1, prompt.Max)
	require.Equal(t, []engine.CardID{"copper"}, prompt.CardFilter)
}

func TestMoneylender_OnResolve_TrashCopper_Adds3Coins(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"copper", "estate"}
	gs.Players[0].Coins = 0

	engine.RequestDecision(gs, 0, "moneylender", 0,
		engine.TrashFromHandPrompt{Min: 0, Max: 1, CardFilter: []engine.CardID{"copper"}}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: []engine.CardID{"copper"}},
	}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 3, gs.Players[0].Coins)
	require.Contains(t, gs.Trash, engine.CardID("copper"))
}

func TestMoneylender_OnResolve_Decline_NoEffect(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"copper"}
	gs.Players[0].Coins = 0

	engine.RequestDecision(gs, 0, "moneylender", 0,
		engine.TrashFromHandPrompt{Min: 0, Max: 1, CardFilter: []engine.CardID{"copper"}}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: nil},
	}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 0, gs.Players[0].Coins)
	require.Contains(t, gs.Players[0].Hand, engine.CardID("copper"))
}
