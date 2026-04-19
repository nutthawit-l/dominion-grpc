package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Smithy = &engine.Card{
	ID:    "smithy",
	Name:  "Smithy",
	Cost:  4,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		return engine.DrawCards(s, p, 3)
	},
}

func init() {
	DefaultRegistry.Register(Smithy)
}
