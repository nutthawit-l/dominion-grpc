package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Market = &engine.Card{
	ID:    "market",
	Name:  "Market",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		events := engine.DrawCards(s, p, 1)
		events = append(events, engine.AddActions(s, p, 1)...)
		events = append(events, engine.AddBuys(s, p, 1)...)
		events = append(events, engine.AddCoins(s, p, 1)...)
		return events
	},
}

func init() {
	DefaultRegistry.Register(Market)
}
