package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestVassal_OnPlay_DiscardsTopCard_NonAction_NoDecision(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"vassal"}
	gs.Players[0].Deck = []engine.CardID{"copper"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "vassal"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Contains(t, gs.Players[0].Discard, engine.CardID("copper"))
}

func TestVassal_OnPlay_DiscardsAction_AsksToPlay(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"vassal"}
	gs.Players[0].Deck = []engine.CardID{"village"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "vassal"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)

	prompt, ok := gs.PendingDecision.Prompt.(engine.MayPlayActionPrompt)
	require.True(t, ok)
	require.Equal(t, engine.CardID("village"), prompt.Card)
}

func TestVassal_OnResolve_Yes_PlaysFromDiscard(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Discard = []engine.CardID{"village"}
	gs.Players[0].Deck = []engine.CardID{"gold", "gold"}

	engine.RequestDecision(gs, 0, "vassal", 0,
		engine.MayPlayActionPrompt{Card: "village"},
		map[engine.ContextKey]any{engine.CtxKeyCard: engine.CardID("village")})

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.YesNoAnswer{Yes: true},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[0].InPlay, engine.CardID("village"))
	require.Empty(t, gs.Players[0].Discard)
	require.Len(t, gs.Players[0].Hand, 1) // village drew 1
}

func TestVassal_OnResolve_No_DoesNothing(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Discard = []engine.CardID{"village"}

	engine.RequestDecision(gs, 0, "vassal", 0,
		engine.MayPlayActionPrompt{Card: "village"},
		map[engine.ContextKey]any{engine.CtxKeyCard: engine.CardID("village")})

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.YesNoAnswer{Yes: false},
	}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[0].Discard, engine.CardID("village"))
}

func TestVassal_OnPlay_EmptyDeck_NoEffect(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"vassal"}
	gs.Players[0].Actions = 1

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "vassal"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
}
