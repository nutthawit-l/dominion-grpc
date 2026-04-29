package bot

import (
	"testing"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/stretchr/testify/require"
)

func TestMilitiaBM_Name(t *testing.T) {
	require.Equal(t, "militia_bm", NewMilitiaBM().Name())
}

func TestMilitiaBM_BuysMilitiaOnTurn3(t *testing.T) {
	s := NewMilitiaBM()
	cs := &ClientState{
		Me:           0,
		Phase:        pb.Phase_PHASE_BUY,
		MyTurnsTaken: 3,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Coins: 4, Buys: 1},
				{PlayerIdx: 1},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "militia", Count: 10},
				{CardId: "province", Count: 8},
				{CardId: "gold", Count: 30},
				{CardId: "silver", Count: 40},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.Equal(t, "militia", action.GetBuyCard().CardId)
}

func TestMilitiaBM_Resolve_DiscardsToThree_KeepsBestTreasures(t *testing.T) {
	s := NewMilitiaBM()
	cs := &ClientState{Me: 1, Snapshot: &pb.GameStateSnapshot{
		Players: []*pb.PlayerView{
			{PlayerIdx: 0},
			{PlayerIdx: 1, Hand: []string{"gold", "silver", "copper", "estate", "curse"}},
		},
	}}
	d := &pb.Decision{
		Id: "d1", PlayerIdx: 1, CardId: "militia",
		Prompt: &pb.Decision_DiscardFromHand{DiscardFromHand: &pb.DiscardFromHandPrompt{Min: 2, Max: 2}},
	}

	r := s.Resolve(cs, d)

	require.NotNil(t, r)
	cards := r.GetCardList().Cards
	require.Len(t, cards, 2)
	// Must keep gold, silver, copper (the three best). Must discard estate+curse.
	require.Contains(t, cards, "estate")
	require.Contains(t, cards, "curse")
}

func TestMilitiaBM_DoesNotBuySecondMilitia(t *testing.T) {
	s := NewMilitiaBM()
	s.militiaOwned = true
	cs := &ClientState{
		Me:           0,
		Phase:        pb.Phase_PHASE_BUY,
		MyTurnsTaken: 3,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Coins: 4, Buys: 1},
				{PlayerIdx: 1},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "militia", Count: 10},
				{CardId: "province", Count: 8},
				{CardId: "gold", Count: 30},
				{CardId: "silver", Count: 40},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.NotEqual(t, "militia", action.GetBuyCard().CardId,
		"must not buy a second militia")
}

func TestMilitiaBM_DoesNotBuyMilitiaAfterTurn4(t *testing.T) {
	s := NewMilitiaBM()
	cs := &ClientState{
		Me:           0,
		Phase:        pb.Phase_PHASE_BUY,
		MyTurnsTaken: 5,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Coins: 4, Buys: 1},
				{PlayerIdx: 1},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "militia", Count: 10},
				{CardId: "province", Count: 8},
				{CardId: "gold", Count: 30},
				{CardId: "silver", Count: 40},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.NotEqual(t, "militia", action.GetBuyCard().CardId)
}

func TestMilitiaBM_PlaysWhenActionsAvailable(t *testing.T) {
	s := NewMilitiaBM()
	cs := &ClientState{
		Me:    0,
		Phase: pb.Phase_PHASE_ACTION,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Actions: 1, Hand: []string{"militia", "copper"}},
				{PlayerIdx: 1},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.Equal(t, "militia", action.GetPlayCard().CardId)
}

func TestMilitiaBM_Resolve_OtherPromptTypes_UsesSafeRefusal(t *testing.T) {
	s := NewMilitiaBM()
	cs := &ClientState{Me: 1, Snapshot: &pb.GameStateSnapshot{
		Players: []*pb.PlayerView{{PlayerIdx: 0}, {PlayerIdx: 1, Hand: []string{"copper"}}},
	}}
	d := &pb.Decision{
		Id: "d1", PlayerIdx: 1,
		Prompt: &pb.Decision_TrashFromHand{TrashFromHand: &pb.TrashFromHandPrompt{Min: 0, Max: 1}},
	}
	r := s.Resolve(cs, d)
	require.Equal(t, "d1", r.DecisionId)
}
