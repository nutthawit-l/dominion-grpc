package engine

// Action is the engine's internal action union. The service layer
// translates protobuf actions into these before calling Apply.
type Action interface {
	isAction()
	Player() int
}

// PlayCard plays a specific card from hand.
type PlayCard struct {
	PlayerIdx int
	Card      CardID
}

func (PlayCard) isAction()     {}
func (a PlayCard) Player() int { return a.PlayerIdx }

// BuyCard buys a specific card from the supply.
type BuyCard struct {
	PlayerIdx int
	Card      CardID
}

func (BuyCard) isAction()     {}
func (a BuyCard) Player() int { return a.PlayerIdx }

// EndPhase ends the current phase (Action → Buy, or Buy → Cleanup).
// Cleanup auto-advances to the next player; EndPhase cannot be called
// during Cleanup.
type EndPhase struct {
	PlayerIdx int
}

func (EndPhase) isAction()     {}
func (a EndPhase) Player() int { return a.PlayerIdx }

// ResolveDecision answers a pending Decision.
type ResolveDecision struct {
	PlayerIdx  int
	DecisionID string
	Answer     Answer
}

func (ResolveDecision) isAction()     {}
func (a ResolveDecision) Player() int { return a.PlayerIdx }
