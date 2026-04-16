package engine

// cleanupAndEndTurn performs the cleanup phase for the current player,
// then advances to the next player's Action phase. Cleanup:
//   - discards all cards in InPlay
//   - discards all cards in Hand
//   - draws 5 new cards
//
// Next player's resources are reset to 1 action, 1 buy, 0 coins.
func cleanupAndEndTurn(s *GameState) []Event {
	p := s.CurrentPlayer
	var events []Event

	// Discard in-play.
	for _, c := range s.Players[p].InPlay {
		s.Players[p].Discard = append(s.Players[p].Discard, c)
		events = append(events, Event{Kind: EventCardDiscarded, PlayerIdx: p, CardID: c})
	}
	s.Players[p].InPlay = nil

	// Discard hand.
	for _, c := range s.Players[p].Hand {
		s.Players[p].Discard = append(s.Players[p].Discard, c)
		events = append(events, Event{Kind: EventCardDiscarded, PlayerIdx: p, CardID: c})
	}
	s.Players[p].Hand = nil

	// Draw 5.
	events = append(events, DrawCards(s, p, 5)...)

	// Reset resources — they only apply to the current player's turn
	// and are re-set below for the NEXT player.
	s.Players[p].Actions = 0
	s.Players[p].Buys = 0
	s.Players[p].Coins = 0

	// Advance to next player.
	next := (p + 1) % len(s.Players)
	s.CurrentPlayer = next
	if next == s.StartingPlayer {
		s.Turn++
	}
	s.Phase = PhaseAction
	s.Players[next].Actions = 1
	s.Players[next].Buys = 1
	s.Players[next].Coins = 0

	events = append(events,
		Event{Kind: EventPhaseChanged, PlayerIdx: next, Phase: PhaseAction},
		Event{Kind: EventTurnStarted, PlayerIdx: next, Count: s.Turn},
	)
	return events
}
