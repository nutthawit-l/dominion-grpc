package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Cellar = &engine.Card{
	ID:    "cellar",
	Name:  "Cellar",
	Cost:  2,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.AddActions(gs, px, 1)
		handSize := len(gs.Players[px].Hand)
		events = append(events, engine.RequestDecision(gs, px, "cellar", 0,
			engine.DiscardFromHandPrompt{Min: 0, Max: handSize}, nil)...)
		return events
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		cards := answer.(engine.CardListAnswer).Cards
		events := engine.DiscardFromHand(gs, px, cards)
		events = append(events, engine.DrawCards(gs, px, len(cards))...)
		return events, nil
	},
}

func init() {
	DefaultRegistry.Register(Cellar)
}
