package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Bureaucrat = &engine.Card{
	ID:    "bureaucrat",
	Name:  "Bureaucrat",
	Cost:  4,
	Types: []engine.CardType{engine.TypeAction, engine.TypeAttack},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.GainCard(gs, px, "silver", engine.GainToDeck)
		victims, attackEvents := engine.ResolveAttackVictims(gs, px, "bureaucrat", registryLookup)
		events = append(events, attackEvents...)
		events = append(events, bureaucratQueue(gs, px, victims)...)
		return events
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		chosen := answer.(engine.CardListAnswer).Cards
		if len(chosen) != 1 {
			return nil, fmt.Errorf("bureaucrat: must choose exactly 1 Victory, got %d", len(chosen))
		}
		card, ok := lookup(chosen[0])
		if !ok {
			return nil, fmt.Errorf("bureaucrat: unknown card %q", chosen[0])
		}
		if !card.HasType(engine.TypeVictory) {
			return nil, fmt.Errorf("bureaucrat: chosen card %q is not a Victory", chosen[0])
		}
		if engine.IndexOf(gs.Players[px].Hand, chosen[0]) < 0 {
			return nil, fmt.Errorf("bureaucrat: card %q not in hand", chosen[0])
		}
		events := engine.PutOnDeck(gs, px, chosen)
		attacker := d.Context[engine.CtxKeyAttacker].(engine.PlayerIdx)
		remaining := d.Context[engine.CtxKeyRemainingVictims].([]engine.PlayerIdx)
		events = append(events, bureaucratQueue(gs, attacker, remaining)...)
		return events, nil
	},
}

// bureaucratQueue advances through the victim list, handling 0-Victory
// (reveal only) and 1-Victory (forced put-on-deck) cases inline and
// parking a decision on the first victim with 2+ Victories.
func bureaucratQueue(gs *engine.GameState, attacker engine.PlayerIdx, victims []engine.PlayerIdx) []engine.Event {
	var events []engine.Event
	for i, v := range victims {
		victories := victoriesInHand(gs, v)
		switch len(victories) {
		case 0:
			events = append(events, engine.Event{
				Kind:      engine.EventCardRevealed,
				PlayerIdx: v,
			})
		case 1:
			events = append(events, engine.PutOnDeck(gs, v, victories)...)
		default:
			rest := append([]engine.PlayerIdx(nil), victims[i+1:]...)
			events = append(events, engine.RequestDecision(gs, v, "bureaucrat", 0,
				engine.PutOnDeckPrompt{TypeFilter: []engine.CardType{engine.TypeVictory}},
				map[engine.ContextKey]any{
					engine.CtxKeyAttacker:         attacker,
					engine.CtxKeyRemainingVictims: rest,
				})...)
			return events
		}
	}
	return events
}

func victoriesInHand(gs *engine.GameState, px engine.PlayerIdx) []engine.CardID {
	var out []engine.CardID
	for _, id := range gs.Players[px].Hand {
		card, ok := DefaultRegistry.Lookup(id)
		if !ok {
			continue
		}
		if card.HasType(engine.TypeVictory) {
			out = append(out, id)
		}
	}
	return out
}

func init() {
	DefaultRegistry.Register(Bureaucrat)
}
