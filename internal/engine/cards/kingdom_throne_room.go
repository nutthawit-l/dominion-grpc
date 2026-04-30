package cards

import (
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var ThroneRoom = &engine.Card{
	ID:    "throne_room",
	Name:  "Throne Room",
	Cost:  4,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		// Find any Action in hand. If none, no-op.
		if !handHasAction(gs, px) {
			return nil
		}
		return engine.RequestDecision(gs, px, "throne_room", 0,
			engine.ChooseActionFromHandPrompt{}, nil)
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		choice := answer.(engine.CardChoiceAnswer)
		// Move chosen card from Hand → InPlay.
		idx := engine.IndexOf(gs.Players[px].Hand, choice.Card)
		if idx < 0 {
			return nil, engine.ErrCardNotInHand
		}
		ps := &gs.Players[px]
		ps.Hand = append(ps.Hand[:idx], ps.Hand[idx+1:]...)
		ps.InPlay = append(ps.InPlay, choice.Card)
		events := []engine.Event{{Kind: engine.EventCardPlayed, PlayerIdx: px, CardID: choice.Card}}

		// Queue the second play BEFORE running the first — Apply pops
		// LIFO, so the second play runs after the first's chained
		// decisions (if any) all resolve.
		gs.PendingPlays = append(gs.PendingPlays, engine.PendingPlay{
			PlayerIdx: px, CardID: choice.Card, Source: "throne_room",
		})

		// First play.
		c, ok := lookup(choice.Card)
		if !ok {
			return events, engine.ErrUnknownCard
		}
		if c.OnPlay != nil {
			events = append(events, c.OnPlay(gs, px)...)
		}
		return events, nil
	},
}

// handHasAction returns true iff the player has at least one Action card
// in hand. Throne Room uses this to decide whether to prompt at all.
func handHasAction(gs *engine.GameState, px engine.PlayerIdx) bool {
	for _, c := range gs.Players[px].Hand {
		card, ok := DefaultRegistry.Lookup(c)
		if ok && card.HasType(engine.TypeAction) {
			return true
		}
	}
	return false
}

func init() {
	DefaultRegistry.Register(ThroneRoom)
}
