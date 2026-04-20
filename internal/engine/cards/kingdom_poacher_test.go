package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestPoacher_OnPlay_ZeroEmptyPiles_NoDecision(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"poacher"}
	s.Players[0].Deck = []engine.CardID{"copper"}
	s.Players[0].Actions = 1
	s.Supply.Piles["silver"] = 10
	s.Supply.Piles["gold"] = 10

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "poacher"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, s.PendingDecision)
	require.Equal(t, 1, s.Players[0].Coins)
	require.Equal(t, 1, s.Players[0].Actions)
}

func TestPoacher_OnPlay_OneEmptyPile_DiscardsOne(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"poacher"}
	s.Players[0].Deck = []engine.CardID{"copper"}
	s.Players[0].Actions = 1
	s.Supply.Piles["silver"] = 0
	s.Supply.Piles["gold"] = 10

	_, _, err := engine.Apply(s, engine.PlayCard{PlayerIdx: 0, Card: "poacher"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, s.PendingDecision)

	prompt, ok := s.PendingDecision.Prompt.(engine.DiscardFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 1, prompt.Min)
	require.Equal(t, 1, prompt.Max)
}

func TestPoacher_OnResolve_Discards(t *testing.T) {
	s := newActionPhaseState()
	s.Players[0].Hand = []engine.CardID{"copper", "estate"}

	engine.RequestDecision(s, 0, "poacher", 0,
		engine.DiscardFromHandPrompt{Min: 1, Max: 1}, nil)

	_, _, err := engine.Apply(s, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: s.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: []engine.CardID{"estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.Equal(t, []engine.CardID{"copper"}, s.Players[0].Hand)
	require.Contains(t, s.Players[0].Discard, engine.CardID("estate"))
}
