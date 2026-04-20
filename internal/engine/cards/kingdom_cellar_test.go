package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestCellar_OnPlay_AddsActionAndSetsPrompt(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"cellar", "copper", "estate", "silver"}
	s.Players[0].Deck = []engine.CardID{"gold", "gold", "gold"}
	s.Players[0].Actions = 1

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "cellar"}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 1, s.Players[0].Actions) // +1 action, but 1 consumed = net 1
	require.NotNil(t, s.PendingDecision)

	prompt, ok := s.PendingDecision.Prompt.(engine.DiscardFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 0, prompt.Min)
	require.Equal(t, 3, prompt.Max) // 3 cards left after cellar moved to in-play
}

func TestCellar_OnResolve_DiscardsAndDraws(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"copper", "estate", "silver"}
	s.Players[0].Deck = []engine.CardID{"gold", "gold", "gold"}

	engine.RequestDecision(s, 0, "cellar", 0, engine.DiscardFromHandPrompt{Min: 0, Max: 3}, nil)

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: []engine.CardID{"copper", "estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.Nil(t, s.PendingDecision)
	require.Len(t, s.Players[0].Hand, 3) // silver + 2 drawn
	require.Contains(t, s.Players[0].Hand, engine.CardID("silver"))
}

func TestCellar_OnResolve_DiscardZero(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"copper"}
	s.Players[0].Deck = []engine.CardID{"gold"}

	engine.RequestDecision(s, 0, "cellar", 0, engine.DiscardFromHandPrompt{Min: 0, Max: 1}, nil)

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: nil},
	}, testLookup)
	require.NoError(t, err)
	require.Len(t, s.Players[0].Hand, 1)
}
