package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Harbinger = &engine.Card{
	ID:    "harbinger",
	Name:  "Harbinger",
	Cost:  3,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(s *engine.GameState, p int) []engine.Event {
		events := engine.DrawCards(s, p, 1)
		events = append(events, engine.AddActions(s, p, 1)...)
		if len(s.Players[p].Discard) == 0 {
			return events
		}
		discardCopy := make([]engine.CardID, len(s.Players[p].Discard))
		copy(discardCopy, s.Players[p].Discard)
		events = append(events, engine.RequestDecision(s, p, "harbinger", 0,
			engine.ChooseFromDiscardPrompt{Cards: discardCopy, Optional: true}, nil)...)
		return events
	},
	OnResolve: func(s *engine.GameState, p int, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		choice := answer.(engine.CardChoiceAnswer)
		if choice.None {
			return nil, nil
		}
		ps := &s.Players[p]
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
