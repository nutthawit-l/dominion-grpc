package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Chapel = &engine.Card{
	ID:    "chapel",
	Name:  "Chapel",
	Cost:  2,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		return engine.RequestDecision(s, p, "chapel", 0,
			engine.TrashFromHandPrompt{Min: 0, Max: 4}, nil)
	},
	OnResolve: func(s *engine.GameState, p int, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		cards := answer.(engine.CardListAnswer).Cards
		return engine.TrashFromHand(s, p, cards), nil
	},
}

func init() {
	DefaultRegistry.Register(Chapel)
}
