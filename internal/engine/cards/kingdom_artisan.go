package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Artisan = &engine.Card{
	ID:    "artisan",
	Name:  "Artisan",
	Cost:  6,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		return engine.RequestDecision(s, p, "artisan", 0,
			engine.GainFromSupplyPrompt{MaxCost: 5, Dest: engine.GainToHand}, nil)
	},
	OnResolve: func(s *engine.GameState, p int, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		switch d.Step {
		case 0:
			choice := answer.(engine.CardChoiceAnswer)
			card, ok := lookup(choice.Card)
			if !ok {
				return nil, fmt.Errorf("artisan: unknown card %q", choice.Card)
			}
			if card.Cost > 5 {
				return nil, fmt.Errorf("artisan: card %q costs %d, max 5", choice.Card, card.Cost)
			}
			events := engine.GainCard(s, p, choice.Card, engine.GainToHand)
			events = append(events, engine.RequestDecision(s, p, "artisan", 1,
				engine.PutOnDeckPrompt{}, nil)...)
			return events, nil

		case 1:
			choice := answer.(engine.CardChoiceAnswer)
			return engine.PutOnDeck(s, p, []engine.CardID{choice.Card}), nil
		}
		return nil, fmt.Errorf("artisan: unexpected step %d", d.Step)
	},
}

func init() {
	DefaultRegistry.Register(Artisan)
}
