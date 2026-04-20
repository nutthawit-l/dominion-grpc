package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// BigMoney is the canonical starter strategy: never play action cards,
// play every treasure, buy Provinces at 8+, Gold at 6+, Silver at 3+.
type BigMoney struct{}

// Name implements Strategy.
func (BigMoney) Name() string { return "bigmoney" }

// PickAction implements Strategy.
func (BigMoney) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}
	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		return endPhase(cs.Me)
	case pb.Phase_PHASE_BUY:
		for _, c := range me.Hand {
			if isTreasure(c) {
				return playCard(cs.Me, c)
			}
		}
		if me.Buys <= 0 {
			return endPhase(cs.Me)
		}
		switch {
		case me.Coins >= 8:
			return buyCard(cs.Me, "province")
		case me.Coins >= 6:
			return buyCard(cs.Me, "gold")
		case me.Coins >= 3:
			return buyCard(cs.Me, "silver")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

// Resolve implements Strategy.
func (BigMoney) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	return safeRefusal(cs, d)
}

func isTreasure(id string) bool {
	switch id {
	case "copper", "silver", "gold":
		return true
	}
	return false
}
