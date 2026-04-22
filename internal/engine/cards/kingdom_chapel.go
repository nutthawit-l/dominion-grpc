package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Chapel = &engine.Card{
	ID:    "chapel",
	Name:  "Chapel",
	Cost:  2,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		return engine.RequestDecision(gs, px, "chapel", 0,
			engine.TrashFromHandPrompt{Min: 0, Max: 4}, nil)
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		cards := answer.(engine.CardListAnswer).Cards
		return engine.TrashFromHand(gs, px, cards), nil
	},
}

func init() {
	DefaultRegistry.Register(Chapel)
}
