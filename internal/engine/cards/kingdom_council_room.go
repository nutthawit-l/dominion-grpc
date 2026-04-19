package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var CouncilRoom = &engine.Card{
	ID:    "council_room",
	Name:  "Council Room",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		events := engine.DrawCards(s, p, 4)
		events = append(events, engine.AddBuys(s, p, 1)...)
		events = append(events, engine.EachOtherPlayer(s, p, func(idx int) []engine.Event {
			return engine.DrawCards(s, idx, 1)
		})...)
		return events
	},
}

func init() {
	DefaultRegistry.Register(CouncilRoom)
}
