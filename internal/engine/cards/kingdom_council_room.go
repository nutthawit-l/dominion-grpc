package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var CouncilRoom = &engine.Card{
	ID:    "council_room",
	Name:  "Council Room",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.DrawCards(gs, px, 4)
		events = append(events, engine.AddBuys(gs, px, 1)...)
		events = append(events, engine.EachOtherPlayer(gs, px, func(idx engine.PlayerIdx) []engine.Event {
			return engine.DrawCards(gs, idx, 1)
		})...)
		return events
	},
}

func init() {
	DefaultRegistry.Register(CouncilRoom)
}
