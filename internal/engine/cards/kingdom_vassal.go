package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Vassal = &engine.Card{
	ID:    "vassal",
	Name:  "Vassal",
	Cost:  3,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		revealed, events := engine.RevealAndDiscardFromDeck(gs, px, 1)
		if len(revealed) == 0 {
			return events
		}
		card := revealed[0]
		c, ok := DefaultRegistry.Lookup(card)
		if !ok || !c.HasType(engine.TypeAction) {
			return events
		}
		events = append(events, engine.RequestDecision(gs, px, "vassal", 0,
			engine.MayPlayActionPrompt{Card: card},
			map[engine.ContextKey]any{engine.CtxKeyCard: card})...)
		return events
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		yn := answer.(engine.YesNoAnswer)
		if !yn.Yes {
			return nil, nil
		}
		cardID := d.Context[engine.CtxKeyCard].(engine.CardID)
		return engine.PlayCardFromZone(gs, px, cardID, engine.ZoneDiscard, lookup)
	},
}

func init() {
	DefaultRegistry.Register(Vassal)
}
