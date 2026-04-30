package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Sentry = &engine.Card{
	ID:    "sentry",
	Name:  "Sentry",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.DrawCards(gs, px, 1)
		events = append(events, engine.AddActions(gs, px, 1)...)

		revealed := peelToSetAside(gs, px, 2)
		if len(revealed) == 0 {
			return events
		}
		events = append(events, engine.RequestDecision(gs, px, "sentry", 0,
			engine.TrashFromRevealedPrompt{Cards: revealed}, nil)...)
		return events
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		choice, ok := answer.(engine.CardListAnswer)
		if !ok {
			return nil, fmt.Errorf("sentry step %d: expected CardListAnswer, got %T", d.Step, answer)
		}
		ps := &gs.Players[px]
		var events []engine.Event

		switch d.Step {
		case 0: // trash
			for _, c := range choice.Cards {
				idx := engine.IndexOf(ps.SetAside, c)
				if idx < 0 {
					continue
				}
				ps.SetAside = append(ps.SetAside[:idx], ps.SetAside[idx+1:]...)
				gs.Trash = append(gs.Trash, c)
				events = append(events, engine.Event{
					Kind: engine.EventCardTrashed, PlayerIdx: px, CardID: c,
				})
			}
			return sentryAdvanceFromStep0(gs, px, events), nil

		case 1: // discard
			for _, c := range choice.Cards {
				idx := engine.IndexOf(ps.SetAside, c)
				if idx < 0 {
					continue
				}
				ps.SetAside = append(ps.SetAside[:idx], ps.SetAside[idx+1:]...)
				ps.Discard = append(ps.Discard, c)
				events = append(events, engine.Event{
					Kind: engine.EventCardDiscarded, PlayerIdx: px, CardID: c,
				})
			}
			return sentryAdvanceFromStep1(gs, px, events), nil

		case 2: // reorder
			for _, c := range choice.Cards {
				idx := engine.IndexOf(ps.SetAside, c)
				if idx < 0 {
					return events, fmt.Errorf("sentry step 2: card %q not in SetAside", c)
				}
				ps.SetAside = append(ps.SetAside[:idx], ps.SetAside[idx+1:]...)
				ps.Deck = append(ps.Deck, c)
				events = append(events, engine.Event{
					Kind: engine.EventCardPutOnDeck, PlayerIdx: px, CardID: c,
				})
			}
			// Defensive flush: any cards not named in the answer go on deck too.
			for _, c := range ps.SetAside {
				ps.Deck = append(ps.Deck, c)
				events = append(events, engine.Event{
					Kind: engine.EventCardPutOnDeck, PlayerIdx: px, CardID: c,
				})
			}
			ps.SetAside = nil
			return events, nil
		}
		return events, nil
	},
}

// peelToSetAside moves up to n cards from the top of the deck into SetAside.
// If deck has fewer than n cards and the discard pile is non-empty, it
// shuffles the discard into the deck (once) and continues. Returns the
// CardIDs moved in top-first order (first element = topmost card drawn).
func peelToSetAside(gs *engine.GameState, px engine.PlayerIdx, n int) []engine.CardID {
	ps := &gs.Players[px]
	if len(ps.Deck) < n && len(ps.Discard) > 0 {
		ps.Deck = append(ps.Deck, ps.Discard...)
		ps.Discard = nil
		rng := gs.RNG()
		if rng != nil {
			rng.Shuffle(len(ps.Deck), func(i, j int) {
				ps.Deck[i], ps.Deck[j] = ps.Deck[j], ps.Deck[i]
			})
		}
	}
	out := make([]engine.CardID, 0, n)
	for i := 0; i < n && len(ps.Deck) > 0; i++ {
		top := len(ps.Deck) - 1
		c := ps.Deck[top]
		ps.Deck = ps.Deck[:top]
		ps.SetAside = append(ps.SetAside, c)
		out = append(out, c)
	}
	return out
}

// sentryAdvanceFromStep0 issues the DiscardFromRevealedPrompt (step 1) if
// any cards remain in SetAside after trashing; otherwise terminates.
func sentryAdvanceFromStep0(gs *engine.GameState, px engine.PlayerIdx, prior []engine.Event) []engine.Event {
	ps := &gs.Players[px]
	if len(ps.SetAside) == 0 {
		return prior
	}
	cards := append([]engine.CardID(nil), ps.SetAside...)
	return append(prior, engine.RequestDecision(gs, px, "sentry", 1,
		engine.DiscardFromRevealedPrompt{Cards: cards}, nil)...)
}

// sentryAdvanceFromStep1 handles the end of the discard step:
//   - 0 cards left → done.
//   - 1 card left → silently put on deck, no reorder prompt.
//   - 2 cards left → issue ReorderCardsPrompt (step 2).
func sentryAdvanceFromStep1(gs *engine.GameState, px engine.PlayerIdx, prior []engine.Event) []engine.Event {
	ps := &gs.Players[px]
	switch len(ps.SetAside) {
	case 0:
		return prior
	case 1:
		c := ps.SetAside[0]
		ps.SetAside = nil
		ps.Deck = append(ps.Deck, c)
		return append(prior, engine.Event{
			Kind: engine.EventCardPutOnDeck, PlayerIdx: px, CardID: c,
		})
	default:
		cards := append([]engine.CardID(nil), ps.SetAside...)
		return append(prior, engine.RequestDecision(gs, px, "sentry", 2,
			engine.ReorderCardsPrompt{Cards: cards}, nil)...)
	}
}

func init() {
	DefaultRegistry.Register(Sentry)
}
