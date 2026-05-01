package engine

// Card is the unified definition of every card in the game. Treasures,
// victories, curses, kingdom cards all share this struct.
type Card struct {
	ID    CardID
	Name  string
	Cost  int
	Types []CardType

	// OnPlay runs when a player plays the card. Nil-safe.
	OnPlay func(gs *GameState, px PlayerIdx) []Event

	// VictoryPoints is called at game end. Returns 0 for non-victory cards.
	VictoryPoints func(p PlayerState) int

	// OnResolve is called when ResolveDecision arrives for a decision
	// created by this card. Nil for cards without decisions.
	OnResolve func(gs *GameState, px PlayerIdx, d *Decision, answer Answer, lookup CardLookup) ([]Event, error)

	// OnReaction is called when an event fires that a reaction card can
	// respond to. Returns (blocks, events): blocks=true means the reaction
	// intercepted the triggering effect (e.g., Moat blocks an attack).
	// Nil for cards without reaction behavior.
	OnReaction func(gs *GameState, victim PlayerIdx, trigger Trigger) (blocks bool, events []Event)
}

// HasType reports whether c is tagged with t.
func (c *Card) HasType(t CardType) bool {
	for _, x := range c.Types {
		if x == t {
			return true
		}
	}
	return false
}

// IsKingdom reports whether c is a kingdom card. A kingdom card is any
// card tagged TypeAction (plain actions, attacks, reactions), OR any
// non-basic Victory card (Gardens-shape). Estate / Duchy / Province
// are basic Victory and therefore NOT kingdom; treasures and curses
// are likewise basic.
func (c *Card) IsKingdom() bool {
	if c.HasType(TypeAction) {
		return true
	}
	if c.HasType(TypeVictory) {
		switch c.ID {
		case "estate", "duchy", "province":
			return false
		}
		return true
	}
	return false
}
