package engine

// DrawCards draws up to n cards from player p's deck into their hand.
// If the deck runs out, the discard pile is shuffled and becomes the
// new deck. If both are empty, the draw stops short.
func DrawCards(gs *GameState, p int, n int) []Event {
	events := make([]Event, 0, n)
	ps := &gs.Players[p]
	for i := 0; i < n; i++ {
		if len(ps.Deck) == 0 {
			if len(ps.Discard) == 0 {
				return events
			}
			ps.Deck = ps.Discard
			ps.Discard = nil
			shuffleCards(gs.rng, ps.Deck)
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
func DiscardFromHand(gs *GameState, p int, cards []CardID) []Event {
	ps := &gs.Players[p]
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

// TrashFromHand moves the named cards from hand to the trash pile.
// Cards not present in hand are silently skipped.
func TrashFromHand(gs *GameState, p int, cards []CardID) []Event {
	ps := &gs.Players[p]
	var events []Event
	for _, c := range cards {
		idx := indexOf(ps.Hand, c)
		if idx < 0 {
			continue
		}
		ps.Hand = append(ps.Hand[:idx], ps.Hand[idx+1:]...)
		gs.Trash = append(gs.Trash, c)
		events = append(events, Event{Kind: EventCardTrashed, PlayerIdx: p, CardID: c})
	}
	return events
}

// PutOnDeck moves the named cards from hand to the top of the player's
// deck. Cards not present in hand are silently skipped.
func PutOnDeck(gs *GameState, p int, cards []CardID) []Event {
	ps := &gs.Players[p]
	var events []Event
	for _, c := range cards {
		idx := indexOf(ps.Hand, c)
		if idx < 0 {
			continue
		}
		ps.Hand = append(ps.Hand[:idx], ps.Hand[idx+1:]...)
		ps.Deck = append(ps.Deck, c)
		events = append(events, Event{Kind: EventCardPutOnDeck, PlayerIdx: p, CardID: c})
	}
	return events
}

// RevealAndDiscardFromDeck reveals the top n cards of the player's deck
// and moves them to discard. If the deck is empty, the discard is
// shuffled into the deck first. Returns the revealed card IDs.
func RevealAndDiscardFromDeck(gs *GameState, p int, n int) ([]CardID, []Event) {
	ps := &gs.Players[p]
	var revealed []CardID
	var events []Event
	for i := 0; i < n; i++ {
		if len(ps.Deck) == 0 {
			if len(ps.Discard) == 0 {
				return revealed, events
			}
			ps.Deck = ps.Discard
			ps.Discard = nil
			shuffleCards(gs.rng, ps.Deck)
		}
		top := len(ps.Deck) - 1
		card := ps.Deck[top]
		ps.Deck = ps.Deck[:top]
		ps.Discard = append(ps.Discard, card)
		revealed = append(revealed, card)
		events = append(events, Event{Kind: EventCardDiscarded, PlayerIdx: p, CardID: card})
	}
	return revealed, events
}

// PlayCardFromZone moves a card from the specified zone to in-play and
// calls its OnPlay. Does NOT consume an Action — the caller decides
// whether to decrement actions.
func PlayCardFromZone(gs *GameState, p int, cardID CardID, from Zone, lookup CardLookup) ([]Event, error) {
	card, ok := lookup(cardID)
	if !ok {
		return nil, ErrUnknownCard
	}
	ps := &gs.Players[p]
	switch from {
	case ZoneHand:
		idx := indexOf(ps.Hand, cardID)
		if idx < 0 {
			return nil, ErrCardNotInHand
		}
		ps.Hand = append(ps.Hand[:idx], ps.Hand[idx+1:]...)
	case ZoneDiscard:
		idx := indexOf(ps.Discard, cardID)
		if idx < 0 {
			return nil, ErrCardNotInDiscard
		}
		ps.Discard = append(ps.Discard[:idx], ps.Discard[idx+1:]...)
	}
	ps.InPlay = append(ps.InPlay, cardID)
	events := []Event{{Kind: EventCardPlayed, PlayerIdx: p, CardID: cardID}}
	if card.OnPlay != nil {
		events = append(events, card.OnPlay(gs, p)...)
	}
	return events, nil
}

// IndexOf returns the index of the first occurrence of c in cards, or -1.
// Exported for use by card implementations in the cards package.
func IndexOf(cards []CardID, c CardID) int {
	return indexOf(cards, c)
}

// AddCoins adds n to the player's Coins total and emits one event.
func AddCoins(gs *GameState, p int, n int) []Event {
	gs.Players[p].Coins += n
	return []Event{{Kind: EventCoinsAdded, PlayerIdx: p, Count: n}}
}

// AddBuys adds n to the player's Buys total and emits one event.
func AddBuys(gs *GameState, p int, n int) []Event {
	gs.Players[p].Buys += n
	return []Event{{Kind: EventBuysAdded, PlayerIdx: p, Count: n}}
}

// AddActions adds n to the player's Actions total and emits one event.
func AddActions(gs *GameState, p int, n int) []Event {
	gs.Players[p].Actions += n
	return []Event{{Kind: EventActionsAdded, PlayerIdx: p, Count: n}}
}

// EachOtherPlayer calls fn(idx) for every player except `except`, in
// turn order starting at the next seat, wrapping around. Returns the
// concatenation of all events fn returns.
func EachOtherPlayer(gs *GameState, except int, fn func(idx int) []Event) []Event {
	n := len(gs.Players)
	var events []Event
	for step := 1; step < n; step++ {
		idx := (except + step) % n
		events = append(events, fn(idx)...)
	}
	return events
}

// GainCard gains one copy of card from the supply to the given destination
// for player p. If the supply pile is empty, nothing happens and no event
// is emitted.
func GainCard(gs *GameState, p int, card CardID, dest GainDest) []Event {
	if gs.Supply.Piles[card] <= 0 {
		return nil
	}
	gs.Supply.Piles[card]--
	ps := &gs.Players[p]
	switch dest {
	case GainToHand:
		ps.Hand = append(ps.Hand, card)
	case GainToDeck:
		ps.Deck = append(ps.Deck, card)
	default:
		ps.Discard = append(ps.Discard, card)
	}
	return []Event{{Kind: EventCardGained, PlayerIdx: p, CardID: card}}
}
