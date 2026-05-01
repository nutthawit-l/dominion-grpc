package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

// Merchant is a $3 Action: "+1 Card, +1 Action. The first time you
// play a Silver this turn, +$1."
//
// The "first Silver this turn" trigger is split across two cards:
//   - Merchant.OnPlay increments PlayerState.MerchantBonusCharges.
//   - Silver.OnPlay reads charges + FirstSilverPlayedThisTurn,
//     pays the bonus once per turn.
//
// cleanupAndEndTurn resets both fields. See basics_treasure.go.
var Merchant = &engine.Card{
	ID:    "merchant",
	Name:  "Merchant",
	Cost:  3,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.DrawCards(gs, px, 1)
		events = append(events, engine.AddActions(gs, px, 1)...)
		gs.Players[px].MerchantBonusCharges++
		return events
	},
}

func init() {
	DefaultRegistry.Register(Merchant)
}
