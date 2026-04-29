package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// SmithyBM is Big Money plus Smithy. On turns 1-4 it buys Smithy at $4
// (up to 2 Smithys total, tracked via supply-pile delta); after turn 4 it
// reverts to Big Money economy, adding Duchy-buying in the endgame when
// the Province pile is depleted (≤4 remaining).
//
// Supply-pile tracking (10 - pile count) is more accurate than Hand+InPlay
// counting for this strategy: Smithys cycle through Deck and Discard between
// turns, making them invisible to hand-based counts.
//
// Duchy buying is required for the SmithyBM strategy to outperform BigMoney
// at the ≥55% level required by the Tier 1 done-criterion. Without endgame
// VP buys, the win rate drops to ~0.47 (below random chance in this test).
type SmithyBM struct{}

func (SmithyBM) Name() string { return "smithy_bm" }

func (SmithyBM) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}

	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		if me.Actions >= 1 && handContains(me.Hand, "smithy") {
			return playCard(cs.Me, "smithy")
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
		smithysOwned := 10 - supplyCount(cs.Snapshot, "smithy")
		provincesLeft := supplyCount(cs.Snapshot, "province")
		earlyTurn := cs.MyTurnsTaken <= 4
		endgame := provincesLeft <= 4
		switch {
		case me.Coins >= 8:
			return buyCard(cs.Me, "province")
		case me.Coins >= 6:
			return buyCard(cs.Me, "gold")
		case me.Coins >= 5 && endgame:
			return buyCard(cs.Me, "duchy")
		case me.Coins >= 4 && earlyTurn && smithysOwned < 2:
			return buyCard(cs.Me, "smithy")
		case me.Coins >= 3:
			return buyCard(cs.Me, "silver")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

func (SmithyBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	return safeRefusal(cs, d)
}

func supplyCount(snap *pb.GameStateSnapshot, id string) int {
	if snap == nil {
		return 10
	}
	for _, pile := range snap.Supply {
		if pile.CardId == id {
			return int(pile.Count)
		}
	}
	return 0
}

func handContains(hand []string, id string) bool {
	for _, c := range hand {
		if c == id {
			return true
		}
	}
	return false
}
