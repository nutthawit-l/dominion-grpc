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
func Apply(gs *GameState, act Action, lookup CardLookup) (*GameState, []Event, error) {
	if gs.Ended {
		return gs, nil, ErrGameEnded
	}

	// Decision-pending guard: only ResolveDecision is legal while a
	// decision is pending.
	if gs.PendingDecision != nil {
		resolve, ok := act.(ResolveDecision)
		if !ok {
			return gs, nil, ErrDecisionPending
		}
		ev, err := applyResolveDecision(gs, resolve, lookup)
		return gs, ev, err
	}

	if act.Player() != gs.CurrentPlayer {
		return gs, nil, ErrNotYourTurn
	}
	switch act := act.(type) {
	case PlayCard:
		ev, err := applyPlayCard(gs, act, lookup)
		return gs, ev, err
	case BuyCard:
		ev, err := applyBuyCard(gs, act, lookup)
		return gs, ev, err
	case EndPhase:
		ev, err := applyEndPhase(gs, lookup)
		return gs, ev, err
	case ResolveDecision:
		return gs, nil, ErrNoDecisionPending
	default:
		return gs, nil, ErrUnknownAction
	}
}

func applyResolveDecision(gs *GameState, act ResolveDecision, lookup CardLookup) ([]Event, error) {
	d := gs.PendingDecision
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
	gs.PendingDecision = nil
	return card.OnResolve(gs, d.PlayerIdx, d, act.Answer, lookup)
}

func applyPlayCard(gs *GameState, act PlayCard, lookup CardLookup) ([]Event, error) {
	card, ok := lookup(act.Card)
	if !ok {
		return nil, ErrUnknownCard
	}
	switch gs.Phase {
	case PhaseAction:
		if !card.HasType(TypeAction) {
			return nil, ErrWrongPhase
		}
		if gs.Players[act.PlayerIdx].Actions <= 0 {
			return nil, ErrNoActions
		}
		gs.Players[act.PlayerIdx].Actions--
	case PhaseBuy:
		if !card.HasType(TypeTreasure) {
			return nil, ErrWrongPhase
		}
	default:
		return nil, ErrWrongPhase
	}
	return PlayCardFromZone(gs, act.PlayerIdx, act.Card, ZoneHand, lookup)
}

func applyBuyCard(gs *GameState, act BuyCard, lookup CardLookup) ([]Event, error) {
	if gs.Phase != PhaseBuy {
		return nil, ErrWrongPhase
	}
	card, ok := lookup(act.Card)
	if !ok {
		return nil, ErrUnknownCard
	}
	if gs.Supply.Piles[act.Card] <= 0 {
		return nil, ErrCardNotInSupply
	}
	if gs.Players[act.PlayerIdx].Buys <= 0 {
		return nil, ErrNoBuys
	}
	if gs.Players[act.PlayerIdx].Coins < card.Cost {
		return nil, ErrInsufficientCoins
	}
	gs.Players[act.PlayerIdx].Coins -= card.Cost
	gs.Players[act.PlayerIdx].Buys--
	events := GainCard(gs, act.PlayerIdx, act.Card, GainToDiscard)
	return events, nil
}

func applyEndPhase(gs *GameState, lookup CardLookup) ([]Event, error) {
	switch gs.Phase {
	case PhaseAction:
		gs.Phase = PhaseBuy
		return []Event{{Kind: EventPhaseChanged, PlayerIdx: gs.CurrentPlayer, Phase: PhaseBuy}}, nil
	case PhaseBuy:
		// Cleanup + advance turn.
		gs.Phase = PhaseCleanup
		events := cleanupAndEndTurn(gs)
		// Check game-over AFTER the buy concluded (which is where piles
		// actually emptied) and BEFORE the new player starts acting.
		if IsGameOver(gs) {
			gs.Ended = true
			scores := make([]int, len(gs.Players))
			for i, p := range gs.Players {
				scores[i] = ComputeScore(p, lookup)
			}
			gs.Winners = DetermineWinners(scores)
			events = append(events, Event{Kind: EventGameEnded})
		}
		return events, nil
	default:
		return nil, ErrWrongPhase
	}
}
