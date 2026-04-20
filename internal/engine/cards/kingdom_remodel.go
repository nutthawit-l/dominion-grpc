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
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		if len(s.Players[p].Hand) == 0 {
			return nil
		}
		return engine.RequestDecision(s, p, "remodel", 0,
			engine.TrashFromHandPrompt{Min: 1, Max: 1}, nil)
	},
	OnResolve: func(s *engine.GameState, p int, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		switch d.Step {
		case 0:
			cards := answer.(engine.CardListAnswer).Cards
			if len(cards) != 1 {
				return nil, fmt.Errorf("remodel: must trash exactly 1 card")
			}
			trashed := cards[0]
			events := engine.TrashFromHand(s, p, cards)
			card, ok := lookup(trashed)
			if !ok {
				return nil, fmt.Errorf("remodel: unknown trashed card %q", trashed)
			}
			trashedCost := card.Cost
			events = append(events, engine.RequestDecision(s, p, "remodel", 1,
				engine.GainFromSupplyPrompt{MaxCost: trashedCost + 2, Dest: engine.GainToDiscard},
				map[string]any{"trashed_cost": trashedCost})...)
			return events, nil

		case 1:
			choice := answer.(engine.CardChoiceAnswer)
			card, ok := lookup(choice.Card)
			if !ok {
				return nil, fmt.Errorf("remodel: unknown card %q", choice.Card)
			}
			maxCost := d.Context["trashed_cost"].(int) + 2
			if card.Cost > maxCost {
				return nil, fmt.Errorf("remodel: card %q costs %d, max %d", choice.Card, card.Cost, maxCost)
			}
			return engine.GainCard(s, p, choice.Card, engine.GainToDiscard), nil
		}
		return nil, fmt.Errorf("remodel: unexpected step %d", d.Step)
	},
}

func init() {
	DefaultRegistry.Register(Remodel)
}
