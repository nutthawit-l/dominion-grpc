package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Market = &engine.Card{
	ID:    "market",
	Name:  "Market",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.DrawCards(gs, px, 1)
		events = append(events, engine.AddActions(gs, px, 1)...)
		events = append(events, engine.AddBuys(gs, px, 1)...)
		events = append(events, engine.AddCoins(gs, px, 1)...)
		return events
	},
}

func init() {
	DefaultRegistry.Register(Market)
}
