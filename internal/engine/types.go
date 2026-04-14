package engine

// CardID is the stable string identifier for a card definition.
type CardID string

// CardType tags a card with one or more categorical roles.
type CardType int

const (
	TypeUnknown CardType = iota
	TypeTreasure
	TypeVictory
	TypeCurse
	TypeAction
	TypeAttack
	TypeReaction
)

// Phase is the phase of the current player's turn.
type Phase int

const (
	PhaseAction Phase = iota + 1
	PhaseBuy
	PhaseCleanup
)

// GainDest says where a gained card should land.
type GainDest int

const (
	GainToDiscard GainDest = iota
	GainToHand
	GainToDeck
)
