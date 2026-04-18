package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

func csWithHand(me int, phase pb.Phase, coins int32, buys int32, hand []string) *ClientState {
	return &ClientState{
		Me:            me,
		CurrentPlayer: me,
		Phase:         phase,
		Snapshot: &pb.GameStateSnapshot{
			Phase:         phase,
			CurrentPlayer: int32(me),
			Players: []*pb.PlayerView{
				{PlayerIdx: int32(me), Hand: hand, Coins: coins, Buys: buys},
			},
		},
	}
}

func TestBigMoney_ActionPhase_EndsPhase(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1, []string{"copper"})
	a := (BigMoney{}).PickAction(cs)
	require.NotNil(t, a.GetEndPhase())
}

func TestBigMoney_BuyPhase_PlaysTreasuresFirst(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 0, 1, []string{"estate", "copper"})
	a := (BigMoney{}).PickAction(cs)
	require.NotNil(t, a.GetPlayCard())
	require.Equal(t, "copper", a.GetPlayCard().CardId)
}

func TestBigMoney_BuyPhase_BuysProvinceAtEightCoins(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 8, 1, []string{})
	a := (BigMoney{}).PickAction(cs)
	require.NotNil(t, a.GetBuyCard())
	require.Equal(t, "province", a.GetBuyCard().CardId)
}

func TestBigMoney_BuyPhase_BuysGoldAtSixCoins(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 6, 1, []string{})
	a := (BigMoney{}).PickAction(cs)
	require.Equal(t, "gold", a.GetBuyCard().CardId)
}

func TestBigMoney_BuyPhase_BuysSilverAtThreeCoins(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 3, 1, []string{})
	a := (BigMoney{}).PickAction(cs)
	require.Equal(t, "silver", a.GetBuyCard().CardId)
}

func TestBigMoney_BuyPhase_EndsPhaseWhenBroke(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 2, 1, []string{})
	a := (BigMoney{}).PickAction(cs)
	require.NotNil(t, a.GetEndPhase())
}
