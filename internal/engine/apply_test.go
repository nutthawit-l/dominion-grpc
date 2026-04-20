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
		cc.OnPlay = func(s *GameState, p int) []Event { return AddCoins(s, p, 1) }
		return &cc, true
	case "silver":
		cc := *c
		cc.OnPlay = func(s *GameState, p int) []Event { return AddCoins(s, p, 2) }
		return &cc, true
	case "gold":
		cc := *c
		cc.OnPlay = func(s *GameState, p int) []Event { return AddCoins(s, p, 3) }
		return &cc, true
	}
	return c, true
}

func TestApply_PlayCard_TreasureInBuyPhase(t *testing.T) {
	s, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	s.CurrentPlayer = 0
	// Force a Copper into the current player's hand and go to Buy phase.
	s.Players[0].Hand = []CardID{"copper"}
	s.Phase = PhaseBuy

	_, events, err := Apply(s, PlayCard{PlayerIdx: 0, Card: "copper"}, applyTestLookup)
	require.NoError(t, err)
	require.Equal(t, 1, s.Players[0].Coins)
	require.Contains(t, s.Players[0].InPlay, CardID("copper"))
	require.NotContains(t, s.Players[0].Hand, CardID("copper"))
	require.NotEmpty(t, events)
}

func TestApply_PlayCard_NotMyTurnErrors(t *testing.T) {
	s, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	s.CurrentPlayer = 0
	s.Players[1].Hand = []CardID{"copper"}
	s.Phase = PhaseBuy

	_, _, err := Apply(s, PlayCard{PlayerIdx: 1, Card: "copper"}, applyTestLookup)
	require.ErrorIs(t, err, ErrNotYourTurn)
}

func TestApply_PlayCard_WrongPhaseErrors(t *testing.T) {
	s, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	s.CurrentPlayer = 0
	s.Players[0].Hand = []CardID{"copper"}
	// still in Action phase

	_, _, err := Apply(s, PlayCard{PlayerIdx: 0, Card: "copper"}, applyTestLookup)
	require.ErrorIs(t, err, ErrWrongPhase)
}

func TestApply_BuyCard_DeductsCoinsAndGainsToDiscard(t *testing.T) {
	s, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	s.CurrentPlayer = 0
	s.Phase = PhaseBuy
	s.Players[0].Coins = 3
	s.Players[0].Buys = 1

	_, _, err := Apply(s, BuyCard{PlayerIdx: 0, Card: "silver"}, applyTestLookup)
	require.NoError(t, err)
	require.Equal(t, 0, s.Players[0].Coins)
	require.Equal(t, 0, s.Players[0].Buys)
	require.Contains(t, s.Players[0].Discard, CardID("silver"))
}

func TestApply_BuyCard_NotEnoughCoinsErrors(t *testing.T) {
	s, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	s.CurrentPlayer = 0
	s.Phase = PhaseBuy
	s.Players[0].Coins = 2
	s.Players[0].Buys = 1

	_, _, err := Apply(s, BuyCard{PlayerIdx: 0, Card: "silver"}, applyTestLookup)
	require.ErrorIs(t, err, ErrInsufficientCoins)
}

func TestApply_EndPhase_ActionToBuy(t *testing.T) {
	s, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	s.CurrentPlayer = 0

	_, _, err := Apply(s, EndPhase{PlayerIdx: 0}, applyTestLookup)
	require.NoError(t, err)
	require.Equal(t, PhaseBuy, s.Phase)
	require.Equal(t, 0, s.CurrentPlayer) // still player 0
}

func TestApply_EndPhase_BuyToNextPlayer(t *testing.T) {
	s, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	s.CurrentPlayer = 0
	s.Phase = PhaseBuy

	_, _, err := Apply(s, EndPhase{PlayerIdx: 0}, applyTestLookup)
	require.NoError(t, err)
	require.Equal(t, PhaseAction, s.Phase)
	require.Equal(t, 1, s.CurrentPlayer)
}

func TestApply_EndPhase_GameEndsAfterBuyIfProvinceGone(t *testing.T) {
	s, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	s.CurrentPlayer = 0
	s.Phase = PhaseBuy
	s.Supply.Piles["province"] = 0

	_, _, err := Apply(s, EndPhase{PlayerIdx: 0}, applyTestLookup)
	require.NoError(t, err)
	require.True(t, s.Ended)
}

// basicsLookup2 returns cards with OnPlay attached so Apply works.
func basicsLookup2(id CardID) (*Card, bool) { return applyTestLookup(id) }

func TestApply_ResolveDecision_NoDecisionPending(t *testing.T) {
	s, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	s.CurrentPlayer = 0

	_, _, err := Apply(s, ResolveDecision{PlayerIdx: 0, DecisionID: "d1"}, applyTestLookup)
	require.ErrorIs(t, err, ErrNoDecisionPending)
}

func TestApply_ResolveDecision_WrongID(t *testing.T) {
	s, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	s.CurrentPlayer = 0
	s.PendingDecision = &Decision{
		ID: "d1", PlayerIdx: 0, CardID: "cellar",
		Prompt: DiscardFromHandPrompt{Min: 0, Max: 3},
	}

	_, _, err := Apply(s, ResolveDecision{PlayerIdx: 0, DecisionID: "wrong"}, applyTestLookup)
	require.ErrorIs(t, err, ErrWrongDecisionID)
}

func TestApply_DecisionPending_BlocksNonResolveActions(t *testing.T) {
	s, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	s.CurrentPlayer = 0
	s.PendingDecision = &Decision{
		ID: "d1", PlayerIdx: 0, CardID: "cellar",
		Prompt: DiscardFromHandPrompt{},
	}

	_, _, err := Apply(s, EndPhase{PlayerIdx: 0}, applyTestLookup)
	require.ErrorIs(t, err, ErrDecisionPending)
}

func TestApply_ResolveDecision_FromDecidingPlayer_NotCurrentPlayer(t *testing.T) {
	s, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	s.CurrentPlayer = 0
	// Decision is for player 1 (not the current player).
	resolved := false
	testCard := &Card{
		ID: "testcard", Types: []CardType{TypeAction},
		OnResolve: func(s *GameState, p int, d *Decision, a Answer, lookup CardLookup) ([]Event, error) {
			resolved = true
			return nil, nil
		},
	}
	s.PendingDecision = &Decision{
		ID: "d1", PlayerIdx: 1, CardID: "testcard",
		Prompt: DiscardFromHandPrompt{},
	}
	testLookup := func(id CardID) (*Card, bool) {
		if id == "testcard" {
			return testCard, true
		}
		return applyTestLookup(id)
	}

	_, _, err := Apply(s, ResolveDecision{PlayerIdx: 1, DecisionID: "d1", Answer: CardListAnswer{}}, testLookup)
	require.NoError(t, err)
	require.True(t, resolved)
	require.Nil(t, s.PendingDecision)
}
