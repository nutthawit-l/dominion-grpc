package engine

// PlayCardInPlace runs a card's OnPlay without moving it between zones.
// The caller is responsible for already having the card in InPlay (or
// arranged equivalent state). Used by:
//   - Throne Room's OnResolve, for the first play of the doubled card.
//   - Apply's pending-play unwinder, for queued replays.
//
// Distinct from PlayCardFromZone (Vassal's helper), which moves the
// card from Hand/Discard → InPlay first. This helper does NOT consume
// an Action — Throne Room paid an Action to play itself, and the
// doubled card's two plays are free.
func PlayCardInPlace(gs *GameState, px PlayerIdx, cardID CardID, lookup CardLookup) ([]Event, error) {
	card, ok := lookup(cardID)
	if !ok {
		return nil, ErrUnknownCard
	}
	events := []Event{{Kind: EventCardPlayed, PlayerIdx: px, CardID: cardID}}
	if card.OnPlay != nil {
		events = append(events, card.OnPlay(gs, px)...)
	}
	return events, nil
}
