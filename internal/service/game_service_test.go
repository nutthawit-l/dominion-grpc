package service

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	pb "github.com/tie/dominion-grpc/gen/go/dominion/v1"
	"github.com/tie/dominion-grpc/internal/engine"
	"github.com/tie/dominion-grpc/internal/engine/cards"
	"github.com/tie/dominion-grpc/internal/store"
)

func newTestService() *GameService {
	return NewGameService(
		store.NewMemory(),
		func(id engine.CardID) (*engine.Card, bool) {
			return cards.DefaultRegistry.Lookup(id)
		},
	)
}

func TestGameService_CreateGame(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	resp, err := svc.CreateGame(ctx, connect.NewRequest(&pb.CreateGameRequest{
		Players: []string{"alice", "bob"},
		Seed:    42,
	}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.GameId)
	require.NotNil(t, resp.Msg.Snapshot)
	require.Equal(t, pb.Phase_PHASE_ACTION, resp.Msg.Snapshot.Phase)
}

func TestGameService_SubmitAction_EndPhaseActionToBuy(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	create, _ := svc.CreateGame(ctx, connect.NewRequest(&pb.CreateGameRequest{
		Players: []string{"a", "b"}, Seed: 1,
	}))

	_, err := svc.SubmitAction(ctx, connect.NewRequest(&pb.SubmitActionRequest{
		GameId: create.Msg.GameId,
		Action: &pb.Action{Kind: &pb.Action_EndPhase{EndPhase: &pb.EndPhaseAction{PlayerIdx: 0}}},
	}))
	require.NoError(t, err)

	// Verify state advanced.
	s, ok := svc.store.Get(create.Msg.GameId)
	require.True(t, ok)
	require.Equal(t, engine.PhaseBuy, s.Phase)
}

func TestGameService_SubmitAction_UnknownGameReturnsNotFound(t *testing.T) {
	svc := newTestService()
	_, err := svc.SubmitAction(context.Background(), connect.NewRequest(&pb.SubmitActionRequest{
		GameId: "nope",
		Action: &pb.Action{Kind: &pb.Action_EndPhase{EndPhase: &pb.EndPhaseAction{PlayerIdx: 0}}},
	}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeNotFound, ce.Code())
}

func TestGameService_StreamGameEvents_FirstEventIsSnapshot(t *testing.T) {
	svc := newTestService()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	create, _ := svc.CreateGame(ctx, connect.NewRequest(&pb.CreateGameRequest{
		Players: []string{"a", "b"}, Seed: 1,
	}))

	stream := &fakeServerStream{ch: make(chan *pb.StreamGameEventsResponse, 16)}
	go func() {
		_ = svc.streamGameEventsInto(ctx, connect.NewRequest(&pb.StreamGameEventsRequest{
			GameId: create.Msg.GameId, PlayerIdx: 0,
		}), stream)
	}()

	// First event should be a snapshot.
	ev := <-stream.ch
	require.NotNil(t, ev.GetSnapshot())
	require.Equal(t, uint64(0), ev.Sequence)
}

// fakeServerStream is a minimal stand-in for connect.ServerStream[...]
// that captures sent messages onto a channel.
type fakeServerStream struct {
	ch chan *pb.StreamGameEventsResponse
}

func (f *fakeServerStream) Send(ev *pb.StreamGameEventsResponse) error {
	f.ch <- ev
	return nil
}
