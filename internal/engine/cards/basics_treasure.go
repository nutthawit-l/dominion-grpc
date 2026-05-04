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

// Silver is a $3 Treasure: "+$2." It also delivers Merchant's bonus —
// the first time Silver is played in a turn, any Merchants played
// earlier this turn (counted in MerchantBonusCharges) each contribute
// +$1. The bookkeeping fields live on PlayerState and are owned by
// Tier 5; cleanupAndEndTurn resets them.
var Silver = &engine.Card{
	ID:    "silver",
	Name:  "Silver",
	Cost:  3,
	Types: []engine.CardType{engine.TypeTreasure},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.AddCoins(gs, px, 2)
		ps := &gs.Players[px]
		if !ps.FirstSilverPlayedThisTurn {
			ps.FirstSilverPlayedThisTurn = true
			if ps.MerchantBonusCharges > 0 {
				events = append(events, engine.AddCoins(gs, px, ps.MerchantBonusCharges)...)
			}
		}
		return events
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
