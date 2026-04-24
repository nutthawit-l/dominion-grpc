package bot

import (
	"sort"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// MilitiaBM is Big Money plus Militia. Buys one Militia on turns 3-4
// at $4+, then reverts to Big Money. When attacked by Militia (as the
// victim), keeps the highest-value 3 cards; discards the rest.
type MilitiaBM struct {
	militiaOwned bool
}

func NewMilitiaBM() *MilitiaBM { return &MilitiaBM{} }

func (m *MilitiaBM) Name() string { return "militia_bm" }

func (m *MilitiaBM) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}

	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		if me.Actions >= 1 && handContains(me.Hand, "militia") {
			return playCard(cs.Me, "militia")
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
		case me.Coins >= 4 && earlyTurn && !m.militiaOwned && supplyCount(cs.Snapshot, "militia") > 0:
			m.militiaOwned = true
			return buyCard(cs.Me, "militia")
		case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
			return buyCard(cs.Me, "silver")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

func (m *MilitiaBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	if d.GetDiscardFromHand() != nil && d.CardId == "militia" {
		return m.resolveMilitiaDiscard(cs, d)
	}
	return safeRefusal(cs, d)
}

func (m *MilitiaBM) resolveMilitiaDiscard(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	me := cs.MyPlayer()
	prompt := d.GetDiscardFromHand()
	n := int(prompt.Min)

	hand := append([]string(nil), me.Hand...)
	sort.SliceStable(hand, func(i, j int) bool {
		return militiaKeepValue(hand[i]) > militiaKeepValue(hand[j])
	})

	toDiscard := hand[len(hand)-n:]
	return &pb.ResolveDecision{
		DecisionId: d.Id, PlayerIdx: d.PlayerIdx,
		Answer: &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{Cards: toDiscard}},
	}
}

// militiaKeepValue ranks cards for Militia's discard-to-3 attack.
// Higher = keep. Gold > Silver > Copper > (dead VP/Curse).
func militiaKeepValue(id string) int {
	switch id {
	case "gold":
		return 100
	case "silver":
		return 90
	case "copper":
		return 80
	case "curse":
		return -10
	case "estate", "duchy", "province":
		return 0
	}
	return 50
}
