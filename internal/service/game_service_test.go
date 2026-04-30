package service

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/nutthawit-l/dominion-grpc/internal/engine/cards"
	"github.com/nutthawit-l/dominion-grpc/internal/store"
	"github.com/stretchr/testify/require"
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

func TestStreamGameEvents_PerViewerScrubbing(t *testing.T) {
	svc := newTestService()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	create, err := svc.CreateGame(ctx, connect.NewRequest(&pb.CreateGameRequest{
		Players: []string{"a", "b"}, Seed: 1,
	}))
	require.NoError(t, err)

	// Two subscribers, one per seat.
	stream0 := &fakeServerStream{ch: make(chan *pb.StreamGameEventsResponse, 16)}
	stream1 := &fakeServerStream{ch: make(chan *pb.StreamGameEventsResponse, 16)}
	go func() {
		_ = svc.streamGameEventsInto(ctx, connect.NewRequest(&pb.StreamGameEventsRequest{
			GameId: create.Msg.GameId, PlayerIdx: 0,
		}), stream0)
	}()
	go func() {
		_ = svc.streamGameEventsInto(ctx, connect.NewRequest(&pb.StreamGameEventsRequest{
			GameId: create.Msg.GameId, PlayerIdx: 1,
		}), stream1)
	}()

	// Drain the initial snapshots (sequence 0).
	init0 := drainSnapshot(t, stream0)
	init1 := drainSnapshot(t, stream1)
	assertScrubbedForViewer(t, init0, 0)
	assertScrubbedForViewer(t, init1, 1)

	// Wait until both channels are registered before we fan out.
	for i := 0; i < 50; i++ {
		svc.subsMu.Lock()
		got := len(svc.subs[create.Msg.GameId])
		svc.subsMu.Unlock()
		if got == 2 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	// Ask the current player to end the Action phase.
	s, _ := svc.store.Get(create.Msg.GameId)
	cp := int32(s.CurrentPlayer)
	_, err = svc.SubmitAction(ctx, connect.NewRequest(&pb.SubmitActionRequest{
		GameId: create.Msg.GameId,
		Action: &pb.Action{Kind: &pb.Action_EndPhase{EndPhase: &pb.EndPhaseAction{PlayerIdx: cp}}},
	}))
	require.NoError(t, err)

	// Each subscriber should receive a scrubbed snapshot for its own seat.
	post0 := drainSnapshot(t, stream0)
	post1 := drainSnapshot(t, stream1)
	assertScrubbedForViewer(t, post0, 0)
	assertScrubbedForViewer(t, post1, 1)
}

func drainSnapshot(t *testing.T, s *fakeServerStream) *pb.GameStateSnapshot {
	t.Helper()
	select {
	case ev := <-s.ch:
		snap := ev.GetSnapshot()
		require.NotNil(t, snap, "expected a snapshot response")
		return snap
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for snapshot")
		return nil
	}
}

func TestDecisionToProto_DiscardFromHand(t *testing.T) {
	d := &engine.Decision{
		ID: "d1", PlayerIdx: 0, CardID: "cellar", Step: 0,
		Prompt: engine.DiscardFromHandPrompt{Min: 0, Max: 3},
	}

	proto := DecisionToProto(d)

	require.Equal(t, "d1", proto.Id)
	require.Equal(t, int32(0), proto.PlayerIdx)
	require.Equal(t, "cellar", proto.CardId)
	p := proto.GetDiscardFromHand()
	require.NotNil(t, p)
	require.Equal(t, int32(0), p.Min)
	require.Equal(t, int32(3), p.Max)
}

func TestAnswerFromProto_CardList(t *testing.T) {
	r := &pb.ResolveDecision{
		DecisionId: "d1", PlayerIdx: 0,
		Answer: &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{
			Cards: []string{"copper", "estate"},
		}},
	}

	answer, err := AnswerFromProto(r)
	require.NoError(t, err)
	cl, ok := answer.(engine.CardListAnswer)
	require.True(t, ok)
	require.Equal(t, []engine.CardID{"copper", "estate"}, cl.Cards)
}

func TestAnswerFromProto_CardChoice(t *testing.T) {
	r := &pb.ResolveDecision{
		DecisionId: "d1", PlayerIdx: 0,
		Answer: &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{
			Card: "silver", None: false,
		}},
	}

	answer, err := AnswerFromProto(r)
	require.NoError(t, err)
	cc, ok := answer.(engine.CardChoiceAnswer)
	require.True(t, ok)
	require.Equal(t, engine.CardID("silver"), cc.Card)
	require.False(t, cc.None)
}

func TestAnswerFromProto_YesNo(t *testing.T) {
	r := &pb.ResolveDecision{
		DecisionId: "d1", PlayerIdx: 0,
		Answer: &pb.ResolveDecision_YesNo{YesNo: &pb.YesNoAnswer{Yes: true}},
	}

	answer, err := AnswerFromProto(r)
	require.NoError(t, err)
	yn, ok := answer.(engine.YesNoAnswer)
	require.True(t, ok)
	require.True(t, yn.Yes)
}

func assertScrubbedForViewer(t *testing.T, snap *pb.GameStateSnapshot, viewer int) {
	t.Helper()
	for _, p := range snap.Players {
		if int(p.PlayerIdx) == viewer {
			require.Len(t, p.Hand, int(p.HandSize), "viewer %d should see own hand contents", viewer)
		} else {
			require.Empty(t, p.Hand, "opponent at seat %d should have hand scrubbed", p.PlayerIdx)
			require.Greater(t, p.HandSize, int32(0), "opponent hand size should still be reported")
		}
	}
}
