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
				{CardId: "chapel", Count: 9}, // 1 already bought
				{CardId: "province", Count: 8},
			},
		},
		CurrentPlayer: 0,
	}
	strat := NewChapelBM()
	strat.chapelOwned = true
	act := strat.PickAction(cs)
	// Should not buy chapel — should end phase or buy silver if coins allow.
	if act != nil && act.GetBuyCard() != nil {
		require.NotEqual(t, "chapel", act.GetBuyCard().CardId)
	}
}

func TestChapelBM_Resolve_TrashesEstatesFirst(t *testing.T) {
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
	// Should trash both estates.
	estateCount := 0
	for _, c := range cl.Cards {
		if c == "estate" {
			estateCount++
		}
	}
	require.Equal(t, 2, estateCount)
}

func TestChapelBM_Resolve_KeepsThreeCoppers(t *testing.T) {
	cs := &ClientState{
		Me: 0,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Hand: []string{"copper", "copper", "copper", "copper"}},
			},
		},
	}
	strat := NewChapelBM()
	// Starting: 7 coppers. Trash threshold: keep 3. Can trash up to 4.
	// So should trash 4 coppers (7-4=3, at threshold).
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
