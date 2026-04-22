package engine

// Action is the engine's internal action union. The service layer
// translates protobuf actions into these before calling Apply.
type Action interface {
	isAction()
	Player() PlayerIdx
}

// PlayCard plays a specific card from hand.
type PlayCard struct {
	PlayerIdx PlayerIdx
	Card      CardID
}

func (PlayCard) isAction()             {}
func (act PlayCard) Player() PlayerIdx { return act.PlayerIdx }

// BuyCard buys a specific card from the supply.
type BuyCard struct {
	PlayerIdx PlayerIdx
	Card      CardID
}

func (BuyCard) isAction()            {}
func (act BuyCard) Player() PlayerIdx { return act.PlayerIdx }

// EndPhase ends the current phase (Action → Buy, or Buy → Cleanup).
// Cleanup auto-advances to the next player; EndPhase cannot be called
// during Cleanup.
type EndPhase struct {
	PlayerIdx PlayerIdx
}

func (EndPhase) isAction()             {}
func (act EndPhase) Player() PlayerIdx { return act.PlayerIdx }

// ResolveDecision answers a pending Decision.
type ResolveDecision struct {
	PlayerIdx  PlayerIdx
	DecisionID string
	Answer     Answer
}

func (ResolveDecision) isAction()             {}
func (act ResolveDecision) Player() PlayerIdx { return act.PlayerIdx }
