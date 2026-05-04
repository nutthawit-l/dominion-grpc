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
		put := d.GetPutOnDeck()
		if len(put.TypeFilter) > 0 {
			card := firstInHandMatching(cs, put.TypeFilter)
			r.Answer = &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{Cards: []string{card}}}
		} else {
			card := firstInHand(cs)
			r.Answer = &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{Card: card}}
		}
	case *pb.Decision_TrashFromRevealed:
		tfr := d.GetTrashFromRevealed()
		if len(tfr.Cards) == 0 {
			r.Answer = &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{None: true}}
			break
		}
		best := tfr.Cards[0]
		bestCost := cardCost(best)
		for _, c := range tfr.Cards[1:] {
			if cost := cardCost(c); cost < bestCost {
				best, bestCost = c, cost
			}
		}
		r.Answer = &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{Card: best}}
	case *pb.Decision_MayPlayAction:
		r.Answer = &pb.ResolveDecision_YesNo{YesNo: &pb.YesNoAnswer{Yes: false}}
	case *pb.Decision_ChooseActionFromHand:
		// Engine guarantees ≥ 1 Action in hand; pick the first Action we see.
		me := cs.MyPlayer()
		pick := ""
		if me != nil {
			for _, c := range me.Hand {
				if isAction(c) {
					pick = c
					break
				}
			}
			if pick == "" && len(me.Hand) > 0 {
				pick = me.Hand[0] // fallback — shouldn't be reached.
			}
		}
		r.Answer = &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{Card: pick}}
	case *pb.Decision_SetAsideAction:
		// Default: keep in hand (don't set aside) — preserves drawn cards.
		r.Answer = &pb.ResolveDecision_YesNo{YesNo: &pb.YesNoAnswer{Yes: false}}
	case *pb.Decision_DiscardFromRevealed:
		// Default: discard nothing.
		r.Answer = &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{Cards: nil}}
	case *pb.Decision_ReorderCards:
		// Default: identity reorder — return the cards in the order received.
		p := d.GetReorderCards()
		cards := append([]string(nil), p.Cards...)
		r.Answer = &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{Cards: cards}}
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

// firstInHandMatching returns the first card in hand matching any of the
// given type filters. If filter is empty, returns the first card. If no
// match, returns the first card as a fallback.
func firstInHandMatching(cs *ClientState, filter []pb.CardType) string {
	me := cs.MyPlayer()
	if me == nil || len(me.Hand) == 0 {
		return ""
	}
	if len(filter) == 0 {
		return me.Hand[0]
	}
	for _, c := range me.Hand {
		if cardHasAnyType(c, filter) {
			return c
		}
	}
	return me.Hand[0] // fallback
}

func cardHasAnyType(id string, filter []pb.CardType) bool {
	for _, t := range filter {
		if t == pb.CardType_CARD_TYPE_VICTORY && isVictory(id) {
			return true
		}
		if t == pb.CardType_CARD_TYPE_TREASURE && isTreasure(id) {
			return true
		}
	}
	return false
}

func isVictory(id string) bool {
	switch id {
	case "estate", "duchy", "province", "gardens":
		return true
	}
	return false
}

// isAction reports whether the given card ID is an Action card. The
// list mirrors the kingdom cards registered by Tier 0 through Tier 4
// — bots compile-pin this rather than importing the engine.
func isAction(id string) bool {
	switch id {
	case "smithy", "village", "festival", "laboratory", "market",
		"council_room", "moat", "harbinger", "vassal", "workshop",
		"moneylender", "poacher", "remodel", "mine", "artisan",
		"cellar", "chapel", "witch", "militia", "bureaucrat",
		"bandit", "throne_room", "library", "sentry", "merchant":
		return true
	}
	return false
}

// cardCost returns the known cost of a card by ID. This is a simple
// lookup for the base set; it avoids importing the engine package.
func cardCost(id string) int {
	switch id {
	case "copper", "curse":
		return 0
	case "estate", "moat":
		return 2
	case "silver", "cellar", "chapel", "village", "merchant":
		return 3
	case "harbinger", "vassal", "workshop":
		return 3
	case "militia", "bureaucrat", "moneylender", "poacher", "remodel", "smithy", "throne_room", "gardens":
		return 4
	case "mine", "witch", "bandit", "laboratory", "market", "festival", "library", "sentry":
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
