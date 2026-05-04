package bot

import (
	"testing"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/stretchr/testify/require"
)

func TestSmithyBM_Name(t *testing.T) {
	require.Equal(t, "smithy_bm", SmithyBM{}.Name())
}

func TestSmithyBM_PickAction_ActionPhase_PlaysSmithyIfAvailable(t *testing.T) {
	cs := csWithMe(0, pb.Phase_PHASE_ACTION, &pb.PlayerView{
		PlayerIdx: 0, Actions: 1,
		Hand: []string{"copper", "smithy"},
	})
	act := SmithyBM{}.PickAction(cs)
	require.NotNil(t, act)
	require.Equal(t, "smithy", act.GetPlayCard().GetCardId())
}

func TestSmithyBM_PickAction_ActionPhase_NoSmithyEndsPhase(t *testing.T) {
	cs := csWithMe(0, pb.Phase_PHASE_ACTION, &pb.PlayerView{
		PlayerIdx: 0, Actions: 1,
		Hand: []string{"copper", "silver"},
	})
	act := SmithyBM{}.PickAction(cs)
	require.NotNil(t, act)
	require.NotNil(t, act.GetEndPhase())
}

func TestSmithyBM_PickAction_ActionPhase_NoActionsEndsPhase(t *testing.T) {
	cs := csWithMe(0, pb.Phase_PHASE_ACTION, &pb.PlayerView{
		PlayerIdx: 0, Actions: 0,
		Hand: []string{"smithy"},
	})
	act := SmithyBM{}.PickAction(cs)
	require.NotNil(t, act.GetEndPhase())
}

func TestSmithyBM_PickAction_BuyPhase_PlaysTreasuresFirst(t *testing.T) {
	cs := csWithMe(0, pb.Phase_PHASE_BUY, &pb.PlayerView{
		PlayerIdx: 0, Buys: 1, Coins: 0,
		Hand: []string{"copper"},
	})
	act := SmithyBM{}.PickAction(cs)
	require.Equal(t, "copper", act.GetPlayCard().GetCardId())
}

func TestSmithyBM_PickAction_BuyPhase_BuyProvinceAtEightCoins(t *testing.T) {
	cs := csWithMe(0, pb.Phase_PHASE_BUY, &pb.PlayerView{
		PlayerIdx: 0, Buys: 1, Coins: 8, Hand: nil,
	})
	act := SmithyBM{}.PickAction(cs)
	require.Equal(t, "province", act.GetBuyCard().GetCardId())
}

func TestSmithyBM_PickAction_BuyPhase_BuySmithyOnEarlyTurnIfUnderCap(t *testing.T) {
	cs := csWithMe(0, pb.Phase_PHASE_BUY, &pb.PlayerView{
		PlayerIdx: 0, Buys: 1, Coins: 4, Hand: nil,
	})
	// Supply shows full pile (10 remaining) → smithysOwned = 10 - 10 = 0, under cap.
	cs.Snapshot.Supply = []*pb.SupplyPile{{CardId: "smithy", Count: 10}}
	cs.MyTurnsTaken = 3
	act := SmithyBM{}.PickAction(cs)
	require.Equal(t, "smithy", act.GetBuyCard().GetCardId())
}

func TestSmithyBM_PickAction_BuyPhase_BuysDuchyInEndgame(t *testing.T) {
	cs := csWithMe(0, pb.Phase_PHASE_BUY, &pb.PlayerView{
		PlayerIdx: 0, Buys: 1, Coins: 5, Hand: nil,
	})
	// Province pile low → endgame; $5 should buy Duchy, not Silver.
	cs.Snapshot.Supply = []*pb.SupplyPile{
		{CardId: "province", Count: 3},
		{CardId: "smithy", Count: 8},
		{CardId: "duchy", Count: 4},
	}
	cs.MyTurnsTaken = 10 // past earlyTurn gate
	act := SmithyBM{}.PickAction(cs)
	require.Equal(t, "duchy", act.GetBuyCard().GetCardId())
}

func TestSmithyBM_PickAction_BuyPhase_PrefersSilverOverSmithyAfterTurn4(t *testing.T) {
	cs := csWithMe(0, pb.Phase_PHASE_BUY, &pb.PlayerView{
		PlayerIdx: 0, Buys: 1, Coins: 4, Hand: nil,
	})
	cs.MyTurnsTaken = 5
	act := SmithyBM{}.PickAction(cs)
	require.Equal(t, "silver", act.GetBuyCard().GetCardId())
}

func TestSmithyBM_PickAction_BuyPhase_StopsBuyingSmithyAtCap(t *testing.T) {
	cs := csWithMe(0, pb.Phase_PHASE_BUY, &pb.PlayerView{
		PlayerIdx: 0, Buys: 1, Coins: 4, Hand: nil,
	})
	// Supply shows 8 remaining → smithysOwned = 10 - 8 = 2, at cap
	cs.Snapshot.Supply = []*pb.SupplyPile{{CardId: "smithy", Count: 8}}
	cs.MyTurnsTaken = 3

	act := SmithyBM{}.PickAction(cs)
	require.Equal(t, "silver", act.GetBuyCard().GetCardId())
}

func TestSmithyBM_PickAction_BuyPhase_NoAffordableBuyEndsPhase(t *testing.T) {
	cs := csWithMe(0, pb.Phase_PHASE_BUY, &pb.PlayerView{
		PlayerIdx: 0, Buys: 1, Coins: 2, Hand: nil,
	})
	act := SmithyBM{}.PickAction(cs)
	require.NotNil(t, act.GetEndPhase())
}

func csWithMe(me int, phase pb.Phase, view *pb.PlayerView) *ClientState {
	cs := &ClientState{
		Me:            me,
		CurrentPlayer: me,
		Phase:         phase,
		Snapshot: &pb.GameStateSnapshot{
			CurrentPlayer: int32(me),
			Phase:         phase,
			Players:       []*pb.PlayerView{view},
		},
	}
	return cs
}
