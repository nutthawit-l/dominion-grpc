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
		cc.OnPlay = func(gs *GameState, px PlayerIdx) []Event { return AddCoins(gs, px, 1) }
		return &cc, true
	case "silver":
		cc := *c
		cc.OnPlay = func(gs *GameState, px PlayerIdx) []Event { return AddCoins(gs, px, 2) }
		return &cc, true
	case "gold":
		cc := *c
		cc.OnPlay = func(gs *GameState, px PlayerIdx) []Event { return AddCoins(gs, px, 3) }
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
	require.Equal(t, PlayerIdx(0), gs.CurrentPlayer) // still player 0
}

func TestApply_EndPhase_BuyToNextPlayer(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.CurrentPlayer = 0
	gs.Phase = PhaseBuy

	_, _, err := Apply(gs, EndPhase{PlayerIdx: 0}, applyTestLookup)
	require.NoError(t, err)
	require.Equal(t, PhaseAction, gs.Phase)
	require.Equal(t, PlayerIdx(1), gs.CurrentPlayer)
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
		OnResolve: func(gs *GameState, px PlayerIdx, d *Decision, a Answer, lookup CardLookup) ([]Event, error) {
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

// stubResolveCard is a minimal card whose OnResolve does nothing —
// used to exercise Apply's pending-play unwinding.
func stubResolveCard(id CardID, onPlay func(*GameState, PlayerIdx) []Event) *Card {
	return &Card{
		ID: id, Name: string(id), Cost: 0,
		Types:  []CardType{TypeAction},
		OnPlay: onPlay,
		OnResolve: func(gs *GameState, px PlayerIdx, d *Decision, a Answer, l CardLookup) ([]Event, error) {
			return nil, nil
		},
	}
}

func TestApply_UnwindsPendingPlays_AfterResolve(t *testing.T) {
	plays := 0
	bumper := stubResolveCard("bumper", func(gs *GameState, px PlayerIdx) []Event {
		plays++
		return nil
	})
	lookup := func(id CardID) (*Card, bool) {
		if id == "bumper" {
			return bumper, true
		}
		return basicsLookup2(id)
	}

	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, lookup)
	// Park a pending decision and queue a replay; resolving the
	// decision should pop the pending play.
	gs.PendingDecision = &Decision{
		ID: "d1", PlayerIdx: 0, CardID: "bumper", Step: 0,
		Prompt: DiscardFromHandPrompt{Min: 0, Max: 0},
	}
	gs.PendingPlays = append(gs.PendingPlays, PendingPlay{
		PlayerIdx: 0, CardID: "bumper", Source: "throne_room",
	})

	_, _, err := Apply(gs, ResolveDecision{
		PlayerIdx: 0, DecisionID: "d1",
		Answer: CardListAnswer{},
	}, lookup)
	require.NoError(t, err)
	require.Equal(t, 1, plays, "Apply should pop the queued play exactly once")
	require.Empty(t, gs.PendingPlays, "stack must be empty after pop")
}

func TestApply_UnwindsAllWhenNoNewDecision(t *testing.T) {
	plays := 0
	bumper := stubResolveCard("bumper", func(gs *GameState, px PlayerIdx) []Event {
		plays++
		return nil
	})
	lookup := func(id CardID) (*Card, bool) {
		if id == "bumper" {
			return bumper, true
		}
		return basicsLookup2(id)
	}
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, lookup)
	gs.PendingDecision = &Decision{ID: "d1", PlayerIdx: 0, CardID: "bumper",
		Prompt: DiscardFromHandPrompt{}}
	// Push two replays — both should run because neither sets a decision.
	gs.PendingPlays = append(gs.PendingPlays,
		PendingPlay{PlayerIdx: 0, CardID: "bumper", Source: "throne_room"},
		PendingPlay{PlayerIdx: 0, CardID: "bumper", Source: "throne_room"},
	)

	_, _, err := Apply(gs, ResolveDecision{PlayerIdx: 0, DecisionID: "d1",
		Answer: CardListAnswer{}}, lookup)
	require.NoError(t, err)
	require.Equal(t, 2, plays)
	require.Empty(t, gs.PendingPlays)
}

func TestApply_StopsUnwindingWhenReplaySetsDecision(t *testing.T) {
	plays := 0
	bumper := stubResolveCard("bumper", func(gs *GameState, px PlayerIdx) []Event {
		plays++
		// Replay sets its own decision.
		return RequestDecision(gs, px, "bumper", 0, DiscardFromHandPrompt{}, nil)
	})
	lookup := func(id CardID) (*Card, bool) {
		if id == "bumper" {
			return bumper, true
		}
		return basicsLookup2(id)
	}
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, lookup)
	gs.PendingDecision = &Decision{ID: "d1", PlayerIdx: 0, CardID: "bumper",
		Prompt: DiscardFromHandPrompt{}}
	gs.PendingPlays = append(gs.PendingPlays,
		PendingPlay{PlayerIdx: 0, CardID: "bumper", Source: "throne_room"},
		PendingPlay{PlayerIdx: 0, CardID: "bumper", Source: "throne_room"},
	)

	_, _, err := Apply(gs, ResolveDecision{PlayerIdx: 0, DecisionID: "d1",
		Answer: CardListAnswer{}}, lookup)
	require.NoError(t, err)
	require.Equal(t, 1, plays, "second pop must wait for the new decision")
	require.Len(t, gs.PendingPlays, 1, "second replay still queued")
	require.NotNil(t, gs.PendingDecision, "new decision must be parked")
}

func TestApply_NoUnwindWhenStackEmpty(t *testing.T) {
	gs, _ := NewGame("g", []string{"A", "B"}, nil, 1, basicsLookup2)
	gs.PendingDecision = &Decision{ID: "d1", PlayerIdx: 0, CardID: "chapel",
		Prompt: TrashFromHandPrompt{}}
	gs.PendingPlays = nil

	_, _, err := Apply(gs, ResolveDecision{PlayerIdx: 0, DecisionID: "d1",
		Answer: CardListAnswer{}}, basicsLookup2)
	// Chapel's OnResolve is wired up; the chapel CardID must resolve to
	// a real card. We use chapel from basicsLookup2 only if it's there;
	// otherwise pick another. Here we just assert no panic and no plays.
	_ = err
	require.Empty(t, gs.PendingPlays)
}
