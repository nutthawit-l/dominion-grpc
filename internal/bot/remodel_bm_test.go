package bot

import (
	"testing"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/stretchr/testify/require"
)

func TestRemodelBM_BuysRemodelOnEarlyTurn(t *testing.T) {
	cs := &ClientState{
		Me: 0, Phase: pb.Phase_PHASE_BUY, MyTurnsTaken: 2,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Hand: nil, Buys: 1, Coins: 4, Actions: 0},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "remodel", Count: 10},
				{CardId: "province", Count: 8},
			},
		},
		CurrentPlayer: 0,
	}
	strat := NewRemodelBM()
	act := strat.PickAction(cs)
	require.NotNil(t, act)
	buy := act.GetBuyCard()
	require.NotNil(t, buy)
	require.Equal(t, "remodel", buy.CardId)
}

func TestRemodelBM_Resolve_Step0_TrashesEstate(t *testing.T) {
	cs := &ClientState{
		Me: 0,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Hand: []string{"estate", "copper", "silver"}},
			},
		},
	}
	strat := NewRemodelBM()
	d := &pb.Decision{
		Id: "d1", PlayerIdx: 0, CardId: "remodel", Step: 0,
		Prompt: &pb.Decision_TrashFromHand{TrashFromHand: &pb.TrashFromHandPrompt{
			Min: 1, Max: 1,
		}},
	}
	r := strat.Resolve(cs, d)
	cl := r.GetCardList()
	require.NotNil(t, cl)
	require.Equal(t, []string{"estate"}, cl.Cards)
}

func TestRemodelBM_Resolve_Step1_GainsMostExpensive(t *testing.T) {
	cs := &ClientState{
		Me: 0,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "silver", Count: 10},
				{CardId: "gold", Count: 10},
				{CardId: "estate", Count: 8},
			},
		},
	}
	strat := NewRemodelBM()
	d := &pb.Decision{
		Id: "d2", PlayerIdx: 0, CardId: "remodel", Step: 1,
		Prompt: &pb.Decision_GainFromSupply{GainFromSupply: &pb.GainFromSupplyPrompt{
			MaxCost: 4,
		}},
	}
	r := strat.Resolve(cs, d)
	cc := r.GetCardChoice()
	require.NotNil(t, cc)
	require.Equal(t, "silver", cc.Card) // silver costs 3, within max 4
}
