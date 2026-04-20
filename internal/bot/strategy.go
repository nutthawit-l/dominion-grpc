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

// safeRefusal returns the minimum legal answer for a decision. Used by
// strategies that do not handle a particular prompt type.
func safeRefusal(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	r := &pb.ResolveDecision{DecisionId: d.Id, PlayerIdx: d.PlayerIdx}
	switch d.Prompt.(type) {
	case *pb.Decision_DiscardFromHand:
		p := d.GetDiscardFromHand()
		cards := pickFirstN(cs, int(p.Min))
		r.Answer = &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{Cards: cards}}
	case *pb.Decision_TrashFromHand:
		r.Answer = &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{Cards: nil}}
	case *pb.Decision_GainFromSupply:
		card := cheapestInSupply(cs, d.GetGainFromSupply())
		r.Answer = &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{Card: card}}
	case *pb.Decision_ChooseFromDiscard:
		r.Answer = &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{None: true}}
	case *pb.Decision_PutOnDeck:
		card := firstInHand(cs)
		r.Answer = &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{Card: card}}
	case *pb.Decision_MayPlayAction:
		r.Answer = &pb.ResolveDecision_YesNo{YesNo: &pb.YesNoAnswer{Yes: false}}
	default:
		r.Answer = &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{}}
	}
	return r
}

func pickFirstN(cs *ClientState, n int) []string {
	me := cs.MyPlayer()
	if me == nil || n <= 0 {
		return nil
	}
	if n > len(me.Hand) {
		n = len(me.Hand)
	}
	return me.Hand[:n]
}

func cheapestInSupply(cs *ClientState, p *pb.GainFromSupplyPrompt) string {
	if cs.Snapshot == nil {
		return "copper"
	}
	best := ""
	bestCost := int(p.MaxCost) + 1
	for _, pile := range cs.Snapshot.Supply {
		if pile.Count <= 0 {
			continue
		}
		cost := cardCost(pile.CardId)
		if cost <= int(p.MaxCost) && cost < bestCost {
			best = pile.CardId
			bestCost = cost
		}
	}
	if best == "" {
		return "copper"
	}
	return best
}

func firstInHand(cs *ClientState) string {
	me := cs.MyPlayer()
	if me == nil || len(me.Hand) == 0 {
		return ""
	}
	return me.Hand[0]
}

// cardCost returns the known cost of a card by ID. This is a simple
// lookup for the base set; it avoids importing the engine package.
func cardCost(id string) int {
	switch id {
	case "copper", "curse":
		return 0
	case "estate":
		return 2
	case "silver", "cellar", "chapel":
		return 3
	case "harbinger", "vassal", "workshop":
		return 3
	case "moneylender", "poacher", "remodel", "smithy":
		return 4
	case "mine", "laboratory", "market", "festival":
		return 5
	case "gold", "artisan", "council_room":
		return 6
	case "duchy":
		return 5
	case "province":
		return 8
	}
	return 0
}
