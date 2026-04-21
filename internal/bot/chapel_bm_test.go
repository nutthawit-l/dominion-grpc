package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

func TestChapelBM_BuysChapelOnEarlyTurn(t *testing.T) {
	cs := &ClientState{
		Me: 0, Phase: pb.Phase_PHASE_BUY, MyTurnsTaken: 1,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Hand: nil, Buys: 1, Coins: 2, Actions: 0},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "chapel", Count: 10},
				{CardId: "province", Count: 8},
			},
		},
		CurrentPlayer: 0,
	}
	strat := NewChapelBM()
	act := strat.PickAction(cs)
	require.NotNil(t, act)
	buy := act.GetBuyCard()
	require.NotNil(t, buy)
	require.Equal(t, "chapel", buy.CardId)
}

func TestChapelBM_DoesNotBuySecondChapel(t *testing.T) {
	cs := &ClientState{
		Me: 0, Phase: pb.Phase_PHASE_BUY, MyTurnsTaken: 2,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Hand: nil, Buys: 1, Coins: 2, Actions: 0},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "chapel", Count: 9},
				{CardId: "province", Count: 8},
			},
		},
		CurrentPlayer: 0,
	}
	strat := NewChapelBM()
	strat.chapelOwned = true
	act := strat.PickAction(cs)
	if act != nil && act.GetBuyCard() != nil {
		require.NotEqual(t, "chapel", act.GetBuyCard().CardId)
	}
}

func TestChapelBM_Resolve_TrashesEstatesAndCoppers(t *testing.T) {
	cs := &ClientState{
		Me: 0,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Hand: []string{"estate", "copper", "estate", "copper"}},
			},
		},
	}
	strat := NewChapelBM()
	d := &pb.Decision{
		Id: "d1", PlayerIdx: 0, CardId: "chapel",
		Prompt: &pb.Decision_TrashFromHand{TrashFromHand: &pb.TrashFromHandPrompt{
			Min: 0, Max: 4,
		}},
	}
	r := strat.Resolve(cs, d)
	cl := r.GetCardList()
	require.NotNil(t, cl)
	require.Equal(t, 4, len(cl.Cards))
}

func TestChapelBM_Resolve_KeepsMinCoppers(t *testing.T) {
	cs := &ClientState{
		Me: 0,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Hand: []string{"copper", "copper", "copper", "copper"}},
			},
		},
	}
	strat := NewChapelBM()
	strat.SetTrashedCount("copper", 4)
	d := &pb.Decision{
		Id: "d1", PlayerIdx: 0, CardId: "chapel",
		Prompt: &pb.Decision_TrashFromHand{TrashFromHand: &pb.TrashFromHandPrompt{
			Min: 0, Max: 4,
		}},
	}
	r := strat.Resolve(cs, d)
	cl := r.GetCardList()
	require.NotNil(t, cl)
	require.Equal(t, 0, len(cl.Cards))
}
