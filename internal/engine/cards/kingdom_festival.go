package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Festival = &engine.Card{
	ID:    "festival",
	Name:  "Festival",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.AddActions(gs, px, 2)
		events = append(events, engine.AddBuys(gs, px, 1)...)
		events = append(events, engine.AddCoins(gs, px, 2)...)
		return events
	},
}

func init() {
	DefaultRegistry.Register(Festival)
}
