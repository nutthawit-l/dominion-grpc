package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// GardensBM is a deliberate Gardens-rush strategy: bulks the deck with
// Coppers when nothing better is available, buys Gardens at $4 (up to
// 4 copies), and otherwise follows BigMoney's economy ladder. Action
// phase is always a no-op since Gardens is Victory-only.
//
// Gardens-rush in 2-player Base is a known weak strategy (Provinces
// drain too fast). The smoke sweep proves correctness, not strategy
// quality — wins >= 1 over 50 games is the bar.
type GardensBM struct {
	gardens int
}

// NewGardensBM constructs a GardensBM strategy.
func NewGardensBM() *GardensBM { return &GardensBM{} }

// Name implements Strategy.
func (g *GardensBM) Name() string { return "gardens_bm" }

// PickAction implements Strategy.
func (g *GardensBM) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}
	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		return endPhase(cs.Me) // Gardens is Victory-only.

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
		case me.Coins >= 8 && supplyCount(cs.Snapshot, "province") > 0:
			return buyCard(cs.Me, "province")
		case me.Coins >= 6 && supplyCount(cs.Snapshot, "gold") > 0:
			return buyCard(cs.Me, "gold")
		case me.Coins >= 4 && g.gardens < 4 && supplyCount(cs.Snapshot, "gardens") > 0:
			g.gardens++
			return buyCard(cs.Me, "gardens")
		case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
			return buyCard(cs.Me, "silver")
		case me.Coins >= 0 && supplyCount(cs.Snapshot, "copper") > 0:
			return buyCard(cs.Me, "copper")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

// Resolve implements Strategy.
func (g *GardensBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	return safeRefusal(cs, d)
}
