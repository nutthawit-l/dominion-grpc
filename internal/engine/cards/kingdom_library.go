package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Library = &engine.Card{
	ID:    "library",
	Name:  "Library",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		return libraryStep(gs, px)
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		yn, ok := answer.(engine.YesNoAnswer)
		if !ok {
			return nil, fmt.Errorf("library: expected YesNoAnswer, got %T", answer)
		}
		drawn, ok := d.Context[engine.CtxKeyCard].(engine.CardID)
		if !ok {
			return nil, fmt.Errorf("library: CtxKeyCard missing or wrong type in context")
		}

		var events []engine.Event
		if !yn.Yes {
			// Player wants to keep the action card — move it from SetAside → Hand.
			ps := &gs.Players[px]
			idx := engine.IndexOf(ps.SetAside, drawn)
			if idx >= 0 {
				ps.SetAside = append(ps.SetAside[:idx], ps.SetAside[idx+1:]...)
			}
			ps.Hand = append(ps.Hand, drawn)
		}
		// If Yes, leave the card in SetAside — it will be flushed to Discard
		// when the loop ends.
		events = append(events, libraryStep(gs, px)...)
		return events, nil
	},
}

// libraryStep draws until the player's hand reaches 7 cards, parking a
// SetAsideActionPrompt for any Action card drawn along the way. When
// hand hits 7 or both deck+discard are empty, flushes SetAside → Discard.
//
// Action cards are drawn via DrawCards (which goes to hand) and then
// immediately moved from Hand → SetAside before requesting the decision.
// The player answers Yes (leave in SetAside) or No (move back to Hand)
// via OnResolve before the loop continues.
func libraryStep(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
	var events []engine.Event
	for {
		ps := &gs.Players[px]
		if len(ps.Hand) >= 7 {
			events = append(events, flushSetAsideToDiscard(gs, px)...)
			return events
		}
		if len(ps.Deck) == 0 && len(ps.Discard) == 0 {
			events = append(events, flushSetAsideToDiscard(gs, px)...)
			return events
		}
		// Draw one card. DrawCards handles deck-empty reshuffle automatically
		// and puts the card in Hand.
		handBefore := len(ps.Hand)
		drawEvents := engine.DrawCards(gs, px, 1)
		events = append(events, drawEvents...)
		// If nothing was drawn (shouldn't happen given checks above), stop.
		if len(gs.Players[px].Hand) == handBefore {
			events = append(events, flushSetAsideToDiscard(gs, px)...)
			return events
		}
		ps = &gs.Players[px]
		drawn := ps.Hand[len(ps.Hand)-1]

		card, ok := DefaultRegistry.Lookup(drawn)
		if ok && card.HasType(engine.TypeAction) {
			// Action card: move from Hand → SetAside, then ask player.
			ps.Hand = ps.Hand[:len(ps.Hand)-1]
			ps.SetAside = append(ps.SetAside, drawn)
			events = append(events, engine.RequestDecision(gs, px, "library", 0,
				engine.SetAsideActionPrompt{Card: drawn},
				map[engine.ContextKey]any{engine.CtxKeyCard: drawn})...)
			return events
		}
		// Non-action: stays in hand, continue loop.
	}
}

// flushSetAsideToDiscard moves all SetAside cards to Discard.
func flushSetAsideToDiscard(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
	ps := &gs.Players[px]
	var events []engine.Event
	for _, c := range ps.SetAside {
		ps.Discard = append(ps.Discard, c)
		events = append(events, engine.Event{
			Kind: engine.EventCardDiscarded, PlayerIdx: px, CardID: c,
		})
	}
	ps.SetAside = nil
	return events
}

func init() {
	DefaultRegistry.Register(Library)
}
