package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestVassal_OnPlay_DiscardsTopCard_NonAction_NoDecision(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"vassal"}
	s.Players[0].Deck = []engine.CardID{"copper"}
	s.Players[0].Actions = 1

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "vassal"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, s.PendingDecision)
	require.Contains(t, s.Players[0].Discard, engine.CardID("copper"))
}

func TestVassal_OnPlay_DiscardsAction_AsksToPlay(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"vassal"}
	s.Players[0].Deck = []engine.CardID{"village"}
	s.Players[0].Actions = 1

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "vassal"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, s.PendingDecision)

	prompt, ok := s.PendingDecision.Prompt.(engine.MayPlayActionPrompt)
	require.True(t, ok)
	require.Equal(t, engine.CardID("village"), prompt.Card)
}

func TestVassal_OnResolve_Yes_PlaysFromDiscard(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Discard = []engine.CardID{"village"}
	s.Players[0].Deck = []engine.CardID{"gold", "gold"}

	engine.RequestDecision(s, 0, "vassal", 0,
		engine.MayPlayActionPrompt{Card: "village"},
		map[string]any{"card": engine.CardID("village")})

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.YesNoAnswer{Yes: true},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, s.Players[0].InPlay, engine.CardID("village"))
	require.Empty(t, s.Players[0].Discard)
	require.Len(t, s.Players[0].Hand, 1) // village drew 1
}

func TestVassal_OnResolve_No_DoesNothing(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Discard = []engine.CardID{"village"}

	engine.RequestDecision(s, 0, "vassal", 0,
		engine.MayPlayActionPrompt{Card: "village"},
		map[string]any{"card": engine.CardID("village")})

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.YesNoAnswer{Yes: false},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, s.Players[0].Discard, engine.CardID("village"))
}

func TestVassal_OnPlay_EmptyDeck_NoEffect(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"vassal"}
	s.Players[0].Actions = 1

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "vassal"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, s.PendingDecision)
}
