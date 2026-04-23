package engine

// ResolveAttackVictims iterates opponents in turn order (next seat from
// attacker, wrapping), scans each victim's hand for cards with a
// non-nil OnReaction, calls each such reaction, and collects the set
// of victims that were NOT blocked.
//
// Returns:
//   - victims: seats that did not block, in turn order
//   - events: one EventAttackPlayed followed by one event per reaction
//     event returned by any card's OnReaction
//
// Does NOT create decisions. Callers (attack cards) decide per-victim
// whether a prompt is needed and stash the remaining-victims queue across
// OnResolve cycles.
func ResolveAttackVictims(gs *GameState, attacker PlayerIdx, cardID CardID, lookup CardLookup) ([]PlayerIdx, []Event) {
	events := []Event{{Kind: EventAttackPlayed, PlayerIdx: attacker, CardID: cardID}}
	trigger := Trigger{Kind: TriggerAttackPlayed, Attacker: attacker, CardID: cardID}

	n := PlayerIdx(len(gs.Players))
	var victims []PlayerIdx
	for step := PlayerIdx(1); step < n; step++ {
		victim := (attacker + step) % n
		blocked := false
		for _, handCardID := range gs.Players[victim].Hand {
			card, ok := lookup(handCardID)
			if !ok || card.OnReaction == nil {
				continue
			}
			blocks, ev := card.OnReaction(gs, victim, trigger)
			events = append(events, ev...)
			if blocks {
				blocked = true
			}
		}
		if !blocked {
			victims = append(victims, victim)
		}
	}
	return victims, events
}
