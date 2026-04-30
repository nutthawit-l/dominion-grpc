package bot

import (
	"testing"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/stretchr/testify/require"
)

func TestSentryBM_Name(t *testing.T) {
	require.Equal(t, "sentry_bm", NewSentryBM().Name())
}

func TestSentryBM_PlaysSentryWhenInHand(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1,
		[]string{"sentry", "copper"})
	cs.MyPlayer().Actions = 1
	a := NewSentryBM().PickAction(cs)
	require.Equal(t, "sentry", a.GetPlayCard().CardId)
}

func TestSentryBM_BuysSentryAt5(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_BUY, 5, 1, nil)
	cs.Snapshot = snapshotWithSupply(map[string]int{
		"province": 8, "gold": 30, "silver": 40, "sentry": 10,
	})
	cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 5, Buys: 1}}
	a := NewSentryBM().PickAction(cs)
	require.Equal(t, "sentry", a.GetBuyCard().CardId)
}

func TestSentryBM_Resolve_TrashesCurseAndEstate(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1, nil)
	d := &pb.Decision{
		Id: "d1", PlayerIdx: 0, CardId: "sentry", Step: 0,
		Prompt: &pb.Decision_TrashFromRevealed{TrashFromRevealed: &pb.TrashFromRevealedPrompt{
			Cards: []string{"curse", "estate", "silver"},
		}},
	}
	r := NewSentryBM().Resolve(cs, d)
	require.NotNil(t, r.GetCardList())
	got := r.GetCardList().Cards
	require.Contains(t, got, "curse")
	require.Contains(t, got, "estate")
	require.NotContains(t, got, "silver")
}

func TestSentryBM_Resolve_DiscardFromRevealed_Refuses(t *testing.T) {
	cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1, nil)
	d := &pb.Decision{
		Id: "d1", PlayerIdx: 0, CardId: "sentry", Step: 1,
		Prompt: &pb.Decision_DiscardFromRevealed{DiscardFromRevealed: &pb.DiscardFromRevealedPrompt{
			Cards: []string{"silver", "gold"},
		}},
	}
	r := NewSentryBM().Resolve(cs, d)
	require.Empty(t, r.GetCardList().Cards, "discard nothing")
}
