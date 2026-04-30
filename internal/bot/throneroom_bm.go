package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// ThroneRoomBM is Big Money plus the Throne-Room+Witch combo.
type ThroneRoomBM struct {
	witchOwned bool
}

// NewThroneRoomBM constructs a ThroneRoomBM strategy.
func NewThroneRoomBM() *ThroneRoomBM { return &ThroneRoomBM{} }

// Name implements Strategy.
func (t *ThroneRoomBM) Name() string { return "throneroom_bm" }

// PickAction implements Strategy.
func (t *ThroneRoomBM) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}

	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		if me.Actions <= 0 {
			return endPhase(cs.Me)
		}
		if handContains(me.Hand, "throne_room") && handHasNonTRAction(me.Hand) {
			return playCard(cs.Me, "throne_room")
		}
		if handContains(me.Hand, "witch") {
			return playCard(cs.Me, "witch")
		}
		if c := firstActionInHand(me.Hand); c != "" {
			return playCard(cs.Me, c)
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
		earlyTurn := cs.MyTurnsTaken >= 3 && cs.MyTurnsTaken <= 6
		switch {
		case me.Coins >= 8 && supplyCount(cs.Snapshot, "province") > 0:
			return buyCard(cs.Me, "province")
		case me.Coins >= 6 && supplyCount(cs.Snapshot, "gold") > 0:
			return buyCard(cs.Me, "gold")
		case me.Coins >= 5 && earlyTurn && !t.witchOwned && supplyCount(cs.Snapshot, "witch") > 0:
			t.witchOwned = true
			return buyCard(cs.Me, "witch")
		case me.Coins == 4 && earlyTurn && t.witchOwned && supplyCount(cs.Snapshot, "throne_room") > 0:
			return buyCard(cs.Me, "throne_room")
		case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
			return buyCard(cs.Me, "silver")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

// Resolve implements Strategy.
func (t *ThroneRoomBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	if _, ok := d.Prompt.(*pb.Decision_ChooseActionFromHand); ok {
		return t.resolveChooseAction(cs, d)
	}
	return safeRefusal(cs, d)
}

func (t *ThroneRoomBM) resolveChooseAction(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	me := cs.MyPlayer()
	pick := ""
	if me != nil {
		for _, c := range me.Hand {
			if c == "witch" {
				pick = c
				break
			}
		}
		if pick == "" {
			for _, c := range me.Hand {
				if isAction(c) && c != "throne_room" {
					pick = c
					break
				}
			}
		}
		if pick == "" {
			for _, c := range me.Hand {
				if isAction(c) {
					pick = c
					break
				}
			}
		}
	}
	return &pb.ResolveDecision{
		DecisionId: d.Id, PlayerIdx: d.PlayerIdx,
		Answer: &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{Card: pick}},
	}
}

func handHasNonTRAction(hand []string) bool {
	for _, c := range hand {
		if c != "throne_room" && isAction(c) {
			return true
		}
	}
	return false
}

func firstActionInHand(hand []string) string {
	for _, c := range hand {
		if isAction(c) {
			return c
		}
	}
	return ""
}
