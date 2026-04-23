package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestDecision_SetsPendingDecision(t *testing.T) {
	gs := newTestState(2)
	prompt := DiscardFromHandPrompt{Min: 0, Max: 3}

	events := RequestDecision(gs, 0, "cellar", 0, prompt, nil)

	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, PlayerIdx(0), gs.PendingDecision.PlayerIdx)
	require.Equal(t, CardID("cellar"), gs.PendingDecision.CardID)
	require.Equal(t, 0, gs.PendingDecision.Step)
	require.Equal(t, prompt, gs.PendingDecision.Prompt)
	require.Len(t, events, 1)
	require.Equal(t, EventDecisionRequested, events[0].Kind)
}

func TestRequestDecision_DeterministicIDs(t *testing.T) {
	gs := newTestState(2)
	RequestDecision(gs, 0, "cellar", 0, DiscardFromHandPrompt{}, nil)
	id1 := gs.PendingDecision.ID
	gs.PendingDecision = nil

	RequestDecision(gs, 0, "chapel", 0, TrashFromHandPrompt{}, nil)
	id2 := gs.PendingDecision.ID

	require.NotEqual(t, id1, id2, "sequential decision IDs must differ")
}

func TestRequestDecision_WithContext(t *testing.T) {
	gs := newTestState(2)
	ctx := map[ContextKey]any{CtxKeyTrashedCost: 4}

	RequestDecision(gs, 0, "remodel", 1, GainFromSupplyPrompt{MaxCost: 6}, ctx)

	require.Equal(t, 4, gs.PendingDecision.Context[CtxKeyTrashedCost])
	require.Equal(t, 1, gs.PendingDecision.Step)
}

func TestPutOnDeckPrompt_EmptyFilter_MeansAnyCard(t *testing.T) {
	p := PutOnDeckPrompt{}
	require.Empty(t, p.TypeFilter, "default zero-value TypeFilter must be empty")
}

func TestPutOnDeckPrompt_WithTypeFilter(t *testing.T) {
	p := PutOnDeckPrompt{TypeFilter: []CardType{TypeVictory}}
	require.Equal(t, []CardType{TypeVictory}, p.TypeFilter)
}

func TestTrashFromRevealedPrompt_Shape(t *testing.T) {
	p := TrashFromRevealedPrompt{Cards: []CardID{"silver", "gold"}}
	require.Equal(t, []CardID{"silver", "gold"}, p.Cards)
	var _ Prompt = p // compile-time check that it implements Prompt
}

func TestContextKeys_NewInTier3(t *testing.T) {
	require.Equal(t, ContextKey("attacker"), CtxKeyAttacker)
	require.Equal(t, ContextKey("remaining_victims"), CtxKeyRemainingVictims)
	require.Equal(t, ContextKey("revealed_cards"), CtxKeyRevealedCards)
}
