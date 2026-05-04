package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// ChapelBM is Chapel + Big Money. Buys 1 Chapel early, trashes excess
// Coppers to thin the deck, then follows Big Money with endgame Duchy
// buying. Estates are kept for VP.
type ChapelBM struct {
	chapelOwned  bool
	trashedCount map[string]int
}

func NewChapelBM() *ChapelBM {
	return &ChapelBM{trashedCount: map[string]int{}}
}

// SetTrashedCount sets the internal trash counter for testing.
func (c *ChapelBM) SetTrashedCount(card string, n int) {
	c.trashedCount[card] = n
}

func (c *ChapelBM) Name() string { return "chapel_bm" }

func (c *ChapelBM) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}

	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		if me.Actions >= 1 && handContains(me.Hand, "chapel") && c.hasTrashableCards(me) {
			return playCard(cs.Me, "chapel")
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
		endgame := provincesLeft <= 5
		switch {
		case me.Coins >= 8 && provincesLeft > 0:
			return buyCard(cs.Me, "province")
		case endgame && me.Coins >= 5 && supplyCount(cs.Snapshot, "duchy") > 0:
			return buyCard(cs.Me, "duchy")
		case me.Coins >= 6 && supplyCount(cs.Snapshot, "gold") > 0:
			return buyCard(cs.Me, "gold")
		case endgame && me.Coins >= 2 && supplyCount(cs.Snapshot, "estate") > 0:
			return buyCard(cs.Me, "estate")
		case me.Coins >= 2 && !c.chapelOwned && cs.MyTurnsTaken <= 2 && supplyCount(cs.Snapshot, "chapel") > 0:
			c.chapelOwned = true
			return buyCard(cs.Me, "chapel")
		case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
			return buyCard(cs.Me, "silver")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

func (c *ChapelBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	if d.GetTrashFromHand() != nil && d.CardId == "chapel" {
		return c.resolveChapel(cs, d)
	}
	return safeRefusal(cs, d)
}

func (c *ChapelBM) resolveChapel(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	me := cs.MyPlayer()
	prompt := d.GetTrashFromHand()
	max := int(prompt.Max)

	var toTrash []string
	coppersRemaining := 7 - c.trashedCount["copper"]
	const copperKeep = 3

	// Trash estates first (they are dead cards despite their VP).
	for _, card := range me.Hand {
		if len(toTrash) >= max {
			break
		}
		if card == "estate" {
			toTrash = append(toTrash, card)
		}
	}
	// Fill remaining slots with coppers down to threshold.
	for _, card := range me.Hand {
		if len(toTrash) >= max {
			break
		}
		if card == "copper" && coppersRemaining > copperKeep {
			toTrash = append(toTrash, card)
			coppersRemaining--
		}
	}

	for _, card := range toTrash {
		c.trashedCount[card]++
	}

	return &pb.ResolveDecision{
		DecisionId: d.Id, PlayerIdx: d.PlayerIdx,
		Answer: &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{Cards: toTrash}},
	}
}

func (c *ChapelBM) hasTrashableCards(me *pb.PlayerView) bool {
	coppersRemaining := 7 - c.trashedCount["copper"]
	for _, card := range me.Hand {
		if card == "estate" {
			return true
		}
		if card == "copper" && coppersRemaining > 3 {
			return true
		}
	}
	return false
}
