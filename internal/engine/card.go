package engine

// Card is the unified definition of every card in the game. Treasures,
// victories, curses, kingdom cards all share this struct.
type Card struct {
	ID    CardID
	Name  string
	Cost  int
	Types []CardType

	// OnPlay runs when a player plays the card. Nil-safe.
	OnPlay func(s *GameState, p int) []Event

	// VictoryPoints is called at game end. Returns 0 for non-victory cards.
	VictoryPoints func(p PlayerState) int

	// OnResolve is called when ResolveDecision arrives for a decision
	// created by this card. Nil for cards without decisions.
	OnResolve func(s *GameState, p int, d *Decision, answer Answer, lookup CardLookup) ([]Event, error)
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
// card tagged TypeAction (plain actions, attacks, reactions). Basics
// (treasures, victories, curses) are not kingdom cards.
func (c *Card) IsKingdom() bool {
	return c.HasType(TypeAction)
}
