package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestBureaucrat_Metadata(t *testing.T) {
	require.Equal(t, "Bureaucrat", Bureaucrat.Name)
	require.Equal(t, 4, Bureaucrat.Cost)
	require.True(t, Bureaucrat.HasType(engine.TypeAction))
	require.True(t, Bureaucrat.HasType(engine.TypeAttack))
}

func TestBureaucrat_OnPlay_GainsSilverToDeck(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[0].Deck, engine.CardID("silver"))
	require.Equal(t, 9, gs.Supply.Piles["silver"])
}

func TestBureaucrat_OnPlay_SilverPileEmpty_AttackStillRuns(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 0
	gs.Players[1].Hand = []engine.CardID{"estate", "copper"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)
	// 1 Victory (estate) → forced put-on-deck, no prompt
	require.Contains(t, gs.Players[1].Deck, engine.CardID("estate"))
}

func TestBureaucrat_OnPlay_OpponentZeroVictories_NoDecision(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10
	gs.Players[1].Hand = []engine.CardID{"copper", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.NotContains(t, gs.Players[1].Deck, engine.CardID("estate"))
}

func TestBureaucrat_OnPlay_OpponentOneVictory_ForcedNoDecision(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10
	gs.Players[1].Hand = []engine.CardID{"copper", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Contains(t, gs.Players[1].Deck, engine.CardID("estate"),
		"the sole Victory is auto-moved to top of deck")
	require.NotContains(t, gs.Players[1].Hand, engine.CardID("estate"))
}

func TestBureaucrat_OnPlay_OpponentMultipleVictories_Prompts(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10
	gs.Players[1].Hand = []engine.CardID{"estate", "duchy", "copper"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, engine.PlayerIdx(1), gs.PendingDecision.PlayerIdx)
	p, ok := gs.PendingDecision.Prompt.(engine.PutOnDeckPrompt)
	require.True(t, ok)
	require.Contains(t, p.TypeFilter, engine.TypeVictory)
}

func TestBureaucrat_Resolve_PutsChosenVictoryOnDeck(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10
	gs.Players[1].Hand = []engine.CardID{"estate", "duchy", "copper"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardListAnswer{Cards: []engine.CardID{"estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Contains(t, gs.Players[1].Deck, engine.CardID("estate"))
	require.NotContains(t, gs.Players[1].Hand, engine.CardID("estate"))
	require.Contains(t, gs.Players[1].Hand, engine.CardID("duchy"),
		"the other Victory stays in hand")
}

func TestBureaucrat_Resolve_NonVictoryAnswer_Error(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10
	gs.Players[1].Hand = []engine.CardID{"estate", "duchy", "copper"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardListAnswer{Cards: []engine.CardID{"copper"}}, // not a Victory
	}, testLookup)
	require.Error(t, err)
}

func TestBureaucrat_OnPlay_MoatBlocksOpponent(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10
	gs.Players[1].Hand = []engine.CardID{"moat", "estate", "estate"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.NotContains(t, gs.Players[1].Deck, engine.CardID("estate"))
	require.Contains(t, gs.Players[0].Deck, engine.CardID("silver"),
		"attacker's own Silver gain to deck still happens")
}
