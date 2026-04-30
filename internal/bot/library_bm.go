package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// LibraryBM is Big Money plus Library. Buys up to 2 Libraries at 5+,
// then reverts to Big Money. Plays Library whenever in hand. Defaults
// to refusing the SetAside prompt — this is a BigMoney-shape deck and
// a drawn Action is rare enough to be worth keeping.
type LibraryBM struct {
	libraries int
}

// NewLibraryBM constructs a LibraryBM strategy.
func NewLibraryBM() *LibraryBM { return &LibraryBM{} }

// Name implements Strategy.
func (l *LibraryBM) Name() string { return "library_bm" }

// PickAction implements Strategy.
func (l *LibraryBM) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}
	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		if me.Actions >= 1 && handContains(me.Hand, "library") {
			return playCard(cs.Me, "library")
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
		switch {
		case me.Coins >= 8 && supplyCount(cs.Snapshot, "province") > 0:
			return buyCard(cs.Me, "province")
		case me.Coins >= 6 && supplyCount(cs.Snapshot, "gold") > 0:
			return buyCard(cs.Me, "gold")
		case me.Coins >= 5 && l.libraries < 2 && supplyCount(cs.Snapshot, "library") > 0:
			l.libraries++
			return buyCard(cs.Me, "library")
		case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
			return buyCard(cs.Me, "silver")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

// Resolve implements Strategy.
func (l *LibraryBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	return safeRefusal(cs, d)
}
