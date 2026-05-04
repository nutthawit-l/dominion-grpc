package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

// Gardens is a Victory card whose value scales with deck size:
// "Worth 1 Victory Point per 10 cards you have (round down)."
//
// Each Gardens copy in any zone independently contributes
// floor(n/10) VP, where n counts every card the player owns:
// Hand + Deck + Discard + InPlay + SetAside. Cards in trash do
// not count (trash is not part of PlayerState).
//
// SetAside is included defensively. By end-of-game cleanup it's
// empty for the active player; counting it here makes the function
// correct for any caller (debug snapshots, mid-game previews) with
// no separate "active vs other player" code path.
var Gardens = &engine.Card{
	ID:    "gardens",
	Name:  "Gardens",
	Cost:  4,
	Types: []engine.CardType{engine.TypeVictory},
	VictoryPoints: func(p engine.PlayerState) int {
		n := len(p.Hand) + len(p.Deck) + len(p.Discard) + len(p.InPlay) + len(p.SetAside)
		return n / 10
	},
}

func init() {
	DefaultRegistry.Register(Gardens)
}
