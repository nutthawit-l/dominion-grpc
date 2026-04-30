package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestCellar_OnPlay_AddsActionAndSetsPrompt(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"cellar", "copper", "estate", "silver"}
	gs.Players[0].Deck = []engine.CardID{"gold", "gold", "gold"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "cellar"}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 1, gs.Players[0].Actions) // +1 action, but 1 consumed = net 1
	require.NotNil(t, gs.PendingDecision)

	prompt, ok := gs.PendingDecision.Prompt.(engine.DiscardFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 0, prompt.Min)
	require.Equal(t, 3, prompt.Max) // 3 cards left after cellar moved to in-play
}

func TestCellar_OnResolve_DiscardsAndDraws(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"copper", "estate", "silver"}
	gs.Players[0].Deck = []engine.CardID{"gold", "gold", "gold"}

	engine.RequestDecision(gs, 0, "cellar", 0, engine.DiscardFromHandPrompt{Min: 0, Max: 3}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: []engine.CardID{"copper", "estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Len(t, gs.Players[0].Hand, 3) // silver + 2 drawn
	require.Contains(t, gs.Players[0].Hand, engine.CardID("silver"))
}

func TestCellar_OnResolve_DiscardZero(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"copper"}
	gs.Players[0].Deck = []engine.CardID{"gold"}

	engine.RequestDecision(gs, 0, "cellar", 0, engine.DiscardFromHandPrompt{Min: 0, Max: 1}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: nil},
	}, testLookup)
	require.NoError(t, err)
	require.Len(t, gs.Players[0].Hand, 1)
}
