package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestPoacher_OnPlay_ZeroEmptyPiles_NoDecision(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"poacher"}
	gs.Players[0].Deck = []engine.CardID{"copper"}
	gs.Players[0].Actions = 1
	gs.Supply.Piles["silver"] = 10
	gs.Supply.Piles["gold"] = 10

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "poacher"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Equal(t, 1, gs.Players[0].Coins)
	require.Equal(t, 1, gs.Players[0].Actions)
}

func TestPoacher_OnPlay_OneEmptyPile_DiscardsOne(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"poacher"}
	gs.Players[0].Deck = []engine.CardID{"copper"}
	gs.Players[0].Actions = 1
	gs.Supply.Piles["silver"] = 0
	gs.Supply.Piles["gold"] = 10

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "poacher"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)

	prompt, ok := gs.PendingDecision.Prompt.(engine.DiscardFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 1, prompt.Min)
	require.Equal(t, 1, prompt.Max)
}

func TestPoacher_OnResolve_Discards(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"copper", "estate"}

	engine.RequestDecision(gs, 0, "poacher", 0,
		engine.DiscardFromHandPrompt{Min: 1, Max: 1}, nil)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: []engine.CardID{"estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.Equal(t, []engine.CardID{"copper"}, gs.Players[0].Hand)
	require.Contains(t, gs.Players[0].Discard, engine.CardID("estate"))
}
