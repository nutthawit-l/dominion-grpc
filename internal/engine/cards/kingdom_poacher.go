package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Poacher = &engine.Card{
	ID:    "poacher",
	Name:  "Poacher",
	Cost:  4,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.DrawCards(gs, px, 1)
		events = append(events, engine.AddActions(gs, px, 1)...)
		events = append(events, engine.AddCoins(gs, px, 1)...)

		emptyPiles := 0
		for _, n := range gs.Supply.Piles {
			if n <= 0 {
				emptyPiles++
			}
		}
		if emptyPiles == 0 {
			return events
		}
		events = append(events, engine.RequestDecision(gs, px, "poacher", 0,
			engine.DiscardFromHandPrompt{Min: emptyPiles, Max: emptyPiles}, nil)...)
		return events
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		cards := answer.(engine.CardListAnswer).Cards
		return engine.DiscardFromHand(gs, px, cards), nil
	},
}

func init() {
	DefaultRegistry.Register(Poacher)
}
