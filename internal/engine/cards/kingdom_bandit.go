package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Bandit = &engine.Card{
	ID:    "bandit",
	Name:  "Bandit",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction, engine.TypeAttack},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.GainCard(gs, px, "gold", engine.GainToDiscard)
		victims, attackEvents := engine.ResolveAttackVictims(gs, px, "bandit", registryLookup)
		events = append(events, attackEvents...)
		events = append(events, banditQueue(gs, px, victims)...)
		return events
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		choice := answer.(engine.CardChoiceAnswer)
		revealed := d.Context[engine.CtxKeyRevealedCards].([]engine.CardID)
		if !containsCard(revealed, choice.Card) {
			return nil, fmt.Errorf("bandit: card %q not in revealed list", choice.Card)
		}
		card, ok := lookup(choice.Card)
		if !ok {
			return nil, fmt.Errorf("bandit: unknown card %q", choice.Card)
		}
		if !card.HasType(engine.TypeTreasure) || choice.Card == "copper" {
			return nil, fmt.Errorf("bandit: chosen card %q is not a non-Copper Treasure", choice.Card)
		}
		events := engine.TrashFromDeck(gs, px, choice.Card)
		for _, c := range revealed {
			if c == choice.Card {
				continue
			}
			events = append(events, engine.DiscardFromDeck(gs, px, c)...)
		}
		attacker := d.Context[engine.CtxKeyAttacker].(engine.PlayerIdx)
		remaining := d.Context[engine.CtxKeyRemainingVictims].([]engine.PlayerIdx)
		events = append(events, banditQueue(gs, attacker, remaining)...)
		return events, nil
	},
}

// banditQueue advances through the victim list. For each victim:
// reveal top 2; zero non-Copper Treasures revealed → discard both;
// one → auto-trash, discard other; two → prompt.
func banditQueue(gs *engine.GameState, attacker engine.PlayerIdx, victims []engine.PlayerIdx) []engine.Event {
	var events []engine.Event
	for i, v := range victims {
		revealed, revealEvents := engine.RevealFromDeck(gs, v, 2)
		events = append(events, revealEvents...)
		events = append(events, engine.Event{Kind: engine.EventCardRevealed, PlayerIdx: v})

		treasures := nonCopperTreasures(revealed)
		switch len(treasures) {
		case 0:
			for _, c := range revealed {
				events = append(events, engine.DiscardFromDeck(gs, v, c)...)
			}
		case 1:
			events = append(events, engine.TrashFromDeck(gs, v, treasures[0])...)
			for _, c := range revealed {
				if c == treasures[0] {
					continue
				}
				events = append(events, engine.DiscardFromDeck(gs, v, c)...)
			}
		default:
			rest := append([]engine.PlayerIdx(nil), victims[i+1:]...)
			events = append(events, engine.RequestDecision(gs, v, "bandit", 0,
				engine.TrashFromRevealedPrompt{Cards: treasures},
				map[engine.ContextKey]any{
					engine.CtxKeyAttacker:         attacker,
					engine.CtxKeyRemainingVictims: rest,
					engine.CtxKeyRevealedCards:    revealed,
				})...)
			return events
		}
	}
	return events
}

func nonCopperTreasures(cards []engine.CardID) []engine.CardID {
	var out []engine.CardID
	for _, id := range cards {
		if id == "copper" {
			continue
		}
		card, ok := DefaultRegistry.Lookup(id)
		if !ok {
			continue
		}
		if card.HasType(engine.TypeTreasure) {
			out = append(out, id)
		}
	}
	return out
}

func containsCard(cards []engine.CardID, target engine.CardID) bool {
	for _, c := range cards {
		if c == target {
			return true
		}
	}
	return false
}

func init() {
	DefaultRegistry.Register(Bandit)
}
