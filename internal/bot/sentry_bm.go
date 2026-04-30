package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// SentryBM is Big Money plus Sentry — buys one Sentry at 5+ for deck
// thinning, then reverts to Big Money. When Sentry reveals top-2:
// trashes any Curse/Estate; discards nothing extra; identity reorder.
type SentryBM struct {
	sentries int
}

// NewSentryBM constructs a SentryBM strategy.
func NewSentryBM() *SentryBM { return &SentryBM{} }

// Name implements Strategy.
func (s *SentryBM) Name() string { return "sentry_bm" }

// PickAction implements Strategy.
func (s *SentryBM) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}
	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		if me.Actions >= 1 && handContains(me.Hand, "sentry") {
			return playCard(cs.Me, "sentry")
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
		case me.Coins >= 5 && s.sentries < 1 && supplyCount(cs.Snapshot, "sentry") > 0:
			s.sentries++
			return buyCard(cs.Me, "sentry")
		case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
			return buyCard(cs.Me, "silver")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

// Resolve implements Strategy.
func (s *SentryBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	if d.CardId != "sentry" {
		return safeRefusal(cs, d)
	}
	switch d.Prompt.(type) {
	case *pb.Decision_TrashFromRevealed:
		return s.resolveTrash(d)
	case *pb.Decision_DiscardFromRevealed:
		return safeRefusal(cs, d)
	case *pb.Decision_ReorderCards:
		return safeRefusal(cs, d)
	}
	return safeRefusal(cs, d)
}

func (s *SentryBM) resolveTrash(d *pb.Decision) *pb.ResolveDecision {
	cards := d.GetTrashFromRevealed().Cards
	var toTrash []string
	for _, c := range cards {
		if c == "curse" || c == "estate" {
			toTrash = append(toTrash, c)
		}
	}
	return &pb.ResolveDecision{
		DecisionId: d.Id, PlayerIdx: d.PlayerIdx,
		Answer: &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{Cards: toTrash}},
	}
}


