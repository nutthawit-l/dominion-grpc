package engine

// DrawCards draws up to n cards from player p's deck into their hand.
// If the deck runs out, the discard pile is shuffled and becomes the
// new deck. If both are empty, the draw stops short.
func DrawCards(s *GameState, p int, n int) []Event {
	events := make([]Event, 0, n)
	ps := &s.Players[p]
	for i := 0; i < n; i++ {
		if len(ps.Deck) == 0 {
			if len(ps.Discard) == 0 {
				return events
			}
			ps.Deck = ps.Discard
			ps.Discard = nil
			shuffleCards(s.rng, ps.Deck)
		}
		top := len(ps.Deck) - 1
		card := ps.Deck[top]
		ps.Deck = ps.Deck[:top]
		ps.Hand = append(ps.Hand, card)
		events = append(events, Event{Kind: EventCardDrawn, PlayerIdx: p, CardID: card})
	}
	return events
}

// DiscardFromHand moves the named cards from hand to discard. Cards
// not present in hand are silently skipped; the returned events reflect
// only cards that were actually moved.
func DiscardFromHand(s *GameState, p int, cards []CardID) []Event {
	ps := &s.Players[p]
	var events []Event
	for _, c := range cards {
		idx := indexOf(ps.Hand, c)
		if idx < 0 {
			continue
		}
		ps.Hand = append(ps.Hand[:idx], ps.Hand[idx+1:]...)
		ps.Discard = append(ps.Discard, c)
		events = append(events, Event{Kind: EventCardDiscarded, PlayerIdx: p, CardID: c})
	}
	return events
}

// indexOf returns the index of the first occurrence of c in cards, or
// -1 if not present.
func indexOf(cards []CardID, c CardID) int {
	for i, x := range cards {
		if x == c {
			return i
		}
	}
	return -1
}

// AddCoins adds n to the player's Coins total and emits one event.
func AddCoins(s *GameState, p int, n int) []Event {
	s.Players[p].Coins += n
	return []Event{{Kind: EventCoinsAdded, PlayerIdx: p, Count: n}}
}

// AddBuys adds n to the player's Buys total and emits one event.
func AddBuys(s *GameState, p int, n int) []Event {
	s.Players[p].Buys += n
	return []Event{{Kind: EventBuysAdded, PlayerIdx: p, Count: n}}
}

// AddActions adds n to the player's Actions total and emits one event.
func AddActions(s *GameState, p int, n int) []Event {
	s.Players[p].Actions += n
	return []Event{{Kind: EventActionsAdded, PlayerIdx: p, Count: n}}
}
