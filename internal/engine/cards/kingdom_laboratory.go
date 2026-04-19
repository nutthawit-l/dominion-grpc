package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Laboratory = &engine.Card{
	ID:    "laboratory",
	Name:  "Laboratory",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		events := engine.DrawCards(s, p, 2)
		events = append(events, engine.AddActions(s, p, 1)...)
		return events
	},
}

func init() {
	DefaultRegistry.Register(Laboratory)
}
