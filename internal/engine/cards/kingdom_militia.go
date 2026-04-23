package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Militia = &engine.Card{
	ID:    "militia",
	Name:  "Militia",
	Cost:  4,
	Types: []engine.CardType{engine.TypeAction, engine.TypeAttack},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.AddCoins(gs, px, 2)
		victims, attackEvents := engine.ResolveAttackVictims(gs, px, "militia", registryLookup)
		events = append(events, attackEvents...)
		events = append(events, militiaQueue(gs, px, victims)...)
		return events
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		cards := answer.(engine.CardListAnswer).Cards
		expected := len(gs.Players[px].Hand) - 3
		if expected < 0 {
			expected = 0
		}
		if len(cards) != expected {
			return nil, fmt.Errorf("militia: must discard exactly %d cards, got %d", expected, len(cards))
		}
		events := engine.DiscardFromHand(gs, px, cards)
		attacker := d.Context[engine.CtxKeyAttacker].(engine.PlayerIdx)
		remaining := d.Context[engine.CtxKeyRemainingVictims].([]engine.PlayerIdx)
		events = append(events, militiaQueue(gs, attacker, remaining)...)
		return events, nil
	},
}

// militiaQueue advances through the victim list, skipping victims whose
// hand is already at or below 3 cards and parking a decision on the
// first victim who needs to discard. Returns only the RequestDecision
// event (if any); caller accumulates additional events.
func militiaQueue(gs *engine.GameState, attacker engine.PlayerIdx, victims []engine.PlayerIdx) []engine.Event {
	for i, v := range victims {
		handSize := len(gs.Players[v].Hand)
		if handSize <= 3 {
			continue
		}
		n := handSize - 3
		rest := append([]engine.PlayerIdx(nil), victims[i+1:]...)
		return engine.RequestDecision(gs, v, "militia", 0,
			engine.DiscardFromHandPrompt{Min: n, Max: n},
			map[engine.ContextKey]any{
				engine.CtxKeyAttacker:         attacker,
				engine.CtxKeyRemainingVictims: rest,
			})
	}
	return nil
}

func init() {
	DefaultRegistry.Register(Militia)
}
