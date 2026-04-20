package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestHarbinger_OnPlay_DrawsAndAddsAction(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"harbinger"}
	s.Players[0].Deck = []engine.CardID{"copper"}
	s.Players[0].Discard = []engine.CardID{"silver", "gold"}
	s.Players[0].Actions = 1

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "harbinger"}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 1, s.Players[0].Actions)
	require.Contains(t, s.Players[0].Hand, engine.CardID("copper"))
	require.NotNil(t, s.PendingDecision)

	prompt, ok := s.PendingDecision.Prompt.(engine.ChooseFromDiscardPrompt)
	require.True(t, ok)
	require.True(t, prompt.Optional)
	require.Len(t, prompt.Cards, 2)
}

func TestHarbinger_OnPlay_EmptyDiscard_NoDecision(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"harbinger"}
	s.Players[0].Deck = []engine.CardID{"copper"}
	s.Players[0].Actions = 1

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "harbinger"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, s.PendingDecision)
}

func TestHarbinger_OnResolve_PutsCardOnDeck(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Discard = []engine.CardID{"silver", "gold"}
	s.Players[0].Deck = []engine.CardID{"copper"}

	engine.RequestDecision(s, 0, "harbinger", 0,
		engine.ChooseFromDiscardPrompt{Cards: []engine.CardID{"silver", "gold"}, Optional: true}, nil)

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{Card: "gold"},
	}, testLookup)
	require.NoError(t, err)
	require.Equal(t, []engine.CardID{"silver"}, s.Players[0].Discard)
	require.Equal(t, engine.CardID("gold"), s.Players[0].Deck[len(s.Players[0].Deck)-1])
}

func TestHarbinger_OnResolve_DeclineIsLegal(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Discard = []engine.CardID{"silver"}

	engine.RequestDecision(s, 0, "harbinger", 0,
		engine.ChooseFromDiscardPrompt{Cards: []engine.CardID{"silver"}, Optional: true}, nil)

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardChoiceAnswer{None: true},
	}, testLookup)
	require.NoError(t, err)
	require.Equal(t, []engine.CardID{"silver"}, s.Players[0].Discard)
}
