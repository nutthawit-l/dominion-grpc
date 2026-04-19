package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

func TestClientState_Apply_Snapshot(t *testing.T) {
	cs := &ClientState{Me: 0}
	ev := &pb.StreamGameEventsResponse{
		Sequence: 0,
		Kind: &pb.StreamGameEventsResponse_Snapshot{Snapshot: &pb.GameStateSnapshot{
			GameId:        "g",
			Turn:          1,
			CurrentPlayer: 0,
			Phase:         pb.Phase_PHASE_ACTION,
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Hand: []string{"copper", "copper", "copper", "estate", "estate"}},
				{PlayerIdx: 1, HandSize: 5},
			},
		}},
	}
	require.NoError(t, cs.Apply(ev))
	require.Equal(t, int32(1), cs.Snapshot.Turn)
	require.Equal(t, 0, cs.CurrentPlayer)
	require.Equal(t, pb.Phase_PHASE_ACTION, cs.Phase)
}

func TestClientState_Apply_SequenceGap(t *testing.T) {
	cs := &ClientState{Me: 0, LastSeq: 5}
	ev := &pb.StreamGameEventsResponse{
		Sequence: 7, // gap — expected 6
		Kind:     &pb.StreamGameEventsResponse_Snapshot{Snapshot: &pb.GameStateSnapshot{}},
	}
	err := cs.Apply(ev)
	require.ErrorIs(t, err, ErrSequenceGap)
}

func TestClientState_IsMyTurn(t *testing.T) {
	cs := &ClientState{Me: 0, CurrentPlayer: 0}
	require.True(t, cs.IsMyTurn())
	cs.CurrentPlayer = 1
	require.False(t, cs.IsMyTurn())
	cs.CurrentPlayer = 0
	cs.Ended = true
	require.False(t, cs.IsMyTurn())
}

func TestClientState_MyTurnsTaken_IncrementsOnMyTurnSnapshots(t *testing.T) {
	cs := &ClientState{Me: 0}

	require.NoError(t, cs.Apply(snapshot(0, 1, 0)))
	require.Equal(t, 1, cs.MyTurnsTaken)

	require.NoError(t, cs.Apply(snapshot(1, 1, 0)))
	require.Equal(t, 1, cs.MyTurnsTaken)

	require.NoError(t, cs.Apply(snapshot(2, 1, 1)))
	require.Equal(t, 1, cs.MyTurnsTaken)

	require.NoError(t, cs.Apply(snapshot(3, 2, 0)))
	require.Equal(t, 2, cs.MyTurnsTaken)
}

func TestClientState_MyTurnsTaken_StaysZeroIfStartingPlayerIsOpponent(t *testing.T) {
	cs := &ClientState{Me: 0}
	require.NoError(t, cs.Apply(snapshot(0, 1, 1)))
	require.Equal(t, 0, cs.MyTurnsTaken)
}

func snapshot(seq uint64, turn, currentPlayer int32) *pb.StreamGameEventsResponse {
	return &pb.StreamGameEventsResponse{
		Sequence: seq,
		Kind: &pb.StreamGameEventsResponse_Snapshot{Snapshot: &pb.GameStateSnapshot{
			Turn: turn, CurrentPlayer: currentPlayer,
		}},
	}
}
