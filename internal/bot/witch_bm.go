package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// WitchBM is Big Money plus Witch. Buys exactly one Witch on turns 3-4
// when $5+ is available, then reverts to Big Money economy. The attack
// is the strategy's value; a second Witch displaces treasure buys without
// increasing Curse output meaningfully.
type WitchBM struct {
	witchOwned bool
}

// NewWitchBM constructs a WitchBM strategy.
func NewWitchBM() *WitchBM { return &WitchBM{} }

// Name implements Strategy.
func (w *WitchBM) Name() string { return "witch_bm" }

// PickAction implements Strategy.
func (w *WitchBM) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}

	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		if me.Actions >= 1 && handContains(me.Hand, "witch") {
			return playCard(cs.Me, "witch")
		}
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
		earlyTurn := cs.MyTurnsTaken >= 3 && cs.MyTurnsTaken <= 4
		switch {
		case me.Coins >= 8 && supplyCount(cs.Snapshot, "province") > 0:
			return buyCard(cs.Me, "province")
		case me.Coins >= 6 && supplyCount(cs.Snapshot, "gold") > 0:
			return buyCard(cs.Me, "gold")
		case me.Coins >= 5 && earlyTurn && !w.witchOwned && supplyCount(cs.Snapshot, "witch") > 0:
			w.witchOwned = true
			return buyCard(cs.Me, "witch")
		case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
			return buyCard(cs.Me, "silver")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

// Resolve implements Strategy.
func (w *WitchBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	return safeRefusal(cs, d)
}
