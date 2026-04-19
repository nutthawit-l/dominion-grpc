package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Festival = &engine.Card{
	ID:    "festival",
	Name:  "Festival",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		events := engine.AddActions(s, p, 2)
		events = append(events, engine.AddBuys(s, p, 1)...)
		events = append(events, engine.AddCoins(s, p, 2)...)
		return events
	},
}

func init() {
	DefaultRegistry.Register(Festival)
}
