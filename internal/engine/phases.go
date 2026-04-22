package engine

// cleanupAndEndTurn performs the cleanup phase for the current player,
// then advances to the next player's Action phase. Cleanup:
//   - discards all cards in InPlay
//   - discards all cards in Hand
//   - draws 5 new cards
//
// Next player's resources are reset to 1 action, 1 buy, 0 coins.
func cleanupAndEndTurn(gs *GameState) []Event {
	p := gs.CurrentPlayer
	var events []Event

	// Discard in-play.
	for _, c := range gs.Players[p].InPlay {
		gs.Players[p].Discard = append(gs.Players[p].Discard, c)
		events = append(events, Event{Kind: EventCardDiscarded, PlayerIdx: p, CardID: c})
	}
	gs.Players[p].InPlay = nil

	// Discard hand.
	for _, c := range gs.Players[p].Hand {
		gs.Players[p].Discard = append(gs.Players[p].Discard, c)
		events = append(events, Event{Kind: EventCardDiscarded, PlayerIdx: p, CardID: c})
	}
	gs.Players[p].Hand = nil

	// Draw 5.
	events = append(events, DrawCards(gs, p, 5)...)

	// Reset resources — they only apply to the current player's turn
	// and are re-set below for the NEXT player.
	gs.Players[p].Actions = 0
	gs.Players[p].Buys = 0
	gs.Players[p].Coins = 0

	// Advance to next player.
	next := (p + 1) % len(gs.Players)
	gs.CurrentPlayer = next
	if next == gs.StartingPlayer {
		gs.Turn++
	}
	gs.Phase = PhaseAction
	gs.Players[next].Actions = 1
	gs.Players[next].Buys = 1
	gs.Players[next].Coins = 0

	events = append(events,
		Event{Kind: EventPhaseChanged, PlayerIdx: next, Phase: PhaseAction},
		Event{Kind: EventTurnStarted, PlayerIdx: next, Count: gs.Turn},
	)
	return events
}
