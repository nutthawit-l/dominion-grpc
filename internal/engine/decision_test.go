package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestDecision_SetsPendingDecision(t *testing.T) {
	s := newTestState(2)
	prompt := DiscardFromHandPrompt{Min: 0, Max: 3}

	events := RequestDecision(s, 0, "cellar", 0, prompt, nil)

	require.NotNil(t, s.PendingDecision)
	require.Equal(t, 0, s.PendingDecision.PlayerIdx)
	require.Equal(t, CardID("cellar"), s.PendingDecision.CardID)
	require.Equal(t, 0, s.PendingDecision.Step)
	require.Equal(t, prompt, s.PendingDecision.Prompt)
	require.Len(t, events, 1)
	require.Equal(t, EventDecisionRequested, events[0].Kind)
}

func TestRequestDecision_DeterministicIDs(t *testing.T) {
	s := newTestState(2)
	RequestDecision(s, 0, "cellar", 0, DiscardFromHandPrompt{}, nil)
	id1 := s.PendingDecision.ID
	s.PendingDecision = nil

	RequestDecision(s, 0, "chapel", 0, TrashFromHandPrompt{}, nil)
	id2 := s.PendingDecision.ID

	require.NotEqual(t, id1, id2, "sequential decision IDs must differ")
}

func TestRequestDecision_WithContext(t *testing.T) {
	s := newTestState(2)
	ctx := map[string]any{"trashed_cost": 4}

	RequestDecision(s, 0, "remodel", 1, GainFromSupplyPrompt{MaxCost: 6}, ctx)

	require.Equal(t, 4, s.PendingDecision.Context["trashed_cost"])
	require.Equal(t, 1, s.PendingDecision.Step)
}
