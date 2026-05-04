package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// RemodelBM is Remodel + Big Money. Buys 1 Remodel early, remodels
// Estates into better cards, otherwise follows Big Money.
type RemodelBM struct {
	remodelOwned bool
}

func NewRemodelBM() *RemodelBM {
	return &RemodelBM{}
}

func (r *RemodelBM) Name() string { return "remodel_bm" }

func (r *RemodelBM) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}

	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		if me.Actions >= 1 && handContains(me.Hand, "remodel") {
			return playCard(cs.Me, "remodel")
		}
		return endPhase(cs.Me)

	case pb.Phase_PHASE_BUY:
		for _, card := range me.Hand {
			if isTreasure(card) {
				return playCard(cs.Me, card)
			}
		}
		if me.Buys <= 0 {
			return endPhase(cs.Me)
		}
		provincesLeft := supplyCount(cs.Snapshot, "province")
		endgame := provincesLeft <= 4
		switch {
		case me.Coins >= 8 && provincesLeft > 0:
			return buyCard(cs.Me, "province")
		case me.Coins >= 6 && supplyCount(cs.Snapshot, "gold") > 0:
			return buyCard(cs.Me, "gold")
		case me.Coins >= 5 && endgame && supplyCount(cs.Snapshot, "duchy") > 0:
			return buyCard(cs.Me, "duchy")
		case me.Coins >= 4 && !r.remodelOwned && cs.MyTurnsTaken <= 4 && supplyCount(cs.Snapshot, "remodel") > 0:
			r.remodelOwned = true
			return buyCard(cs.Me, "remodel")
		case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
			return buyCard(cs.Me, "silver")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

func (r *RemodelBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	if d.CardId == "remodel" {
		return r.resolveRemodel(cs, d)
	}
	return safeRefusal(cs, d)
}

func (r *RemodelBM) resolveRemodel(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	me := cs.MyPlayer()

	switch d.Step {
	case 0: // trash
		// Prefer trashing estate, then cheapest card.
		best := ""
		bestCost := 999
		for _, card := range me.Hand {
			if card == "estate" {
				best = "estate"
				break
			}
			cost := cardCost(card)
			if best == "" || cost < bestCost {
				best = card
				bestCost = cost
			}
		}
		return &pb.ResolveDecision{
			DecisionId: d.Id, PlayerIdx: d.PlayerIdx,
			Answer: &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{
				Cards: []string{best},
			}},
		}

	case 1: // gain — most expensive within limit
		prompt := d.GetGainFromSupply()
		maxCost := int(prompt.MaxCost)
		best := ""
		bestCost := -1
		// Prefer Province > Gold > Silver > anything else.
		priorities := []string{"province", "gold", "silver", "duchy"}
		for _, target := range priorities {
			cost := cardCost(target)
			if cost <= maxCost && supplyCount(cs.Snapshot, target) > 0 {
				best = target
				break
			}
		}
		if best == "" {
			// Fallback: most expensive available.
			for _, pile := range cs.Snapshot.Supply {
				if pile.Count <= 0 {
					continue
				}
				cost := cardCost(pile.CardId)
				if cost <= maxCost && cost > bestCost {
					best = pile.CardId
					bestCost = cost
				}
			}
		}
		if best == "" {
			best = "copper"
		}
		return &pb.ResolveDecision{
			DecisionId: d.Id, PlayerIdx: d.PlayerIdx,
			Answer: &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{
				Card: best,
			}},
		}
	}

	return safeRefusal(cs, d)
}
