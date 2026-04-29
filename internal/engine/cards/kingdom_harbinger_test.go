package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestHarbinger_OnPlay_DrawsAndAddsAction(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"harbinger"}
	gs.Players[0].Deck = []engine.CardID{"copper"}
	gs.Players[0].Discard = []engine.CardID{"silver", "gold"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "harbinger"}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 1, gs.Players[0].Actions)
	require.Contains(t, gs.Players[0].Hand, engine.CardID("copper"))
	require.NotNil(t, gs.PendingDecision)

	prompt, ok := gs.PendingDecision.Prompt.(engine.ChooseFromDiscardPrompt)
	require.True(t, ok)
	require.True(t, prompt.Optional)
	require.Len(t, prompt.Cards, 2)
}

func TestHarbinger_OnPlay_EmptyDiscard_NoDecision(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"harbinger"}
	gs.Players[0].Deck = []engine.CardID{"copper"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "harbinger"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
}

func TestHarbinger_OnResolve_PutsCardOnDeck(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Discard = []engine.CardID{"silver", "gold"}
	gs.Players[0].Deck = []engine.CardID{"copper"}

	engine.RequestDecision(gs, 0, "harbinger", 0,
		engine.ChooseFromDiscardPrompt{Cards: []engine.CardID{"silver", "gold"}, Optional: true}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{Card: "gold"},
	}, testLookup)
	require.NoError(t, err)
	require.Equal(t, []engine.CardID{"silver"}, gs.Players[0].Discard)
	require.Equal(t, engine.CardID("gold"), gs.Players[0].Deck[len(gs.Players[0].Deck)-1])
}

func TestHarbinger_OnResolve_DeclineIsLegal(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Discard = []engine.CardID{"silver"}

	engine.RequestDecision(gs, 0, "harbinger", 0,
		engine.ChooseFromDiscardPrompt{Cards: []engine.CardID{"silver"}, Optional: true}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{None: true},
	}, testLookup)
	require.NoError(t, err)
	require.Equal(t, []engine.CardID{"silver"}, gs.Players[0].Discard)
}
