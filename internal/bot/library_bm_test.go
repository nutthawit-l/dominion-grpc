package bot

import (
	"testing"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/stretchr/testify/require"
)

func TestLibraryBM_Name(t *testing.T) {
	require.Equal(t, "library_bm", NewLibraryBM().Name())
}

func TestLibraryBM_PlaysLibraryWhenInHand(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1,
		[]string{"library", "copper"})
	cs.MyPlayer().Actions = 1

	a := NewLibraryBM().PickAction(cs)
	require.Equal(t, "library", a.GetPlayCard().CardId)
}

func TestLibraryBM_BuysLibraryAt5_FirstCopy(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 5, 1, nil)
	cs.Snapshot = snapshotWithSupply(map[string]int{
		"province": 8, "gold": 30, "silver": 40, "library": 10,
	})
	cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 5, Buys: 1}}

	a := NewLibraryBM().PickAction(cs)
	require.Equal(t, "library", a.GetBuyCard().CardId)
}

func TestLibraryBM_StopsBuyingLibrariesAfterTwo(t *testing.T) {
	s := NewLibraryBM()
	s.libraries = 2
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 5, 1, nil)
	cs.Snapshot = snapshotWithSupply(map[string]int{
		"province": 8, "gold": 30, "silver": 40, "library": 10,
	})
	cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 5, Buys: 1}}
	a := s.PickAction(cs)
	// Should fall through to Silver buy at 3+.
	require.Equal(t, "silver", a.GetBuyCard().CardId)
}

func TestLibraryBM_Resolve_SetAsidePrompt_DefaultsToKeep(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1, []string{"smithy"})
	d := &pb.Decision{
		Id: "d1", PlayerIdx: 0, CardId: "library",
		Prompt: &pb.Decision_SetAsideAction{SetAsideAction: &pb.SetAsideActionPrompt{CardId: "smithy"}},
	}
	r := NewLibraryBM().Resolve(cs, d)
	require.False(t, r.GetYesNo().Yes, "default = keep in hand")
}
