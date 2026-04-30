package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestChapel_OnPlay_SetsTrashPrompt(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"chapel", "copper", "estate", "estate", "copper"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "chapel"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)

	prompt, ok := gs.PendingDecision.Prompt.(engine.TrashFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 0, prompt.Min)
	require.Equal(t, 4, prompt.Max)
}

func TestChapel_OnResolve_TrashesCards(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"copper", "estate", "estate", "copper"}

	engine.RequestDecision(gs, 0, "chapel", 0, engine.TrashFromHandPrompt{Min: 0, Max: 4}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: []engine.CardID{"estate", "estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.Len(t, gs.Players[0].Hand, 2)
	require.Len(t, gs.Trash, 2)
}

func TestChapel_OnResolve_TrashZero(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"copper"}

	engine.RequestDecision(gs, 0, "chapel", 0, engine.TrashFromHandPrompt{Min: 0, Max: 4}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: nil},
	}, testLookup)
	require.NoError(t, err)
	require.Len(t, gs.Players[0].Hand, 1)
	require.Empty(t, gs.Trash)
}
