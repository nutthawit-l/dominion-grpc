package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Village = &engine.Card{
	ID:    "village",
	Name:  "Village",
	Cost:  3,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.DrawCards(gs, px, 1)
		events = append(events, engine.AddActions(gs, px, 2)...)
		return events
	},
}

func init() {
	DefaultRegistry.Register(Village)
}
