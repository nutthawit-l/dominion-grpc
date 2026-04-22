package service

import (
	"context"
	"errors"
	"sync"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/nutthawit-l/dominion-grpc/internal/store"
)

// subscriber carries the per-stream metadata used during fanout.
type subscriber struct {
	ch     chan *pb.StreamGameEventsResponse
	viewer engine.PlayerIdx
}

// GameService implements the Connect GameServiceHandler interface.
type GameService struct {
	store  *store.Memory
	lookup engine.CardLookup

	subsMu sync.Mutex
	// subs maps game id -> slice of subscribers. Each StreamGameEvents call
	// appends one subscriber; SubmitAction fans out per-viewer snapshots to
	// every subscriber for that game.
	subs map[string][]subscriber
	// seq is the per-game monotonic event sequence counter.
	seq map[string]uint64
}

// NewGameService constructs a service backed by the given store and
// card lookup.
func NewGameService(s *store.Memory, lookup engine.CardLookup) *GameService {
	return &GameService{
		store:  s,
		lookup: lookup,
		subs:   map[string][]subscriber{},
		seq:    map[string]uint64{},
	}
}

// CreateGame handles the CreateGame RPC.
func (g *GameService) CreateGame(ctx context.Context, req *connect.Request[pb.CreateGameRequest]) (*connect.Response[pb.CreateGameResponse], error) {
	names := req.Msg.Players
	if len(names) != 2 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("expected exactly 2 players in Tier 0"))
	}
	kingdom := make([]engine.CardID, len(req.Msg.Kingdom))
	for i, k := range req.Msg.Kingdom {
		kingdom[i] = engine.CardID(k)
	}
	id := uuid.NewString()
	gs, err := engine.NewGame(id, names, kingdom, req.Msg.Seed, g.lookup)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	g.store.Put(gs)
	return connect.NewResponse(&pb.CreateGameResponse{
		GameId: id,
	}), nil
}

// SubmitAction handles the SubmitAction RPC.
func (g *GameService) SubmitAction(ctx context.Context, req *connect.Request[pb.SubmitActionRequest]) (*connect.Response[pb.SubmitActionResponse], error) {
	act, err := ActionFromProto(req.Msg.Action)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	err = g.store.WithLock(req.Msg.GameId, func(s *engine.GameState) error {
		_, events, applyErr := engine.Apply(s, act, g.lookup)
		if applyErr != nil {
			return applyErr
		}
		g.fanOut(req.Msg.GameId, s, events)
		return nil
	})
	if errors.Is(err, store.ErrGameNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&pb.SubmitActionResponse{}), nil
}

// fanOut sends one snapshot per SubmitAction call to every subscriber.
// Sending a single snapshot per action (rather than one per engine event)
// prevents bots from acting on stale intermediate events that all say
// "it's your turn." The game-ended event is sent as a structured message
// so subscribers can detect termination without polling.
// Must be called with the store mutex held (which it already is from WithLock).
func (g *GameService) fanOut(gameID string, s *engine.GameState, events []engine.Event) {
	g.subsMu.Lock()
	defer g.subsMu.Unlock()

	// Check whether the game ended so we send the right final event.
	gameEnded := false
	for _, ev := range events {
		if ev.Kind == engine.EventGameEnded {
			gameEnded = true
		}
	}

	g.seq[gameID]++
	seq := g.seq[gameID]

	if gameEnded {
		resp := &pb.StreamGameEventsResponse{
			Sequence: seq,
			Kind:     &pb.StreamGameEventsResponse_Ended{Ended: &pb.GameEnded{}},
		}
		for _, sub := range g.subs[gameID] {
			select {
			case sub.ch <- resp:
			default:
				// Drop on slow consumer.
			}
		}
		return
	}

	// Build one snapshot response per distinct viewer seat, then send each
	// subscriber the snapshot for their own seat.
	snapByViewer := map[engine.PlayerIdx]*pb.StreamGameEventsResponse{}
	for _, sub := range g.subs[gameID] {
		if _, ok := snapByViewer[sub.viewer]; ok {
			continue
		}
		snapByViewer[sub.viewer] = &pb.StreamGameEventsResponse{
			Sequence: seq,
			Kind:     &pb.StreamGameEventsResponse_Snapshot{Snapshot: SnapshotFromState(s, sub.viewer)},
		}
	}
	for _, sub := range g.subs[gameID] {
		resp := snapByViewer[sub.viewer]
		select {
		case sub.ch <- resp:
		default:
			// Drop on slow consumer.
		}
	}
}

// eventSink is the minimal interface StreamGameEvents depends on, so
// tests can inject a fake without constructing a real connect stream.
type eventSink interface {
	Send(*pb.StreamGameEventsResponse) error
}

// StreamGameEvents handles the StreamGameEvents server-streaming RPC.
func (g *GameService) StreamGameEvents(ctx context.Context, req *connect.Request[pb.StreamGameEventsRequest], stream *connect.ServerStream[pb.StreamGameEventsResponse]) error {
	return g.streamGameEventsInto(ctx, req, stream)
}

func (g *GameService) streamGameEventsInto(ctx context.Context, req *connect.Request[pb.StreamGameEventsRequest], sink eventSink) error {
	gameID := req.Msg.GameId
	viewer := engine.PlayerIdx(req.Msg.PlayerIdx)

	// Send an initial snapshot so the client can render the board immediately.
	s, ok := g.store.Get(gameID)
	if !ok {
		return connect.NewError(connect.CodeNotFound, store.ErrGameNotFound)
	}
	if err := sink.Send(&pb.StreamGameEventsResponse{
		Sequence: 0,
		Kind:     &pb.StreamGameEventsResponse_Snapshot{Snapshot: SnapshotFromState(s, viewer)},
	}); err != nil {
		return err
	}

	// Register subscriber channel.
	ch := make(chan *pb.StreamGameEventsResponse, 64)
	sub := subscriber{ch: ch, viewer: viewer}
	g.subsMu.Lock()
	g.subs[gameID] = append(g.subs[gameID], sub)
	g.subsMu.Unlock()

	defer func() {
		g.subsMu.Lock()
		subs := g.subs[gameID]
		for i, s := range subs {
			if s.ch == ch {
				g.subs[gameID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		g.subsMu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			if err := sink.Send(ev); err != nil {
				return err
			}
		}
	}
}
