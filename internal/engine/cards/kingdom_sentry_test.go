package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestSentry_Metadata(t *testing.T) {
	require.Equal(t, engine.CardID("sentry"), Sentry.ID)
	require.Equal(t, "Sentry", Sentry.Name)
	require.Equal(t, 5, Sentry.Cost)
	require.True(t, Sentry.HasType(engine.TypeAction))
}

func TestSentry_OnPlay_DrawsAndRevealsTwo(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{}
	gs.Players[0].Deck = []engine.CardID{
		"estate", // bottom — won't be revealed.
		"silver",
		"estate", // top-2 of deck (slice end is top): silver and estate.
		"silver",
	}
	gs.Players[0].InPlay = []engine.CardID{"sentry"}

	Sentry.OnPlay(gs, 0)
	// +1 card → Hand has 1.
	require.Len(t, gs.Players[0].Hand, 1)
	// 2 cards revealed into SetAside.
	require.Len(t, gs.Players[0].SetAside, 2,
		"Sentry reveals top 2 cards into SetAside")
	// Trash prompt parked.
	require.NotNil(t, gs.PendingDecision)
	p, ok := gs.PendingDecision.Prompt.(engine.TrashFromRevealedPrompt)
	require.True(t, ok)
	require.Equal(t, gs.Players[0].SetAside, p.Cards)
}

func TestSentry_OnPlay_AddsOneAction(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{}
	gs.Players[0].Deck = []engine.CardID{"copper", "silver", "estate"}
	gs.Players[0].InPlay = []engine.CardID{"sentry"}
	gs.Players[0].Actions = 0

	Sentry.OnPlay(gs, 0)
	require.Equal(t, 1, gs.Players[0].Actions, "+1 Action")
}

func TestSentry_OnPlay_EmptyDeckAndDiscard_NoDecision(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{}
	gs.Players[0].Deck = nil
	gs.Players[0].Discard = nil
	gs.Players[0].InPlay = []engine.CardID{"sentry"}

	Sentry.OnPlay(gs, 0)
	require.Nil(t, gs.PendingDecision, "nothing to reveal, no prompt")
	require.Empty(t, gs.Players[0].SetAside)
}

func TestSentry_Step0_TrashAll_Terminates(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{}
	gs.Players[0].Deck = []engine.CardID{"silver", "estate", "curse", "estate"}
	gs.Players[0].InPlay = []engine.CardID{"sentry"}

	Sentry.OnPlay(gs, 0)
	require.NotNil(t, gs.PendingDecision)

	revealed := append([]engine.CardID(nil), gs.Players[0].SetAside...)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: revealed},
	}, registryLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision, "trash all → no further prompts")
	require.Empty(t, gs.Players[0].SetAside)
	for _, c := range revealed {
		require.Contains(t, gs.Trash, c)
	}
}

func TestSentry_Step0_TrashNone_AdvancesToDiscard(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{}
	gs.Players[0].Deck = []engine.CardID{"silver", "silver", "silver", "silver"}
	gs.Players[0].InPlay = []engine.CardID{"sentry"}

	Sentry.OnPlay(gs, 0)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: nil},
	}, registryLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)
	_, ok := gs.PendingDecision.Prompt.(engine.DiscardFromRevealedPrompt)
	require.True(t, ok)
}

func TestSentry_Step1_DiscardNone_AdvancesToReorder(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{}
	gs.Players[0].Deck = []engine.CardID{"silver", "silver", "silver", "silver"}
	gs.Players[0].InPlay = []engine.CardID{"sentry"}

	Sentry.OnPlay(gs, 0)
	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: nil},
	}, registryLookup)
	require.NoError(t, err)
	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: nil},
	}, registryLookup)
	require.NoError(t, err)

	require.NotNil(t, gs.PendingDecision)
	rp, ok := gs.PendingDecision.Prompt.(engine.ReorderCardsPrompt)
	require.True(t, ok)
	require.Len(t, rp.Cards, 2)
}

func TestSentry_Step1_OneLeft_PutsBackNoReorder(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{}
	gs.Players[0].Deck = []engine.CardID{"silver", "silver", "estate", "silver"}
	gs.Players[0].InPlay = []engine.CardID{"sentry"}

	Sentry.OnPlay(gs, 0)
	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: []engine.CardID{"estate"}},
	}, registryLookup)
	require.NoError(t, err)
	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: nil},
	}, registryLookup)
	require.NoError(t, err)

	require.Nil(t, gs.PendingDecision, "1 left → no reorder prompt")
	require.Empty(t, gs.Players[0].SetAside)
	require.Equal(t, engine.CardID("silver"),
		gs.Players[0].Deck[len(gs.Players[0].Deck)-1])
}

func TestSentry_Step2_Reorder_PutsBackInOrder(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{}
	gs.Players[0].Deck = []engine.CardID{"copper", "copper", "silver", "estate"}
	gs.Players[0].InPlay = []engine.CardID{"sentry"}

	Sentry.OnPlay(gs, 0)
	revealed := append([]engine.CardID(nil), gs.Players[0].SetAside...)
	_, _, _ = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{},
	}, registryLookup)
	_, _, _ = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{},
	}, registryLookup)
	require.NotNil(t, gs.PendingDecision)

	reversed := []engine.CardID{revealed[1], revealed[0]}
	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.CardListAnswer{Cards: reversed},
	}, registryLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Empty(t, gs.Players[0].SetAside)
	n := len(gs.Players[0].Deck)
	require.Equal(t, reversed[0], gs.Players[0].Deck[n-2])
	require.Equal(t, reversed[1], gs.Players[0].Deck[n-1])
}
