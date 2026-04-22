package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Remodel = &engine.Card{
	ID:    "remodel",
	Name:  "Remodel",
	Cost:  4,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		if len(gs.Players[px].Hand) == 0 {
			return nil
		}
		return engine.RequestDecision(gs, px, "remodel", 0,
			engine.TrashFromHandPrompt{Min: 1, Max: 1}, nil)
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		switch d.Step {
		case 0:
			cards := answer.(engine.CardListAnswer).Cards
			if len(cards) != 1 {
				return nil, fmt.Errorf("remodel: must trash exactly 1 card")
			}
			trashed := cards[0]
			events := engine.TrashFromHand(gs, px, cards)
			card, ok := lookup(trashed)
			if !ok {
				return nil, fmt.Errorf("remodel: unknown trashed card %q", trashed)
			}
			trashedCost := card.Cost
			events = append(events, engine.RequestDecision(gs, px, "remodel", 1,
				engine.GainFromSupplyPrompt{MaxCost: trashedCost + 2, Dest: engine.GainToDiscard},
				map[engine.ContextKey]any{engine.CtxKeyTrashedCost: trashedCost})...)
			return events, nil

		case 1:
			choice := answer.(engine.CardChoiceAnswer)
			card, ok := lookup(choice.Card)
			if !ok {
				return nil, fmt.Errorf("remodel: unknown card %q", choice.Card)
			}
			maxCost := d.Context[engine.CtxKeyTrashedCost].(int) + 2
			if card.Cost > maxCost {
				return nil, fmt.Errorf("remodel: card %q costs %d, max %d", choice.Card, card.Cost, maxCost)
			}
			return engine.GainCard(gs, px, choice.Card, engine.GainToDiscard), nil
		}
		return nil, fmt.Errorf("remodel: unexpected step %d", d.Step)
	},
}

func init() {
	DefaultRegistry.Register(Remodel)
}
