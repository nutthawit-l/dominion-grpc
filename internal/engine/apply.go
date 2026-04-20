package engine

import "errors"

var (
	ErrNotYourTurn       = errors.New("engine: not your turn")
	ErrWrongPhase        = errors.New("engine: wrong phase for this action")
	ErrCardNotInHand     = errors.New("engine: card not in hand")
	ErrCardNotInSupply   = errors.New("engine: card not in supply")
	ErrInsufficientCoins = errors.New("engine: insufficient coins")
	ErrNoBuys            = errors.New("engine: no buys remaining")
	ErrNoActions         = errors.New("engine: no actions remaining")
	ErrUnknownCard       = errors.New("engine: unknown card")
	ErrGameEnded         = errors.New("engine: game has ended")
	ErrUnknownAction     = errors.New("engine: unknown action type")
	ErrCardNotInDiscard  = errors.New("engine: card not in discard")
	ErrNoDecisionPending = errors.New("engine: no decision pending")
	ErrWrongDecisionID   = errors.New("engine: wrong decision ID")
	ErrDecisionPending   = errors.New("engine: decision pending — only ResolveDecision is legal")
	ErrNoResolveHandler  = errors.New("engine: card has no OnResolve handler")
)

// Apply is the engine's only entry point. It mutates s in place and
// returns the events that resulted. On an illegal action it returns an
// error and leaves s in a consistent pre-action state (because the
// handlers validate before mutating).
func Apply(s *GameState, a Action, lookup CardLookup) (*GameState, []Event, error) {
	if s.Ended {
		return s, nil, ErrGameEnded
	}
	if a.Player() != s.CurrentPlayer {
		return s, nil, ErrNotYourTurn
	}
	switch act := a.(type) {
	case PlayCard:
		ev, err := applyPlayCard(s, act, lookup)
		return s, ev, err
	case BuyCard:
		ev, err := applyBuyCard(s, act, lookup)
		return s, ev, err
	case EndPhase:
		ev, err := applyEndPhase(s, lookup)
		return s, ev, err
	case ResolveDecision:
		return s, nil, ErrUnknownAction // Tier 2 adds this path
	default:
		return s, nil, ErrUnknownAction
	}
}

func applyPlayCard(s *GameState, a PlayCard, lookup CardLookup) ([]Event, error) {
	card, ok := lookup(a.Card)
	if !ok {
		return nil, ErrUnknownCard
	}
	if indexOf(s.Players[a.PlayerIdx].Hand, a.Card) < 0 {
		return nil, ErrCardNotInHand
	}
	// In the Action phase, only action cards may be played, and only
	// if the player has actions available. In the Buy phase, only
	// treasures may be played.
	switch s.Phase {
	case PhaseAction:
		if !card.HasType(TypeAction) {
			return nil, ErrWrongPhase
		}
		if s.Players[a.PlayerIdx].Actions <= 0 {
			return nil, ErrNoActions
		}
		s.Players[a.PlayerIdx].Actions--
	case PhaseBuy:
		if !card.HasType(TypeTreasure) {
			return nil, ErrWrongPhase
		}
	default:
		return nil, ErrWrongPhase
	}
	// Move the card from hand to in-play.
	idx := indexOf(s.Players[a.PlayerIdx].Hand, a.Card)
	s.Players[a.PlayerIdx].Hand = append(s.Players[a.PlayerIdx].Hand[:idx], s.Players[a.PlayerIdx].Hand[idx+1:]...)
	s.Players[a.PlayerIdx].InPlay = append(s.Players[a.PlayerIdx].InPlay, a.Card)
	events := []Event{{Kind: EventCardPlayed, PlayerIdx: a.PlayerIdx, CardID: a.Card}}
	if card.OnPlay != nil {
		events = append(events, card.OnPlay(s, a.PlayerIdx)...)
	}
	return events, nil
}

func applyBuyCard(s *GameState, a BuyCard, lookup CardLookup) ([]Event, error) {
	if s.Phase != PhaseBuy {
		return nil, ErrWrongPhase
	}
	card, ok := lookup(a.Card)
	if !ok {
		return nil, ErrUnknownCard
	}
	if s.Supply.Piles[a.Card] <= 0 {
		return nil, ErrCardNotInSupply
	}
	if s.Players[a.PlayerIdx].Buys <= 0 {
		return nil, ErrNoBuys
	}
	if s.Players[a.PlayerIdx].Coins < card.Cost {
		return nil, ErrInsufficientCoins
	}
	s.Players[a.PlayerIdx].Coins -= card.Cost
	s.Players[a.PlayerIdx].Buys--
	events := GainCard(s, a.PlayerIdx, a.Card, GainToDiscard)
	return events, nil
}

func applyEndPhase(s *GameState, lookup CardLookup) ([]Event, error) {
	switch s.Phase {
	case PhaseAction:
		s.Phase = PhaseBuy
		return []Event{{Kind: EventPhaseChanged, PlayerIdx: s.CurrentPlayer, Phase: PhaseBuy}}, nil
	case PhaseBuy:
		// Cleanup + advance turn.
		s.Phase = PhaseCleanup
		events := cleanupAndEndTurn(s)
		// Check game-over AFTER the buy concluded (which is where piles
		// actually emptied) and BEFORE the new player starts acting.
		if IsGameOver(s) {
			s.Ended = true
			scores := make([]int, len(s.Players))
			for i, p := range s.Players {
				scores[i] = ComputeScore(p, lookup)
			}
			s.Winners = DetermineWinners(scores)
			events = append(events, Event{Kind: EventGameEnded})
		}
		return events, nil
	default:
		return nil, ErrWrongPhase
	}
}
