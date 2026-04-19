package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Village = &engine.Card{
	ID:    "village",
	Name:  "Village",
	Cost:  3,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		events := engine.DrawCards(s, p, 1)
		events = append(events, engine.AddActions(s, p, 2)...)
		return events
	},
}

func init() {
	DefaultRegistry.Register(Village)
}
