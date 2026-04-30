package engine

import "fmt"

// Prompt is the interface for all decision prompts.
type Prompt interface{ isPrompt() }

// Answer is the interface for all decision answers.
type Answer interface{ isAnswer() }

// ContextKey names a value stored in Decision.Context. Using a typed
// string prevents accidental raw-string keys at compile time.
type ContextKey string

const (
	CtxKeyTrashedCost      ContextKey = "trashed_cost"
	CtxKeyCard             ContextKey = "card"
	CtxKeyAttacker         ContextKey = "attacker"
	CtxKeyRemainingVictims ContextKey = "remaining_victims"
	CtxKeyRevealedCards    ContextKey = "revealed_cards"
)

// --- Prompt types ---

// DiscardFromHandPrompt asks the player to discard cards from hand.
type DiscardFromHandPrompt struct {
	Min int
	Max int
}

func (DiscardFromHandPrompt) isPrompt() {}

// TrashFromHandPrompt asks the player to trash cards from hand.
type TrashFromHandPrompt struct {
	Min        int
	Max        int
	TypeFilter []CardType // e.g., [TypeTreasure] for Mine
	CardFilter []CardID   // e.g., ["copper"] for Moneylender
}

func (TrashFromHandPrompt) isPrompt() {}

// GainFromSupplyPrompt asks the player to gain a card from supply.
type GainFromSupplyPrompt struct {
	MaxCost    int
	TypeFilter []CardType
	Dest       GainDest
}

func (GainFromSupplyPrompt) isPrompt() {}

// ChooseFromDiscardPrompt asks the player to choose a card from their
// discard pile. Optional means they may decline.
type ChooseFromDiscardPrompt struct {
	Cards    []CardID
	Optional bool
}

func (ChooseFromDiscardPrompt) isPrompt() {}

// PutOnDeckPrompt asks the player to put one card from hand on top of
// their deck. TypeFilter narrows which hand cards are legal choices;
// an empty/nil filter means "any card."
type PutOnDeckPrompt struct {
	TypeFilter []CardType
}

func (PutOnDeckPrompt) isPrompt() {}

// MayPlayActionPrompt asks whether the player wants to play the given
// action card (e.g., Vassal's revealed action).
type MayPlayActionPrompt struct {
	Card CardID
}

func (MayPlayActionPrompt) isPrompt() {}

// TrashFromRevealedPrompt asks the player to choose one card to trash
// from a pre-revealed list (e.g., Bandit's top-2 deck reveal).
type TrashFromRevealedPrompt struct {
	Cards []CardID
}

func (TrashFromRevealedPrompt) isPrompt() {}

// ChooseActionFromHandPrompt asks the player to choose one Action card
// from their hand to play (e.g., Throne Room's first decision). The
// engine guarantees ≥ 1 legal Action exists before requesting this; the
// answer is a CardChoiceAnswer with None=false.
type ChooseActionFromHandPrompt struct{}

func (ChooseActionFromHandPrompt) isPrompt() {}

// SetAsideActionPrompt asks the player to either set aside the just-
// drawn Action card (Yes) or keep it in hand (No). Used by Library.
type SetAsideActionPrompt struct {
	Card CardID
}

func (SetAsideActionPrompt) isPrompt() {}

// DiscardFromRevealedPrompt asks the player to choose 0..n cards to
// discard from a pre-revealed list (e.g., Sentry's second decision).
type DiscardFromRevealedPrompt struct {
	Cards []CardID
}

func (DiscardFromRevealedPrompt) isPrompt() {}

// ReorderCardsPrompt asks the player to specify the new order of a
// pre-revealed list (e.g., Sentry's third decision). The answer is a
// CardListAnswer whose Cards slice is interpreted as the desired order
// from bottom-to-top in the player's deck — same convention the engine
// already uses everywhere else (slice end = top of deck).
type ReorderCardsPrompt struct {
	Cards []CardID
}

func (ReorderCardsPrompt) isPrompt() {}

// --- Answer types ---

// CardListAnswer holds zero or more cards chosen by the player.
type CardListAnswer struct {
	Cards []CardID
}

func (CardListAnswer) isAnswer() {}

// CardChoiceAnswer holds a single card choice, or None to decline.
type CardChoiceAnswer struct {
	Card CardID
	None bool
}

func (CardChoiceAnswer) isAnswer() {}

// YesNoAnswer holds a boolean response.
type YesNoAnswer struct {
	Yes bool
}

func (YesNoAnswer) isAnswer() {}

// RequestDecision sets a pending decision on the game state. The engine
// will reject all non-ResolveDecision actions until this is resolved.
func RequestDecision(gs *GameState, px PlayerIdx, cardID CardID, step int, prompt Prompt, ctx map[ContextKey]any) []Event {
	gs.DecisionSeq++
	gs.PendingDecision = &Decision{
		ID:        fmt.Sprintf("d%d", gs.DecisionSeq),
		PlayerIdx: px,
		CardID:    cardID,
		Step:      step,
		Prompt:    prompt,
		Context:   ctx,
	}
	return []Event{{Kind: EventDecisionRequested, PlayerIdx: px}}
}
