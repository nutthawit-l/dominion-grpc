package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func applyTestLookup(id CardID) (*Card, bool) {
	c, ok := basicsLookup[id]
	if !ok {
		return nil, false
	}
	// Treasures need an OnPlay that adds coins for Apply tests.
	switch id {
	case "copper":
		cc := *c
		cc.OnPlay = func(gs *GameState, p int) []Event { return AddCoins(gs, p, 1) }
		return &cc, true
	case "silver":
		cc := *c
		cc.OnPlay = func(gs *GameState, p int) []Event { return AddCoins(gs, p, 2) }
		return &cc, true
	case "gold":
		cc := *c
		cc.OnPlay = func(gs *GameState, p int) []Event { return AddCoins(gs, p, 3) }
		return &cc, true
	}
	return c, true
}

func TestApply_PlayCard_TreasureInBuyPhase(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0
	// Force a Copper into the current player's hand and go to Buy phase.
	gs.Players[0].Hand = []CardID{"copper"}
	gs.Phase = PhaseBuy

	_, events, err := Apply(gs, PlayCard{PlayerIdx: 0, Card: "copper"}, applyTestLookup)
	require.NoError(t, err)
	require.Equal(t, 1, gs.Players[0].Coins)
	require.Contains(t, gs.Players[0].InPlay, CardID("copper"))
	require.NotContains(t, gs.Players[0].Hand, CardID("copper"))
	require.NotEmpty(t, events)
}

func TestApply_PlayCard_NotMyTurnErrors(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0
	gs.Players[1].Hand = []CardID{"copper"}
	gs.Phase = PhaseBuy

	_, _, err := Apply(gs, PlayCard{PlayerIdx: 1, Card: "copper"}, applyTestLookup)
	require.ErrorIs(t, err, ErrNotYourTurn)
}

func TestApply_PlayCard_WrongPhaseErrors(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0
	gs.Players[0].Hand = []CardID{"copper"}
	// still in Action phase

	_, _, err := Apply(gs, PlayCard{PlayerIdx: 0, Card: "copper"}, applyTestLookup)
	require.ErrorIs(t, err, ErrWrongPhase)
}

func TestApply_BuyCard_DeductsCoinsAndGainsToDiscard(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0
	gs.Phase = PhaseBuy
	gs.Players[0].Coins = 3
	gs.Players[0].Buys = 1

	_, _, err := Apply(gs, BuyCard{PlayerIdx: 0, Card: "silver"}, applyTestLookup)
	require.NoError(t, err)
	require.Equal(t, 0, gs.Players[0].Coins)
	require.Equal(t, 0, gs.Players[0].Buys)
	require.Contains(t, gs.Players[0].Discard, CardID("silver"))
}

func TestApply_BuyCard_NotEnoughCoinsErrors(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0
	gs.Phase = PhaseBuy
	gs.Players[0].Coins = 2
	gs.Players[0].Buys = 1

	_, _, err := Apply(gs, BuyCard{PlayerIdx: 0, Card: "silver"}, applyTestLookup)
	require.ErrorIs(t, err, ErrInsufficientCoins)
}

func TestApply_EndPhase_ActionToBuy(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0

	_, _, err := Apply(gs, EndPhase{PlayerIdx: 0}, applyTestLookup)
	require.NoError(t, err)
	require.Equal(t, PhaseBuy, gs.Phase)
	require.Equal(t, 0, gs.CurrentPlayer) // still player 0
}

func TestApply_EndPhase_BuyToNextPlayer(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0
	gs.Phase = PhaseBuy

	_, _, err := Apply(gs, EndPhase{PlayerIdx: 0}, applyTestLookup)
	require.NoError(t, err)
	require.Equal(t, PhaseAction, gs.Phase)
	require.Equal(t, 1, gs.CurrentPlayer)
}

func TestApply_EndPhase_GameEndsAfterBuyIfProvinceGone(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0
	gs.Phase = PhaseBuy
	gs.Supply.Piles["province"] = 0

	_, _, err := Apply(gs, EndPhase{PlayerIdx: 0}, applyTestLookup)
	require.NoError(t, err)
	require.True(t, gs.Ended)
}

// basicsLookup2 returns cards with OnPlay attached so Apply works.
func basicsLookup2(id CardID) (*Card, bool) { return applyTestLookup(id) }

func TestApply_ResolveDecision_NoDecisionPending(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0

	_, _, err := Apply(gs, ResolveDecision{PlayerIdx: 0, DecisionID: "d1"}, applyTestLookup)
	require.ErrorIs(t, err, ErrNoDecisionPending)
}

func TestApply_ResolveDecision_WrongID(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0
	gs.PendingDecision = &Decision{
		ID: "d1", PlayerIdx: 0, CardID: "cellar",
		Prompt: DiscardFromHandPrompt{Min: 0, Max: 3},
	}

	_, _, err := Apply(gs, ResolveDecision{PlayerIdx: 0, DecisionID: "wrong"}, applyTestLookup)
	require.ErrorIs(t, err, ErrWrongDecisionID)
}

func TestApply_DecisionPending_BlocksNonResolveActions(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0
	gs.PendingDecision = &Decision{
		ID: "d1", PlayerIdx: 0, CardID: "cellar",
		Prompt: DiscardFromHandPrompt{},
	}

	_, _, err := Apply(gs, EndPhase{PlayerIdx: 0}, applyTestLookup)
	require.ErrorIs(t, err, ErrDecisionPending)
}

func TestApply_ResolveDecision_FromDecidingPlayer_NotCurrentPlayer(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0
	// Decision is for player 1 (not the current player).
	resolved := false
	testCard := &Card{
		ID: "testcard", Types: []CardType{TypeAction},
		OnResolve: func(gs *GameState, p int, d *Decision, a Answer, lookup CardLookup) ([]Event, error) {
			resolved = true
			return nil, nil
		},
	}
	gs.PendingDecision = &Decision{
		ID: "d1", PlayerIdx: 1, CardID: "testcard",
		Prompt: DiscardFromHandPrompt{},
	}
	testLookup := func(id CardID) (*Card, bool) {
		if id == "testcard" {
			return testCard, true
		}
		return applyTestLookup(id)
	}

	_, _, err := Apply(gs, ResolveDecision{PlayerIdx: 1, DecisionID: "d1", Answer: CardListAnswer{}}, testLookup)
	require.NoError(t, err)
	require.True(t, resolved)
	require.Nil(t, gs.PendingDecision)
}
