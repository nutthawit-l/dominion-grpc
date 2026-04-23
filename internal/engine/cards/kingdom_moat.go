package cards

import (
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Moat = &engine.Card{
	ID:    "moat",
	Name:  "Moat",
	Cost:  2,
	Types: []engine.CardType{engine.TypeAction, engine.TypeReaction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		return engine.DrawCards(gs, px, 2)
	},
	OnReaction: func(gs *engine.GameState, victim engine.PlayerIdx, trigger engine.Trigger) (bool, []engine.Event) {
		if trigger.Kind != engine.TriggerAttackPlayed {
			return false, nil
		}
		return true, []engine.Event{{
			Kind:      engine.EventReactionTriggered,
			PlayerIdx: victim,
			CardID:    "moat",
		}}
	},
}

func init() {
	DefaultRegistry.Register(Moat)
}
