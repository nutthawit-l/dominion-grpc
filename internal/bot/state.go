package bot

import (
	"errors"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// ErrSequenceGap is returned by Apply when the incoming event's
// sequence number is not exactly LastSeq + 1.
var ErrSequenceGap = errors.New("bot: sequence gap — resubscribe required")

// ClientState is the bot's reduction of the event stream into a
// local view of the game.
type ClientState struct {
	Me              int
	LastSeq         uint64
	Snapshot        *pb.GameStateSnapshot
	Phase           pb.Phase
	Turn            int
	CurrentPlayer   int
	PendingDecision *pb.Decision
	DecidingPlayer  int
	Ended           bool
	Winners         []int
	MyTurnsTaken    int
	LastMyTurn      int
}

// Apply reduces one event into the client state.
func (cs *ClientState) Apply(ev *pb.StreamGameEventsResponse) error {
	if cs.LastSeq != 0 && ev.Sequence != cs.LastSeq+1 && ev.Sequence != 0 {
		return ErrSequenceGap
	}
	cs.LastSeq = ev.Sequence

	switch k := ev.Kind.(type) {
	case *pb.StreamGameEventsResponse_Snapshot:
		cs.Snapshot = k.Snapshot
		cs.Phase = k.Snapshot.Phase
		cs.Turn = int(k.Snapshot.Turn)
		cs.CurrentPlayer = int(k.Snapshot.CurrentPlayer)
		cs.PendingDecision = k.Snapshot.PendingDecision
		cs.Ended = k.Snapshot.Ended
		cs.Winners = nil
		for _, w := range k.Snapshot.Winners {
			cs.Winners = append(cs.Winners, int(w))
		}
		if cs.CurrentPlayer == cs.Me && cs.Turn != cs.LastMyTurn {
			cs.MyTurnsTaken++
			cs.LastMyTurn = cs.Turn
		}
	case *pb.StreamGameEventsResponse_PhaseChanged:
		cs.Phase = k.PhaseChanged.NewPhase
	case *pb.StreamGameEventsResponse_TurnStarted:
		cs.Turn = int(k.TurnStarted.Turn)
		cs.CurrentPlayer = int(k.TurnStarted.PlayerIdx)
		cs.PendingDecision = nil
	case *pb.StreamGameEventsResponse_Decision:
		cs.PendingDecision = k.Decision.Decision
		cs.DecidingPlayer = int(k.Decision.PlayerIdx)
	case *pb.StreamGameEventsResponse_Ended:
		cs.Ended = true
	}
	return nil
}

// IsMyTurn reports whether it is the receiver's turn AND the game has
// not ended.
func (cs *ClientState) IsMyTurn() bool {
	return cs.CurrentPlayer == cs.Me && !cs.Ended
}

// MyPlayer returns the PlayerView for the receiver from the last snapshot.
func (cs *ClientState) MyPlayer() *pb.PlayerView {
	if cs.Snapshot == nil {
		return nil
	}
	for _, p := range cs.Snapshot.Players {
		if int(p.PlayerIdx) == cs.Me {
			return p
		}
	}
	return nil
}
