package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Mine = &engine.Card{
	ID:    "mine",
	Name:  "Mine",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		hasTreasure := false
		for _, c := range s.Players[p].Hand {
			card, ok := DefaultRegistry.Lookup(c)
			if ok && card.HasType(engine.TypeTreasure) {
				hasTreasure = true
				break
			}
		}
		if !hasTreasure {
			return nil
		}
		return engine.RequestDecision(s, p, "mine", 0,
			engine.TrashFromHandPrompt{Min: 0, Max: 1, TypeFilter: []engine.CardType{engine.TypeTreasure}}, nil)
	},
	OnResolve: func(s *engine.GameState, p int, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		switch d.Step {
		case 0:
			cards := answer.(engine.CardListAnswer).Cards
			if len(cards) == 0 {
				return nil, nil
			}
			trashed := cards[0]
			events := engine.TrashFromHand(s, p, cards)
			card, ok := lookup(trashed)
			if !ok {
				return nil, fmt.Errorf("mine: unknown card %q", trashed)
			}
			trashedCost := card.Cost
			events = append(events, engine.RequestDecision(s, p, "mine", 1,
				engine.GainFromSupplyPrompt{
					MaxCost:    trashedCost + 3,
					TypeFilter: []engine.CardType{engine.TypeTreasure},
					Dest:       engine.GainToHand,
				},
				map[string]any{"trashed_cost": trashedCost})...)
			return events, nil

		case 1:
			choice := answer.(engine.CardChoiceAnswer)
			card, ok := lookup(choice.Card)
			if !ok {
				return nil, fmt.Errorf("mine: unknown card %q", choice.Card)
			}
			maxCost := d.Context["trashed_cost"].(int) + 3
			if card.Cost > maxCost {
				return nil, fmt.Errorf("mine: card %q costs %d, max %d", choice.Card, card.Cost, maxCost)
			}
			if !card.HasType(engine.TypeTreasure) {
				return nil, fmt.Errorf("mine: card %q is not a treasure", choice.Card)
			}
			return engine.GainCard(s, p, choice.Card, engine.GainToHand), nil
		}
		return nil, fmt.Errorf("mine: unexpected step %d", d.Step)
	},
}

func init() {
	DefaultRegistry.Register(Mine)
}
