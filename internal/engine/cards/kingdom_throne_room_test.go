package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestThroneRoom_Metadata(t *testing.T) {
	require.Equal(t, engine.CardID("throne_room"), ThroneRoom.ID)
	require.Equal(t, "Throne Room", ThroneRoom.Name)
	require.Equal(t, 4, ThroneRoom.Cost)
	require.True(t, ThroneRoom.HasType(engine.TypeAction))
}

func TestThroneRoom_OnPlay_NoActionInHand_NoDecision(t *testing.T) {
	gs := newTestStateForCards(2)
	gs.Players[0].Hand = []engine.CardID{"copper", "estate"} // no actions
	gs.Players[0].InPlay = []engine.CardID{"throne_room"}

	events := ThroneRoom.OnPlay(gs, 0)
	require.Nil(t, gs.PendingDecision, "no Action → no decision")
	require.Empty(t, gs.PendingPlays, "no Action → no queued replay")
	_ = events
}

func TestThroneRoom_OnPlay_OneAction_RequestsChoice(t *testing.T) {
	gs := newTestStateForCards(2)
	gs.Players[0].Hand = []engine.CardID{"copper", "smithy"}
	gs.Players[0].InPlay = []engine.CardID{"throne_room"}

	ThroneRoom.OnPlay(gs, 0)
	require.NotNil(t, gs.PendingDecision)
	_, ok := gs.PendingDecision.Prompt.(engine.ChooseActionFromHandPrompt)
	require.True(t, ok, "prompt must be ChooseActionFromHandPrompt")
}

func TestThroneRoom_OnResolve_PlaysSmithyTwice(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{"smithy"}
	gs.Players[0].Deck = []engine.CardID{
		"copper", "copper", "copper", "copper", "copper", "copper",
	}
	gs.Players[0].InPlay = []engine.CardID{"throne_room"}

	// Trigger the prompt.
	ThroneRoom.OnPlay(gs, 0)
	require.NotNil(t, gs.PendingDecision)

	// Resolve it via Apply so the unwinder runs.
	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  0,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardChoiceAnswer{Card: "smithy"},
	}, registryLookup)
	require.NoError(t, err)

	// Smithy draws 3 + 3 = 6 coppers from deck.
	require.Len(t, gs.Players[0].Hand, 6, "Smithy played twice draws 6")
	require.Empty(t, gs.PendingPlays, "stack drained after second play")
	require.Nil(t, gs.PendingDecision)
	// Smithy lives in InPlay (next to Throne Room).
	require.Contains(t, gs.Players[0].InPlay, engine.CardID("smithy"))
	require.Contains(t, gs.Players[0].InPlay, engine.CardID("throne_room"))
}

func TestThroneRoom_OnResolve_FirstPlaySetsDecision_SecondQueued(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	// Cellar prompts to discard, then draws — first play sets a decision.
	gs.Players[0].Hand = []engine.CardID{"cellar", "copper", "estate"}
	gs.Players[0].Deck = []engine.CardID{"silver", "silver", "silver", "silver"}
	gs.Players[0].InPlay = []engine.CardID{"throne_room"}
	gs.Players[0].Actions = 1

	ThroneRoom.OnPlay(gs, 0)
	require.NotNil(t, gs.PendingDecision)

	// Resolve TR's choice → Cellar.
	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  0,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardChoiceAnswer{Card: "cellar"},
	}, registryLookup)
	require.NoError(t, err)

	// Cellar's first play parked a discard prompt; second Cellar play
	// is on the stack.
	require.NotNil(t, gs.PendingDecision, "Cellar's prompt should be parked")
	require.Len(t, gs.PendingPlays, 1, "second Cellar play queued")
	require.Equal(t, engine.CardID("cellar"), gs.PendingPlays[0].CardID)
}

func TestThroneRoom_RecursiveTR_TR_Witch_Gives2Curses(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 7)
	// Player 0 plays the outer TR; their hand contains another TR plus
	// a Witch they'll throne with the inner TR.
	gs.Players[0].Hand = []engine.CardID{"throne_room", "witch", "copper", "copper"}
	gs.Players[0].Deck = []engine.CardID{"silver", "silver", "silver", "silver", "silver", "silver"}
	gs.Players[0].InPlay = []engine.CardID{"throne_room"} // outer TR
	gs.Players[1].Deck = []engine.CardID{"copper", "copper", "copper", "copper", "copper"}
	gs.Supply.Piles["curse"] = 10

	ThroneRoom.OnPlay(gs, 0)
	require.NotNil(t, gs.PendingDecision)

	// Outer TR → choose inner TR.
	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  0,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardChoiceAnswer{Card: "throne_room"},
	}, registryLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision, "inner TR must prompt for an Action")

	// Inner TR → choose Witch.
	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  0,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardChoiceAnswer{Card: "witch"},
	}, registryLookup)
	require.NoError(t, err)

	// With only 1 Witch, inner TR's second play has no Action available → no-op.
	// Net: 2 Witch plays → 2 Curses.
	require.Equal(t, 8, gs.Supply.Piles["curse"], "2 Witch plays gave 2 Curses")
}

func TestThroneRoom_RecursiveTR_TR_TwoWitches_Gives4Curses(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 7)
	gs.Players[0].Hand = []engine.CardID{"throne_room", "witch", "witch", "copper"}
	gs.Players[0].Deck = []engine.CardID{"silver", "silver", "silver", "silver", "silver", "silver", "silver", "silver"}
	gs.Players[0].InPlay = []engine.CardID{"throne_room"}
	gs.Players[1].Deck = []engine.CardID{"copper", "copper", "copper", "copper", "copper"}
	gs.Supply.Piles["curse"] = 10

	ThroneRoom.OnPlay(gs, 0)

	// Outer TR → inner TR.
	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  0,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardChoiceAnswer{Card: "throne_room"},
	}, registryLookup)
	require.NoError(t, err)

	// Inner TR (first play) → Witch #1.
	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  0,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardChoiceAnswer{Card: "witch"},
	}, registryLookup)
	require.NoError(t, err)

	// After unwinding, inner-TR (second play) must prompt again for an
	// Action choice from hand. Witch #2 should be there.
	require.NotNil(t, gs.PendingDecision,
		"inner TR's second play must prompt for another Action")

	// Inner TR (second play) → Witch #2.
	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  0,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardChoiceAnswer{Card: "witch"},
	}, registryLookup)
	require.NoError(t, err)

	// Net plays: Witch #1 twice + Witch #2 twice = 4 Witch plays = 4 Curses.
	require.Equal(t, 6, gs.Supply.Piles["curse"], "4 Curses to opponent")
	require.Empty(t, gs.PendingPlays, "stack fully drained")
	require.Nil(t, gs.PendingDecision, "no decision pending at end")
}
