package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestBandit_Metadata(t *testing.T) {
	require.Equal(t, "Bandit", Bandit.Name)
	require.Equal(t, 5, Bandit.Cost)
	require.True(t, Bandit.HasType(engine.TypeAction))
	require.True(t, Bandit.HasType(engine.TypeAttack))
}

func TestBandit_OnPlay_GainsGoldToDiscard(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[0].Discard, engine.CardID("gold"))
	require.Equal(t, 29, gs.Supply.Piles["gold"])
}

func TestBandit_OnPlay_OpponentNoNonCopperTreasures_AutoDiscardsBoth(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Deck = []engine.CardID{"estate", "copper"} // copper is top

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Empty(t, gs.Players[1].Deck)
	require.ElementsMatch(t, []engine.CardID{"estate", "copper"}, gs.Players[1].Discard)
	require.Empty(t, gs.Trash)
}

func TestBandit_OnPlay_OpponentOneNonCopperTreasure_AutoTrashes(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Deck = []engine.CardID{"estate", "silver"} // silver on top

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Contains(t, gs.Trash, engine.CardID("silver"))
	require.Contains(t, gs.Players[1].Discard, engine.CardID("estate"))
}

func TestBandit_OnPlay_OpponentTwoNonCopperTreasures_Prompts(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Deck = []engine.CardID{"silver", "gold"} // gold on top

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)
	p, ok := gs.PendingDecision.Prompt.(engine.TrashFromRevealedPrompt)
	require.True(t, ok)
	require.ElementsMatch(t, []engine.CardID{"silver", "gold"}, p.Cards)
}

func TestBandit_Resolve_TrashesChosenAndDiscardsRest(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Deck = []engine.CardID{"silver", "gold"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardChoiceAnswer{Card: "gold"},
	}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Contains(t, gs.Trash, engine.CardID("gold"))
	require.Contains(t, gs.Players[1].Discard, engine.CardID("silver"))
	require.Empty(t, gs.Players[1].Deck)
}

func TestBandit_Resolve_CardNotInRevealed_Error(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Deck = []engine.CardID{"silver", "gold"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardChoiceAnswer{Card: "estate"}, // not in revealed
	}, testLookup)
	require.Error(t, err)
}

func TestBandit_OnPlay_OpponentDeckEmpty_ShufflesOrSkips(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Deck = nil
	gs.Players[1].Discard = nil

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision, "no cards to reveal; no prompt")
}

func TestBandit_OnPlay_MoatBlocksOpponent(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Hand = []engine.CardID{"moat"}
	gs.Players[1].Deck = []engine.CardID{"silver", "gold"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Equal(t, []engine.CardID{"silver", "gold"}, gs.Players[1].Deck,
		"blocked opponent's deck must be untouched")
	require.Empty(t, gs.Trash)
	require.Contains(t, gs.Players[0].Discard, engine.CardID("gold"),
		"attacker still gains the Gold")
}
