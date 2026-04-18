package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// Strategy decides actions for a bot.
type Strategy interface {
	Name() string
	PickAction(cs *ClientState) *pb.Action
	Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision
}

// endPhase is a small constructor helper used by strategies.
func endPhase(me int) *pb.Action {
	return &pb.Action{Kind: &pb.Action_EndPhase{EndPhase: &pb.EndPhaseAction{PlayerIdx: int32(me)}}}
}

// playCard is a small constructor helper used by strategies.
func playCard(me int, card string) *pb.Action {
	return &pb.Action{Kind: &pb.Action_PlayCard{PlayCard: &pb.PlayCardAction{
		PlayerIdx: int32(me), CardId: card,
	}}}
}

// buyCard is a small constructor helper used by strategies.
func buyCard(me int, card string) *pb.Action {
	return &pb.Action{Kind: &pb.Action_BuyCard{BuyCard: &pb.BuyCardAction{
		PlayerIdx: int32(me), CardId: card,
	}}}
}

// safeRefusal returns the minimum legal answer for a decision, used
// by strategies that do not know how to handle a particular prompt.
// Tier 0 never triggers a decision, so this is a stub that returns a
// ResolveDecision echoing the id with no answer. Tier 2 will expand it.
func safeRefusal(d *pb.Decision) *pb.ResolveDecision {
	return &pb.ResolveDecision{DecisionId: d.Id, PlayerIdx: d.PlayerIdx}
}
