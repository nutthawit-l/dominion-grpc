package engine

import (
	"math/rand"
)

// GameState is the complete authoritative game state. It is the only
// input/output of the engine.
type GameState struct {
	GameID        string
	Seed          int64
	rng           *rand.Rand
	Players       []PlayerState
	CurrentPlayer  int
	StartingPlayer int
	Phase          Phase
	Supply        Supply
	Trash         []CardID
	Turn          int
	Ended         bool
	Winners       []int

	// PendingDecision is set by RequestDecision when a card needs
	// player input. While set, only ResolveDecision is legal.
	PendingDecision *Decision

	// DecisionSeq is a monotonic counter for generating deterministic
	// decision IDs. Incremented by RequestDecision.
	DecisionSeq uint64
}

// PlayerState tracks one player's zones and resources.
type PlayerState struct {
	Name    string
	Hand    []CardID
	Deck    []CardID // top of deck = end of slice
	Discard []CardID
	InPlay  []CardID
	Actions int
	Buys    int
	Coins   int
}

// Supply tracks pile counts by card ID.
type Supply struct {
	Piles map[CardID]int
}

// Decision is a server-generated prompt requiring player input.
type Decision struct {
	ID        string
	PlayerIdx int
	CardID    CardID
	Step      int
	Prompt    Prompt
	Context   map[string]any
}

// Event is an engine-level notification of something that happened.
// The service layer translates these into protobuf GameEvents.
type Event struct {
	Kind      EventKind
	PlayerIdx int
	CardID    CardID
	Count     int
	Phase     Phase
}

type EventKind int

const (
	EventUnknown EventKind = iota
	EventCardDrawn
	EventCardDiscarded
	EventCardPlayed
	EventCardGained
	EventCardTrashed
	EventCoinsAdded
	EventBuysAdded
	EventActionsAdded
	EventPhaseChanged
	EventTurnStarted
	EventGameEnded
	EventDecisionRequested
)

// RNG exposes the per-game random source for code inside the engine
// package. External code must never reach the underlying field.
func (s *GameState) RNG() *rand.Rand { return s.rng }
