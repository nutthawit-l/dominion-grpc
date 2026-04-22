package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Workshop = &engine.Card{
	ID:    "workshop",
	Name:  "Workshop",
	Cost:  3,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		return engine.RequestDecision(gs, px, "workshop", 0,
			engine.GainFromSupplyPrompt{MaxCost: 4, Dest: engine.GainToDiscard}, nil)
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		choice := answer.(engine.CardChoiceAnswer)
		card, ok := lookup(choice.Card)
		if !ok {
			return nil, fmt.Errorf("workshop: unknown card %q", choice.Card)
		}
		prompt := d.Prompt.(engine.GainFromSupplyPrompt)
		if card.Cost > prompt.MaxCost {
			return nil, fmt.Errorf("workshop: card %q costs %d, max %d", choice.Card, card.Cost, prompt.MaxCost)
		}
		return engine.GainCard(gs, px, choice.Card, prompt.Dest), nil
	},
}

func init() {
	DefaultRegistry.Register(Workshop)
}
