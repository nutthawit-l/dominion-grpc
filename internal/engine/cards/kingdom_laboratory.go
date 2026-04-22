package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Laboratory = &engine.Card{
	ID:    "laboratory",
	Name:  "Laboratory",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.DrawCards(gs, px, 2)
		events = append(events, engine.AddActions(gs, px, 1)...)
		return events
	},
}

func init() {
	DefaultRegistry.Register(Laboratory)
}
