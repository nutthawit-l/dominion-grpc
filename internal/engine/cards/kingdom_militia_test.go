package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestMilitia_Metadata(t *testing.T) {
	require.Equal(t, "Militia", Militia.Name)
	require.Equal(t, 4, Militia.Cost)
	require.True(t, Militia.HasType(engine.TypeAction))
	require.True(t, Militia.HasType(engine.TypeAttack))
}

func TestMilitia_OnPlay_AddsCoinsAndPromptsOpponent(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"militia"}
	gs.Players[1].Hand = []engine.CardID{"copper", "copper", "estate", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "militia"}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 2, gs.Players[0].Coins)

	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, engine.PlayerIdx(1), gs.PendingDecision.PlayerIdx)
	p, ok := gs.PendingDecision.Prompt.(engine.DiscardFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 2, p.Min) // 5-card hand, discard to 3
	require.Equal(t, 2, p.Max)
}

func TestMilitia_OnPlay_OpponentWithThreeOrFewer_NoDecision(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"militia"}
	gs.Players[1].Hand = []engine.CardID{"copper", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "militia"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
}

func TestMilitia_Resolve_OpponentDiscardsChosenCards(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"militia"}
	gs.Players[1].Hand = []engine.CardID{"copper", "copper", "estate", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "militia"}, testLookup)
	require.NoError(t, err)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardListAnswer{Cards: []engine.CardID{"estate", "estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Len(t, gs.Players[1].Hand, 3)
	require.Contains(t, gs.Players[1].Discard, engine.CardID("estate"))
}

func TestMilitia_Resolve_WrongCount_Error(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"militia"}
	gs.Players[1].Hand = []engine.CardID{"copper", "copper", "estate", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "militia"}, testLookup)
	require.NoError(t, err)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardListAnswer{Cards: []engine.CardID{"estate"}}, // only 1, need 2
	}, testLookup)
	require.Error(t, err)
}

func TestMilitia_OnPlay_MoatBlocksOpponent(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"militia"}
	gs.Players[1].Hand = []engine.CardID{"moat", "copper", "copper", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "militia"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision, "Moat-holding opponent must not be prompted")
	require.Equal(t, 2, gs.Players[0].Coins, "+2 coins still happens for the attacker")
}

func TestMilitia_ThreePlayer_QueuesBothOpponents(t *testing.T) {
	gs := newTestStateForCards(3)
	gs.Phase = engine.PhaseAction
	gs.CurrentPlayer = 0
	gs.Players[0].Actions = 1
	gs.Players[0].Hand = []engine.CardID{"militia"}
	gs.Players[1].Hand = []engine.CardID{"copper", "copper", "estate", "estate", "silver"}
	gs.Players[2].Hand = []engine.CardID{"copper", "copper", "gold", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "militia"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, engine.PlayerIdx(1), gs.PendingDecision.PlayerIdx, "opp at seat 1 is prompted first")

	// Resolve opp 1
	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardListAnswer{Cards: []engine.CardID{"estate", "estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision, "opp 2 must now be prompted")
	require.Equal(t, engine.PlayerIdx(2), gs.PendingDecision.PlayerIdx)

	// Resolve opp 2
	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  2,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardListAnswer{Cards: []engine.CardID{"copper", "copper"}},
	}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision, "all opponents resolved")
}
