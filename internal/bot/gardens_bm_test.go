package bot

import (
	"testing"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/stretchr/testify/require"
)

func TestGardensBM_Name(t *testing.T) {
	require.Equal(t, "gardens_bm", NewGardensBM().Name())
}

func TestGardensBM_ActionPhase_AlwaysEndsPhase(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1,
		[]string{"copper", "estate"})
	a := NewGardensBM().PickAction(cs)
	require.NotNil(t, a.GetEndPhase(),
		"Gardens is Victory-only — nothing to play")
}

func TestGardensBM_BuysGardensAt4_FirstCopy(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 4, 1, nil)
	cs.Snapshot = snapshotWithSupply(map[string]int{
		"province": 8, "gold": 30, "silver": 40, "gardens": 8, "copper": 40,
	})
	cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 4, Buys: 1}}
	a := NewGardensBM().PickAction(cs)
	require.Equal(t, "gardens", a.GetBuyCard().CardId)
}

func TestGardensBM_StopsBuyingGardensAfterFour(t *testing.T) {
	s := NewGardensBM()
	s.gardens = 4
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 4, 1, nil)
	cs.Snapshot = snapshotWithSupply(map[string]int{
		"province": 8, "gold": 30, "silver": 40, "gardens": 8, "copper": 40,
	})
	cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 4, Buys: 1}}
	a := s.PickAction(cs)
	// 4-cost slot empty after Gardens cap; falls through to Silver at 3+.
	require.Equal(t, "silver", a.GetBuyCard().CardId)
}

func TestGardensBM_BuysProvinceAtEight(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 8, 1, nil)
	cs.Snapshot = snapshotWithSupply(map[string]int{
		"province": 8, "gold": 30, "silver": 40, "gardens": 8, "copper": 40,
	})
	cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 8, Buys: 1}}
	a := NewGardensBM().PickAction(cs)
	require.Equal(t, "province", a.GetBuyCard().CardId)
}

func TestGardensBM_BulksWithCopper(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 0, 1, nil)
	cs.Snapshot = snapshotWithSupply(map[string]int{
		"province": 8, "gold": 30, "silver": 40, "gardens": 8, "copper": 40,
	})
	cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 0, Buys: 1}}
	a := NewGardensBM().PickAction(cs)
	require.Equal(t, "copper", a.GetBuyCard().CardId)
}
