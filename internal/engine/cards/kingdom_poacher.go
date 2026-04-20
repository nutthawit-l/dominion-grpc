package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Poacher = &engine.Card{
	ID:    "poacher",
	Name:  "Poacher",
	Cost:  4,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		events := engine.DrawCards(s, p, 1)
		events = append(events, engine.AddActions(s, p, 1)...)
		events = append(events, engine.AddCoins(s, p, 1)...)

		emptyPiles := 0
		for _, n := range s.Supply.Piles {
			if n <= 0 {
				emptyPiles++
			}
		}
		if emptyPiles == 0 {
			return events
		}
		events = append(events, engine.RequestDecision(s, p, "poacher", 0,
			engine.DiscardFromHandPrompt{Min: emptyPiles, Max: emptyPiles}, nil)...)
		return events
	},
	OnResolve: func(s *engine.GameState, p int, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		cards := answer.(engine.CardListAnswer).Cards
		return engine.DiscardFromHand(s, p, cards), nil
	},
}

func init() {
	DefaultRegistry.Register(Poacher)
}
