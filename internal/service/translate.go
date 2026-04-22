package service

import (
	"fmt"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

// ActionFromProto converts a proto Action union into the engine's
// internal Action type.
func ActionFromProto(a *pb.Action) (engine.Action, error) {
	if a == nil {
		return nil, fmt.Errorf("service: nil action")
	}
	switch k := a.Kind.(type) {
	case *pb.Action_PlayCard:
		return engine.PlayCard{
			PlayerIdx: engine.PlayerIdx(k.PlayCard.PlayerIdx),
			Card:      engine.CardID(k.PlayCard.CardId),
		}, nil
	case *pb.Action_BuyCard:
		return engine.BuyCard{
			PlayerIdx: engine.PlayerIdx(k.BuyCard.PlayerIdx),
			Card:      engine.CardID(k.BuyCard.CardId),
		}, nil
	case *pb.Action_EndPhase:
		return engine.EndPhase{PlayerIdx: engine.PlayerIdx(k.EndPhase.PlayerIdx)}, nil
	case *pb.Action_Resolve:
		answer, err := AnswerFromProto(k.Resolve)
		if err != nil {
			return nil, err
		}
		return engine.ResolveDecision{
			PlayerIdx:  engine.PlayerIdx(k.Resolve.PlayerIdx),
			DecisionID: k.Resolve.DecisionId,
			Answer:     answer,
		}, nil
	default:
		return nil, fmt.Errorf("service: unknown action kind %T", k)
	}
}

// SnapshotFromState produces a proto snapshot scrubbed to the given
// viewer. The viewer sees their own hand contents; opponents' hand
// contents are omitted (but HandSize is still reported).
func SnapshotFromState(gs *engine.GameState, viewer engine.PlayerIdx) *pb.GameStateSnapshot {
	snap := &pb.GameStateSnapshot{
		GameId:        gs.GameID,
		Seed:          gs.Seed,
		Turn:          int32(gs.Turn),
		CurrentPlayer: int32(gs.CurrentPlayer),
		Phase:         phaseToProto(gs.Phase),
		TrashSize:     int32(len(gs.Trash)),
		Ended:         gs.Ended,
	}
	for i, p := range gs.Players {
		pv := &pb.PlayerView{
			PlayerIdx:   int32(i),
			Name:        p.Name,
			HandSize:    int32(len(p.Hand)),
			DeckSize:    int32(len(p.Deck)),
			DiscardSize: int32(len(p.Discard)),
			InPlaySize:  int32(len(p.InPlay)),
			Actions:     int32(p.Actions),
			Buys:        int32(p.Buys),
			Coins:       int32(p.Coins),
		}
		if engine.PlayerIdx(i) == viewer {
			for _, c := range p.Hand {
				pv.Hand = append(pv.Hand, string(c))
			}
		}
		for _, c := range p.InPlay {
			pv.InPlay = append(pv.InPlay, string(c))
		}
		snap.Players = append(snap.Players, pv)
	}
	for id, n := range gs.Supply.Piles {
		snap.Supply = append(snap.Supply, &pb.SupplyPile{
			CardId: string(id),
			Count:  int32(n),
		})
	}
	for _, w := range gs.Winners {
		snap.Winners = append(snap.Winners, int32(w))
	}
	snap.PendingDecision = DecisionToProto(gs.PendingDecision)
	return snap
}

// DecisionToProto translates an engine Decision to its proto representation.
func DecisionToProto(d *engine.Decision) *pb.Decision {
	if d == nil {
		return nil
	}
	pd := &pb.Decision{
		Id:        d.ID,
		PlayerIdx: int32(d.PlayerIdx),
		CardId:    string(d.CardID),
		Step:      int32(d.Step),
	}
	switch p := d.Prompt.(type) {
	case engine.DiscardFromHandPrompt:
		pd.Prompt = &pb.Decision_DiscardFromHand{DiscardFromHand: &pb.DiscardFromHandPrompt{
			Min: int32(p.Min), Max: int32(p.Max),
		}}
	case engine.TrashFromHandPrompt:
		tf := make([]pb.CardType, len(p.TypeFilter))
		for i, ct := range p.TypeFilter {
			tf[i] = cardTypeToProto(ct)
		}
		cf := make([]string, len(p.CardFilter))
		for i, c := range p.CardFilter {
			cf[i] = string(c)
		}
		pd.Prompt = &pb.Decision_TrashFromHand{TrashFromHand: &pb.TrashFromHandPrompt{
			Min: int32(p.Min), Max: int32(p.Max), TypeFilter: tf, CardFilter: cf,
		}}
	case engine.GainFromSupplyPrompt:
		tf := make([]pb.CardType, len(p.TypeFilter))
		for i, ct := range p.TypeFilter {
			tf[i] = cardTypeToProto(ct)
		}
		pd.Prompt = &pb.Decision_GainFromSupply{GainFromSupply: &pb.GainFromSupplyPrompt{
			MaxCost: int32(p.MaxCost), TypeFilter: tf, Dest: gainDestToProto(p.Dest),
		}}
	case engine.ChooseFromDiscardPrompt:
		cards := make([]string, len(p.Cards))
		for i, c := range p.Cards {
			cards[i] = string(c)
		}
		pd.Prompt = &pb.Decision_ChooseFromDiscard{ChooseFromDiscard: &pb.ChooseFromDiscardPrompt{
			Cards: cards, Optional: p.Optional,
		}}
	case engine.PutOnDeckPrompt:
		pd.Prompt = &pb.Decision_PutOnDeck{PutOnDeck: &pb.PutOnDeckPrompt{}}
	case engine.MayPlayActionPrompt:
		pd.Prompt = &pb.Decision_MayPlayAction{MayPlayAction: &pb.MayPlayActionPrompt{
			CardId: string(p.Card),
		}}
	}
	return pd
}

// AnswerFromProto translates a proto ResolveDecision answer into an engine Answer.
func AnswerFromProto(r *pb.ResolveDecision) (engine.Answer, error) {
	switch a := r.Answer.(type) {
	case *pb.ResolveDecision_CardList:
		cards := make([]engine.CardID, len(a.CardList.Cards))
		for i, c := range a.CardList.Cards {
			cards[i] = engine.CardID(c)
		}
		return engine.CardListAnswer{Cards: cards}, nil
	case *pb.ResolveDecision_CardChoice:
		return engine.CardChoiceAnswer{
			Card: engine.CardID(a.CardChoice.Card),
			None: a.CardChoice.None,
		}, nil
	case *pb.ResolveDecision_YesNo:
		return engine.YesNoAnswer{Yes: a.YesNo.Yes}, nil
	default:
		return nil, fmt.Errorf("service: unknown answer type %T", a)
	}
}

func cardTypeToProto(ct engine.CardType) pb.CardType {
	switch ct {
	case engine.TypeTreasure:
		return pb.CardType_CARD_TYPE_TREASURE
	case engine.TypeVictory:
		return pb.CardType_CARD_TYPE_VICTORY
	case engine.TypeCurse:
		return pb.CardType_CARD_TYPE_CURSE
	case engine.TypeAction:
		return pb.CardType_CARD_TYPE_ACTION
	case engine.TypeAttack:
		return pb.CardType_CARD_TYPE_ATTACK
	case engine.TypeReaction:
		return pb.CardType_CARD_TYPE_REACTION
	}
	return pb.CardType_CARD_TYPE_UNSPECIFIED
}

func gainDestToProto(d engine.GainDest) pb.GainDest {
	switch d {
	case engine.GainToDiscard:
		return pb.GainDest_GAIN_DEST_DISCARD
	case engine.GainToHand:
		return pb.GainDest_GAIN_DEST_HAND
	case engine.GainToDeck:
		return pb.GainDest_GAIN_DEST_DECK
	}
	return pb.GainDest_GAIN_DEST_UNSPECIFIED
}

func phaseToProto(p engine.Phase) pb.Phase {
	switch p {
	case engine.PhaseAction:
		return pb.Phase_PHASE_ACTION
	case engine.PhaseBuy:
		return pb.Phase_PHASE_BUY
	case engine.PhaseCleanup:
		return pb.Phase_PHASE_CLEANUP
	}
	return pb.Phase_PHASE_UNSPECIFIED
}
