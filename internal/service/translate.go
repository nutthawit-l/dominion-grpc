package service

import (
	"fmt"

	pb "github.com/tie/dominion-grpc/gen/go/dominion/v1"
	"github.com/tie/dominion-grpc/internal/engine"
)

// ActionFromProto converts a proto Action union into the engine's
// internal Action type.
func ActionFromProto(a *pb.Action) (engine.Action, error) {
	if a == nil {
		return nil, fmt.Errorf("service: nil action")
	}
	switch k := a.Kind.(type) {
	case *pb.Action_PlayCard:
		return engine.PlayCard{
			PlayerIdx: int(k.PlayCard.PlayerIdx),
			Card:      engine.CardID(k.PlayCard.CardId),
		}, nil
	case *pb.Action_BuyCard:
		return engine.BuyCard{
			PlayerIdx: int(k.BuyCard.PlayerIdx),
			Card:      engine.CardID(k.BuyCard.CardId),
		}, nil
	case *pb.Action_EndPhase:
		return engine.EndPhase{PlayerIdx: int(k.EndPhase.PlayerIdx)}, nil
	case *pb.Action_Resolve:
		return engine.ResolveDecision{
			PlayerIdx:  int(k.Resolve.PlayerIdx),
			DecisionID: k.Resolve.DecisionId,
		}, nil
	default:
		return nil, fmt.Errorf("service: unknown action kind %T", k)
	}
}

// SnapshotFromState produces a proto snapshot scrubbed to the given
// viewer. The viewer sees their own hand contents; opponents' hand
// contents are omitted (but HandSize is still reported).
func SnapshotFromState(s *engine.GameState, viewer int) *pb.GameStateSnapshot {
	snap := &pb.GameStateSnapshot{
		GameId:        s.GameID,
		Seed:          s.Seed,
		Turn:          int32(s.Turn),
		CurrentPlayer: int32(s.CurrentPlayer),
		Phase:         phaseToProto(s.Phase),
		TrashSize:     int32(len(s.Trash)),
		Ended:         s.Ended,
	}
	for i, p := range s.Players {
		pv := &pb.PlayerView{
			PlayerIdx:   int32(i),
			Name:        p.Name,
			HandSize:    int32(len(p.Hand)),
			DeckSize:    int32(len(p.Deck)),
			DiscardSize: int32(len(p.Discard)),
			InPlaySize:  int32(len(p.InPlay)),
			Actions:     int32(p.Actions),
			Buys:        int32(p.Buys),
			Coins:       int32(p.Coins),
		}
		if i == viewer {
			for _, c := range p.Hand {
				pv.Hand = append(pv.Hand, string(c))
			}
		}
		for _, c := range p.InPlay {
			pv.InPlay = append(pv.InPlay, string(c))
		}
		snap.Players = append(snap.Players, pv)
	}
	for id, n := range s.Supply.Piles {
		snap.Supply = append(snap.Supply, &pb.SupplyPile{
			CardId: string(id),
			Count:  int32(n),
		})
	}
	for _, w := range s.Winners {
		snap.Winners = append(snap.Winners, int32(w))
	}
	return snap
}

func phaseToProto(p engine.Phase) pb.Phase {
	switch p {
	case engine.PhaseAction:
		return pb.Phase_PHASE_ACTION
	case engine.PhaseBuy:
		return pb.Phase_PHASE_BUY
	case engine.PhaseCleanup:
		return pb.Phase_PHASE_CLEANUP
	}
	return pb.Phase_PHASE_UNSPECIFIED
}
