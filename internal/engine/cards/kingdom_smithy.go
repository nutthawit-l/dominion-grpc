package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Smithy = &engine.Card{
	ID:    "smithy",
	Name:  "Smithy",
	Cost:  4,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		return engine.DrawCards(gs, px, 3)
	},
}

func init() {
	DefaultRegistry.Register(Smithy)
}
