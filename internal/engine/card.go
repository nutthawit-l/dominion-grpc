package engine

// Card is the unified definition of every card in the game. Treasures,
// victories, curses, kingdom cards all share this struct.
type Card struct {
	ID    CardID
	Name  string
	Cost  int
	Types []CardType

	// OnPlay runs when a player plays the card. Nil-safe.
	OnPlay func(s *GameState, playerIdx int) []Event

	// VictoryPoints is called at game end. Returns 0 for non-victory cards.
	VictoryPoints func(p PlayerState) int
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
