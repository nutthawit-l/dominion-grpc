package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

func TestWitchBM_Name(t *testing.T) {
	require.Equal(t, "witch_bm", NewWitchBM().Name())
}

func TestWitchBM_BuysWitchOnTurn3_WhenCoinsAtLeast5(t *testing.T) {
	s := NewWitchBM()
	cs := &ClientState{
		Me:           0,
		Phase:        pb.Phase_PHASE_BUY,
		MyTurnsTaken: 3,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Coins: 5, Buys: 1},
				{PlayerIdx: 1},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "witch", Count: 10},
				{CardId: "province", Count: 8},
				{CardId: "gold", Count: 30},
				{CardId: "silver", Count: 40},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.Equal(t, "witch", action.GetBuyCard().CardId)
}

func TestWitchBM_DoesNotBuySecondWitch(t *testing.T) {
	s := NewWitchBM()
	s.witchOwned = true
	cs := &ClientState{
		Me:           0,
		Phase:        pb.Phase_PHASE_BUY,
		MyTurnsTaken: 3,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Coins: 5, Buys: 1},
				{PlayerIdx: 1},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "witch", Count: 10},
				{CardId: "province", Count: 8},
				{CardId: "gold", Count: 30},
				{CardId: "silver", Count: 40},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.NotEqual(t, "witch", action.GetBuyCard().CardId,
		"must not buy a second witch")
}

func TestWitchBM_DoesNotBuyWitchAfterTurn4(t *testing.T) {
	s := NewWitchBM()
	cs := &ClientState{
		Me:           0,
		Phase:        pb.Phase_PHASE_BUY,
		MyTurnsTaken: 5,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Coins: 5, Buys: 1},
				{PlayerIdx: 1},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "witch", Count: 10},
				{CardId: "province", Count: 8},
				{CardId: "gold", Count: 30},
				{CardId: "silver", Count: 40},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.NotEqual(t, "witch", action.GetBuyCard().CardId)
}

func TestWitchBM_PlaysWitchWhenActionsAvailable(t *testing.T) {
	s := NewWitchBM()
	cs := &ClientState{
		Me:    0,
		Phase: pb.Phase_PHASE_ACTION,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Actions: 1, Hand: []string{"witch", "copper"}},
				{PlayerIdx: 1},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.Equal(t, "witch", action.GetPlayCard().CardId)
}

func TestWitchBM_Resolve_UsesSafeRefusal(t *testing.T) {
	s := NewWitchBM()
	cs := &ClientState{Me: 0, Snapshot: &pb.GameStateSnapshot{
		Players: []*pb.PlayerView{{PlayerIdx: 0, Hand: []string{"copper"}}, {PlayerIdx: 1}},
	}}
	d := &pb.Decision{
		Id: "d1", PlayerIdx: 0,
		Prompt: &pb.Decision_DiscardFromHand{DiscardFromHand: &pb.DiscardFromHandPrompt{Min: 0, Max: 0}},
	}
	r := s.Resolve(cs, d)
	require.Equal(t, "d1", r.DecisionId)
}
