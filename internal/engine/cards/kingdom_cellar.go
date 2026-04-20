package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Cellar = &engine.Card{
	ID:    "cellar",
	Name:  "Cellar",
	Cost:  2,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		events := engine.AddActions(s, p, 1)
		handSize := len(s.Players[p].Hand)
		events = append(events, engine.RequestDecision(s, p, "cellar", 0,
			engine.DiscardFromHandPrompt{Min: 0, Max: handSize}, nil)...)
		return events
	},
	OnResolve: func(s *engine.GameState, p int, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		cards := answer.(engine.CardListAnswer).Cards
		events := engine.DiscardFromHand(s, p, cards)
		events = append(events, engine.DrawCards(s, p, len(cards))...)
		return events, nil
	},
}

func init() {
	DefaultRegistry.Register(Cellar)
}
