package service

import (
	"context"
	"errors"
	"sync"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/nutthawit-l/dominion-grpc/internal/bot"
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

	// Spawn a server-side bot goroutine for any player named "bigmoney".
	for i, name := range names {
		if name == "bigmoney" {
			botIdx := i
			gameID := id
			go g.runBigMoneyBot(gameID, botIdx)
		}
	}

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

// playBotTurnBatch executes the bot's entire turn inside a single store lock,
// then emits exactly one fanOut event. This keeps SSE traffic to one update
// per bot turn so the frontend renders once instead of once per bot action.
func (g *GameService) playBotTurnBatch(gameID string, botIdx int, strat bot.BigMoney) {
	_ = g.store.WithLock(gameID, func(s *engine.GameState) error {
		if int(s.CurrentPlayer) != botIdx || s.Ended {
			return nil
		}
		var gameEnded bool
		for int(s.CurrentPlayer) == botIdx && !s.Ended {
			cs := &bot.ClientState{Me: botIdx}
			_ = cs.Apply(&pb.StreamGameEventsResponse{
				Kind: &pb.StreamGameEventsResponse_Snapshot{
					Snapshot: SnapshotFromState(s, engine.PlayerIdx(botIdx)),
				},
			})
			var pbAct *pb.Action
			if cs.PendingDecision != nil && int(cs.PendingDecision.PlayerIdx) == botIdx {
				pbAct = &pb.Action{Kind: &pb.Action_Resolve{Resolve: strat.Resolve(cs, cs.PendingDecision)}}
			} else {
				pbAct = strat.PickAction(cs)
			}
			if pbAct == nil {
				break
			}
			act, err := ActionFromProto(pbAct)
			if err != nil {
				break
			}
			_, events, err := engine.Apply(s, act, g.lookup)
			if err != nil {
				break
			}
			for _, ev := range events {
				if ev.Kind == engine.EventGameEnded {
					gameEnded = true
				}
			}
		}
		// Emit exactly one fanOut for the entire bot turn.
		var finalEvents []engine.Event
		if gameEnded {
			finalEvents = []engine.Event{{Kind: engine.EventGameEnded}}
		}
		g.fanOut(gameID, s, finalEvents)
		return nil
	})
}

// runBigMoneyBot subscribes to the game as botIdx, waits for its turns, and
// plays each turn atomically via playBotTurnBatch.
func (g *GameService) runBigMoneyBot(gameID string, botIdx int) {
	strat := bot.BigMoney{}
	cs := &bot.ClientState{Me: botIdx}

	ch := make(chan *pb.StreamGameEventsResponse, 64)
	sub := subscriber{ch: ch, viewer: engine.PlayerIdx(botIdx)}

	s, ok := g.store.Get(gameID)
	if !ok {
		return
	}
	initEvt := &pb.StreamGameEventsResponse{
		Sequence: 0,
		Kind:     &pb.StreamGameEventsResponse_Snapshot{Snapshot: SnapshotFromState(s, engine.PlayerIdx(botIdx))},
	}

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

	_ = cs.Apply(initEvt)
	if cs.IsMyTurn() {
		g.playBotTurnBatch(gameID, botIdx, strat)
	}

	for evt := range ch {
		if err := cs.Apply(evt); err != nil {
			continue
		}
		if cs.Ended {
			return
		}
		if cs.IsMyTurn() {
			g.playBotTurnBatch(gameID, botIdx, strat)
		}
	}
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
