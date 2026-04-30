package bot

import (
	"testing"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/stretchr/testify/require"
)

func TestThroneRoomBM_Name(t *testing.T) {
	require.Equal(t, "throneroom_bm", NewThroneRoomBM().Name())
}

func TestThroneRoomBM_PlaysThroneRoomWhenWitchAlsoInHand(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1,
		[]string{"throne_room", "witch", "copper"})
	cs.MyPlayer().Actions = 1

	a := NewThroneRoomBM().PickAction(cs)
	require.Equal(t, "throne_room", a.GetPlayCard().CardId)
}

func TestThroneRoomBM_PlaysWitchWhenNoTRInHand(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1,
		[]string{"witch", "copper"})
	cs.MyPlayer().Actions = 1

	a := NewThroneRoomBM().PickAction(cs)
	require.Equal(t, "witch", a.GetPlayCard().CardId)
}

func TestThroneRoomBM_BuysWitchAt5(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 5, 1, nil)
	cs.MyTurnsTaken = 3
	cs.Snapshot = snapshotWithSupply(map[string]int{
		"province": 8, "gold": 30, "silver": 40, "witch": 10, "throne_room": 10,
	})
	cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 5, Buys: 1}}

	a := NewThroneRoomBM().PickAction(cs)
	require.Equal(t, "witch", a.GetBuyCard().CardId)
}

func TestThroneRoomBM_BuysThroneRoomAt4_OnlyIfOwnsWitch(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 4, 1, nil)
	cs.MyTurnsTaken = 3
	cs.Snapshot = snapshotWithSupply(map[string]int{
		"province": 8, "gold": 30, "silver": 40, "witch": 10, "throne_room": 10,
	})
	cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 4, Buys: 1}}
	s := NewThroneRoomBM()

	// Without owning a Witch: skip TR buy, fall through to Silver.
	a := s.PickAction(cs)
	require.Equal(t, "silver", a.GetBuyCard().CardId)

	// After owning a Witch (simulate via internal flag), TR buy fires.
	s.witchOwned = true
	a = s.PickAction(cs)
	require.Equal(t, "throne_room", a.GetBuyCard().CardId)
}

func TestThroneRoomBM_Resolve_ChooseAction_PrefersWitch(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1,
		[]string{"witch", "smithy"})

	d := &pb.Decision{
		Id: "d1", PlayerIdx: 0, CardId: "throne_room", Step: 0,
		Prompt: &pb.Decision_ChooseActionFromHand{ChooseActionFromHand: &pb.ChooseActionFromHandPrompt{}},
	}
	r := NewThroneRoomBM().Resolve(cs, d)
	require.Equal(t, "witch", r.GetCardChoice().Card)
}

func TestThroneRoomBM_Resolve_UnrecognizedPrompt_UsesSafeRefusal(t *testing.T) {
	cs := &ClientState{Me: 0, Snapshot: &pb.GameStateSnapshot{
		Players: []*pb.PlayerView{{PlayerIdx: 0, Hand: []string{"copper"}}},
	}}
	d := &pb.Decision{
		Id: "d2", PlayerIdx: 0,
		Prompt: &pb.Decision_TrashFromHand{TrashFromHand: &pb.TrashFromHandPrompt{Min: 0, Max: 1}},
	}
	r := NewThroneRoomBM().Resolve(cs, d)
	require.Equal(t, "d2", r.DecisionId)
}
