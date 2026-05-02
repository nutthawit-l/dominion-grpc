package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// MerchantBM is Big Money plus Merchant. Buys up to 2 Merchants at $3
// (counter incremented on each buy), then reverts to Big Money. Plays
// Merchant whenever in hand and Actions remain. Defaults to safe
// refusal on every prompt — Merchant itself prompts no decisions.
type MerchantBM struct {
	merchants int
}

// NewMerchantBM constructs a MerchantBM strategy.
func NewMerchantBM() *MerchantBM { return &MerchantBM{} }

// Name implements Strategy.
func (m *MerchantBM) Name() string { return "merchant_bm" }

// PickAction implements Strategy.
func (m *MerchantBM) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}
	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		if me.Actions >= 1 && handContains(me.Hand, "merchant") {
			return playCard(cs.Me, "merchant")
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
		case me.Coins >= 3 && m.merchants < 2 && supplyCount(cs.Snapshot, "merchant") > 0:
			m.merchants++
			return buyCard(cs.Me, "merchant")
		case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
			return buyCard(cs.Me, "silver")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

// Resolve implements Strategy.
func (m *MerchantBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	return safeRefusal(cs, d)
}
