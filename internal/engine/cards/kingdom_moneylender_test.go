package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestMoneylender_OnPlay_NoCopperInHand_NoDecision(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"moneylender", "estate", "silver"}
	s.Players[0].Actions = 1

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "moneylender"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, s.PendingDecision)
}

func TestMoneylender_OnPlay_CopperInHand_SetsPrompt(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"moneylender", "copper", "estate"}
	s.Players[0].Actions = 1

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "moneylender"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, s.PendingDecision)

	prompt, ok := s.PendingDecision.Prompt.(engine.TrashFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 0, prompt.Min)
	require.Equal(t, 1, prompt.Max)
	require.Equal(t, []engine.CardID{"copper"}, prompt.CardFilter)
}

func TestMoneylender_OnResolve_TrashCopper_Adds3Coins(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"copper", "estate"}
	s.Players[0].Coins = 0

	engine.RequestDecision(s, 0, "moneylender", 0,
		engine.TrashFromHandPrompt{Min: 0, Max: 1, CardFilter: []engine.CardID{"copper"}}, nil)

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: []engine.CardID{"copper"}},
	}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 3, s.Players[0].Coins)
	require.Contains(t, s.Trash, engine.CardID("copper"))
}

func TestMoneylender_OnResolve_Decline_NoEffect(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"copper"}
	s.Players[0].Coins = 0

	engine.RequestDecision(s, 0, "moneylender", 0,
		engine.TrashFromHandPrompt{Min: 0, Max: 1, CardFilter: []engine.CardID{"copper"}}, nil)

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: nil},
	}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 0, s.Players[0].Coins)
	require.Contains(t, s.Players[0].Hand, engine.CardID("copper"))
}
