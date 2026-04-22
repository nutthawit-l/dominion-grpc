package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Harbinger = &engine.Card{
	ID:    "harbinger",
	Name:  "Harbinger",
	Cost:  3,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.DrawCards(gs, px, 1)
		events = append(events, engine.AddActions(gs, px, 1)...)
		if len(gs.Players[px].Discard) == 0 {
			return events
		}
		discardCopy := make([]engine.CardID, len(gs.Players[px].Discard))
		copy(discardCopy, gs.Players[px].Discard)
		events = append(events, engine.RequestDecision(gs, px, "harbinger", 0,
			engine.ChooseFromDiscardPrompt{Cards: discardCopy, Optional: true}, nil)...)
		return events
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		choice := answer.(engine.CardChoiceAnswer)
		if choice.None {
			return nil, nil
		}
		ps := &gs.Players[px]
		idx := engine.IndexOf(ps.Discard, choice.Card)
		if idx < 0 {
			return nil, nil
		}
		ps.Discard = append(ps.Discard[:idx], ps.Discard[idx+1:]...)
		ps.Deck = append(ps.Deck, choice.Card)
		return nil, nil
	},
}

func init() {
	DefaultRegistry.Register(Harbinger)
}
