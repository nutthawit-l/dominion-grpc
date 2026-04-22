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
func Apply(s *GameState, act Action, lookup CardLookup) (*GameState, []Event, error) {
	if s.Ended {
		return s, nil, ErrGameEnded
	}

	// Decision-pending guard: only ResolveDecision is legal while a
	// decision is pending.
	if s.PendingDecision != nil {
		resolve, ok := act.(ResolveDecision)
		if !ok {
			return s, nil, ErrDecisionPending
		}
		ev, err := applyResolveDecision(s, resolve, lookup)
		return s, ev, err
	}

	if act.Player() != s.CurrentPlayer {
		return s, nil, ErrNotYourTurn
	}
	switch act := act.(type) {
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
		return s, nil, ErrNoDecisionPending
	default:
		return s, nil, ErrUnknownAction
	}
}

func applyResolveDecision(s *GameState, act ResolveDecision, lookup CardLookup) ([]Event, error) {
	d := s.PendingDecision
	if d == nil {
		return nil, ErrNoDecisionPending
	}
	if d.ID != act.DecisionID {
		return nil, ErrWrongDecisionID
	}
	if act.PlayerIdx != d.PlayerIdx {
		return nil, ErrNotYourTurn
	}
	card, ok := lookup(d.CardID)
	if !ok {
		return nil, ErrUnknownCard
	}
	if card.OnResolve == nil {
		return nil, ErrNoResolveHandler
	}
	s.PendingDecision = nil
	return card.OnResolve(s, d.PlayerIdx, d, act.Answer, lookup)
}

func applyPlayCard(s *GameState, act PlayCard, lookup CardLookup) ([]Event, error) {
	card, ok := lookup(act.Card)
	if !ok {
		return nil, ErrUnknownCard
	}
	switch s.Phase {
	case PhaseAction:
		if !card.HasType(TypeAction) {
			return nil, ErrWrongPhase
		}
		if s.Players[act.PlayerIdx].Actions <= 0 {
			return nil, ErrNoActions
		}
		s.Players[act.PlayerIdx].Actions--
	case PhaseBuy:
		if !card.HasType(TypeTreasure) {
			return nil, ErrWrongPhase
		}
	default:
		return nil, ErrWrongPhase
	}
	return PlayCardFromZone(s, act.PlayerIdx, act.Card, ZoneHand, lookup)
}

func applyBuyCard(s *GameState, act BuyCard, lookup CardLookup) ([]Event, error) {
	if s.Phase != PhaseBuy {
		return nil, ErrWrongPhase
	}
	card, ok := lookup(act.Card)
	if !ok {
		return nil, ErrUnknownCard
	}
	if s.Supply.Piles[act.Card] <= 0 {
		return nil, ErrCardNotInSupply
	}
	if s.Players[act.PlayerIdx].Buys <= 0 {
		return nil, ErrNoBuys
	}
	if s.Players[act.PlayerIdx].Coins < card.Cost {
		return nil, ErrInsufficientCoins
	}
	s.Players[act.PlayerIdx].Coins -= card.Cost
	s.Players[act.PlayerIdx].Buys--
	events := GainCard(s, act.PlayerIdx, act.Card, GainToDiscard)
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
