package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Copper = &engine.Card{
	ID:    "copper",
	Name:  "Copper",
	Cost:  0,
	Types: []engine.CardType{engine.TypeTreasure},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		return engine.AddCoins(gs, px, 1)
	},
}

var Silver = &engine.Card{
	ID:    "silver",
	Name:  "Silver",
	Cost:  3,
	Types: []engine.CardType{engine.TypeTreasure},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		return engine.AddCoins(gs, px, 2)
	},
}

var Gold = &engine.Card{
	ID:    "gold",
	Name:  "Gold",
	Cost:  6,
	Types: []engine.CardType{engine.TypeTreasure},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		return engine.AddCoins(gs, px, 3)
	},
}

func init() {
	DefaultRegistry.Register(Copper)
	DefaultRegistry.Register(Silver)
	DefaultRegistry.Register(Gold)
}
