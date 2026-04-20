package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestChapel_OnPlay_SetsTrashPrompt(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"chapel", "copper", "estate", "estate", "copper"}
	s.Players[0].Actions = 1

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "chapel"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, s.PendingDecision)

	prompt, ok := s.PendingDecision.Prompt.(engine.TrashFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 0, prompt.Min)
	require.Equal(t, 4, prompt.Max)
}

func TestChapel_OnResolve_TrashesCards(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"copper", "estate", "estate", "copper"}

	engine.RequestDecision(s, 0, "chapel", 0, engine.TrashFromHandPrompt{Min: 0, Max: 4}, nil)

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: []engine.CardID{"estate", "estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.Len(t, s.Players[0].Hand, 2)
	require.Len(t, s.Trash, 2)
}

func TestChapel_OnResolve_TrashZero(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"copper"}

	engine.RequestDecision(s, 0, "chapel", 0, engine.TrashFromHandPrompt{Min: 0, Max: 4}, nil)

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: nil},
	}, testLookup)
	require.NoError(t, err)
	require.Len(t, s.Players[0].Hand, 1)
	require.Empty(t, s.Trash)
}
