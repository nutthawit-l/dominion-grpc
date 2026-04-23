package cards

import (
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Witch = &engine.Card{
	ID:    "witch",
	Name:  "Witch",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction, engine.TypeAttack},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.DrawCards(gs, px, 2)
		victims, attackEvents := engine.ResolveAttackVictims(gs, px, "witch", registryLookup)
		events = append(events, attackEvents...)
		for _, v := range victims {
			events = append(events, engine.GainCard(gs, v, "curse", engine.GainToDiscard)...)
		}
		return events
	},
}

func init() {
	DefaultRegistry.Register(Witch)
}
