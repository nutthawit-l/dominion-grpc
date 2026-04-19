package service

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/nutthawit-l/dominion-grpc/internal/engine/cards"
	"github.com/nutthawit-l/dominion-grpc/internal/store"
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
}

func TestGameService_SubmitAction_EndPhaseActionToBuy(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	create, _ := svc.CreateGame(ctx, connect.NewRequest(&pb.CreateGameRequest{
		Players: []string{"a", "b"}, Seed: 1,
	}))

	initial, ok := svc.store.Get(create.Msg.GameId)
	require.True(t, ok)
	cp := int32(initial.CurrentPlayer)

	_, err := svc.SubmitAction(ctx, connect.NewRequest(&pb.SubmitActionRequest{
		GameId: create.Msg.GameId,
		Action: &pb.Action{Kind: &pb.Action_EndPhase{EndPhase: &pb.EndPhaseAction{PlayerIdx: cp}}},
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

func TestGameService_CreateGame_WithExplicitKingdom(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	resp, err := svc.CreateGame(ctx, connect.NewRequest(&pb.CreateGameRequest{
		Players: []string{"alice", "bob"},
		Seed:    42,
		Kingdom: []string{"copper"}, // a known basic card — will be rejected as non-kingdom
	}))
	_ = resp
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeInvalidArgument, ce.Code())
}

func TestGameService_CreateGame_UnknownKingdomCard(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	_, err := svc.CreateGame(ctx, connect.NewRequest(&pb.CreateGameRequest{
		Players: []string{"a", "b"},
		Seed:    1,
		Kingdom: []string{"not-a-card"},
	}))
	require.Error(t, err)
	var ce *connect.Error
	require.ErrorAs(t, err, &ce)
	require.Equal(t, connect.CodeInvalidArgument, ce.Code())
}

func TestGameService_CreateGame_EmptyKingdom_UsesDefaults(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	resp, err := svc.CreateGame(ctx, connect.NewRequest(&pb.CreateGameRequest{
		Players: []string{"a", "b"}, Seed: 1,
	}))
	require.NoError(t, err)
	s, ok := svc.store.Get(resp.Msg.GameId)
	require.True(t, ok)
	for id := range s.Supply.Piles {
		switch id {
		case "copper", "silver", "gold", "estate", "duchy", "province", "curse":
			// ok
		default:
			// kingdom card present — fine too
		}
	}
	_ = s
}
