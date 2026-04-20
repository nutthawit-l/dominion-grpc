package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Vassal = &engine.Card{
	ID:    "vassal",
	Name:  "Vassal",
	Cost:  3,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		revealed, events := engine.RevealAndDiscardFromDeck(s, p, 1)
		if len(revealed) == 0 {
			return events
		}
		card := revealed[0]
		c, ok := DefaultRegistry.Lookup(card)
		if !ok || !c.HasType(engine.TypeAction) {
			return events
		}
		events = append(events, engine.RequestDecision(s, p, "vassal", 0,
			engine.MayPlayActionPrompt{Card: card},
			map[string]any{"card": card})...)
		return events
	},
	OnResolve: func(s *engine.GameState, p int, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		yn := answer.(engine.YesNoAnswer)
		if !yn.Yes {
			return nil, nil
		}
		cardID := d.Context["card"].(engine.CardID)
		return engine.PlayCardFromZone(s, p, cardID, engine.ZoneDiscard, lookup)
	},
}

func init() {
	DefaultRegistry.Register(Vassal)
}
