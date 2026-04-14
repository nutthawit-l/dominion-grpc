package bot

import (
	"context"
	"errors"
)

// Run drives a strategy against the given game via c until the game
// ends or ctx cancels. On sequence-gap errors it closes the stream and
// resubscribes; all other errors terminate the loop.
func Run(ctx context.Context, c *Client, gameID string, me int, strat Strategy) error {
	cs := &ClientState{Me: me}

	for {
		stream, err := c.StreamGameEvents(ctx, gameID, me)
		if err != nil {
			return err
		}
		loopErr := driveOnce(ctx, c, gameID, stream, cs, strat)
		_ = stream.Close()
		if loopErr == nil {
			return nil
		}
		if !errors.Is(loopErr, ErrSequenceGap) {
			return loopErr
		}
		// reset sequence tracking so the next stream's initial snapshot
		// (sequence 0) is accepted
		cs.LastSeq = 0
	}
}

// driveOnce drains a single subscription. Returns nil on normal end,
// ErrSequenceGap to trigger reconnection, or any other error to abort.
func driveOnce(ctx context.Context, c *Client, gameID string, stream *Stream, cs *ClientState, strat Strategy) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ev, ok := stream.Receive()
		if !ok {
			return stream.Err()
		}
		if err := cs.Apply(ev); err != nil {
			return err
		}
		if cs.Ended {
			return nil
		}
		switch {
		case cs.PendingDecision != nil && cs.DecidingPlayer == cs.Me:
			resp := strat.Resolve(cs, cs.PendingDecision)
			if err := c.submitResolve(ctx, gameID, resp); err != nil {
				return err
			}
		case cs.IsMyTurn() && cs.PendingDecision == nil:
			act := strat.PickAction(cs)
			if act == nil {
				act = endPhase(cs.Me)
			}
			if err := c.SubmitAction(ctx, gameID, act); err != nil {
				return err
			}
		}
	}
}
