package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Moneylender = &engine.Card{
	ID:    "moneylender",
	Name:  "Moneylender",
	Cost:  4,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		if engine.IndexOf(gs.Players[px].Hand, "copper") < 0 {
			return nil
		}
		return engine.RequestDecision(gs, px, "moneylender", 0,
			engine.TrashFromHandPrompt{Min: 0, Max: 1, CardFilter: []engine.CardID{"copper"}}, nil)
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		cards := answer.(engine.CardListAnswer).Cards
		if len(cards) == 0 {
			return nil, nil
		}
		events := engine.TrashFromHand(gs, px, cards)
		events = append(events, engine.AddCoins(gs, px, 3)...)
		return events, nil
	},
}

func init() {
	DefaultRegistry.Register(Moneylender)
}
