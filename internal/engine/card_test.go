package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCard_OnReaction_NilByDefault(t *testing.T) {
	c := &Card{ID: "nothing_special", Name: "X", Cost: 0}
	require.Nil(t, c.OnReaction, "cards without a reaction must leave OnReaction nil")
}

func TestTrigger_AttackPlayedKind(t *testing.T) {
	tr := Trigger{Kind: TriggerAttackPlayed, Attacker: PlayerIdx(0), CardID: "witch"}
	require.Equal(t, TriggerAttackPlayed, tr.Kind)
	require.Equal(t, PlayerIdx(0), tr.Attacker)
	require.Equal(t, CardID("witch"), tr.CardID)
}

func TestCard_OnReaction_CallableReturnsBlocksAndEvents(t *testing.T) {
	called := false
	c := &Card{
		ID: "fake_reaction", Name: "X", Cost: 2,
		OnReaction: func(gs *GameState, victim PlayerIdx, trigger Trigger) (bool, []Event) {
			called = true
			return true, []Event{{Kind: EventReactionTriggered, PlayerIdx: victim, CardID: "fake_reaction"}}
		},
	}
	gs := NewTestStateWithRNG("t", 1, []PlayerState{{Name: "a"}})
	blocks, ev := c.OnReaction(gs, 0, Trigger{Kind: TriggerAttackPlayed, Attacker: 0, CardID: "witch"})
	require.True(t, called)
	require.True(t, blocks)
	require.Len(t, ev, 1)
	require.Equal(t, EventReactionTriggered, ev[0].Kind)
}
