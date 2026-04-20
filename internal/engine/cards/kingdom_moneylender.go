package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Moneylender = &engine.Card{
	ID:    "moneylender",
	Name:  "Moneylender",
	Cost:  4,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		if engine.IndexOf(s.Players[p].Hand, "copper") < 0 {
			return nil
		}
		return engine.RequestDecision(s, p, "moneylender", 0,
			engine.TrashFromHandPrompt{Min: 0, Max: 1, CardFilter: []engine.CardID{"copper"}}, nil)
	},
	OnResolve: func(s *engine.GameState, p int, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		cards := answer.(engine.CardListAnswer).Cards
		if len(cards) == 0 {
			return nil, nil
		}
		events := engine.TrashFromHand(s, p, cards)
		events = append(events, engine.AddCoins(s, p, 3)...)
		return events, nil
	},
}

func init() {
	DefaultRegistry.Register(Moneylender)
}
