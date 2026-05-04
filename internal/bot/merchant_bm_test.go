package bot

import (
	"testing"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/stretchr/testify/require"
)

func TestMerchantBM_Name(t *testing.T) {
	require.Equal(t, "merchant_bm", NewMerchantBM().Name())
}

func TestMerchantBM_PlaysMerchantWhenInHand(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1,
		[]string{"merchant", "copper"})
	cs.MyPlayer().Actions = 1
	a := NewMerchantBM().PickAction(cs)
	require.Equal(t, "merchant", a.GetPlayCard().CardId)
}

func TestMerchantBM_NoMerchantInHand_EndsActionPhase(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1,
		[]string{"copper", "estate"})
	cs.MyPlayer().Actions = 1
	a := NewMerchantBM().PickAction(cs)
	require.NotNil(t, a.GetEndPhase(),
		"no Merchant in hand → end action phase")
}

func TestMerchantBM_BuysMerchantAt3_FirstCopy(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 3, 1, nil)
	cs.Snapshot = snapshotWithSupply(map[string]int{
		"province": 8, "gold": 30, "silver": 40, "merchant": 10,
	})
	cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 3, Buys: 1}}
	a := NewMerchantBM().PickAction(cs)
	require.Equal(t, "merchant", a.GetBuyCard().CardId,
		"at coins=3 with no Merchants owned, prefer Merchant over Silver")
}

func TestMerchantBM_StopsBuyingAfterTwo(t *testing.T) {
	s := NewMerchantBM()
	s.merchants = 2
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 3, 1, nil)
	cs.Snapshot = snapshotWithSupply(map[string]int{
		"province": 8, "gold": 30, "silver": 40, "merchant": 10,
	})
	cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 3, Buys: 1}}
	a := s.PickAction(cs)
	require.Equal(t, "silver", a.GetBuyCard().CardId,
		"with 2 Merchants owned, fall through to Silver")
}

func TestMerchantBM_BuysProvinceAtEight(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 8, 1, nil)
	cs.Snapshot = snapshotWithSupply(map[string]int{
		"province": 8, "gold": 30, "silver": 40, "merchant": 10,
	})
	cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 8, Buys: 1}}
	a := NewMerchantBM().PickAction(cs)
	require.Equal(t, "province", a.GetBuyCard().CardId)
}
