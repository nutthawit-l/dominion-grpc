package service

import (
	"testing"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestActionFromProto_PlayCard(t *testing.T) {
	pa := &pb.Action{Kind: &pb.Action_PlayCard{PlayCard: &pb.PlayCardAction{
		PlayerIdx: 0, CardId: "copper",
	}}}
	got, err := ActionFromProto(pa)
	require.NoError(t, err)
	require.Equal(t, engine.PlayCard{PlayerIdx: 0, Card: "copper"}, got)
}

func TestActionFromProto_BuyCard(t *testing.T) {
	pa := &pb.Action{Kind: &pb.Action_BuyCard{BuyCard: &pb.BuyCardAction{
		PlayerIdx: 1, CardId: "silver",
	}}}
	got, err := ActionFromProto(pa)
	require.NoError(t, err)
	require.Equal(t, engine.BuyCard{PlayerIdx: 1, Card: "silver"}, got)
}

func TestActionFromProto_EndPhase(t *testing.T) {
	pa := &pb.Action{Kind: &pb.Action_EndPhase{EndPhase: &pb.EndPhaseAction{
		PlayerIdx: 0,
	}}}
	got, err := ActionFromProto(pa)
	require.NoError(t, err)
	require.Equal(t, engine.EndPhase{PlayerIdx: 0}, got)
}

func TestSnapshotFromState_ScrubsOpponentHand(t *testing.T) {
	s := &engine.GameState{
		GameID:        "g",
		Seed:          42,
		Turn:          3,
		CurrentPlayer: 1,
		Phase:         engine.PhaseBuy,
		Players: []engine.PlayerState{
			{Name: "A", Hand: []engine.CardID{"copper", "estate"}},
			{Name: "B", Hand: []engine.CardID{"silver"}},
		},
		Supply: engine.Supply{Piles: map[engine.CardID]int{"copper": 10}},
	}
	snap := SnapshotFromState(s, 0)
	require.Equal(t, "g", snap.GameId)
	require.Equal(t, int32(3), snap.Turn)
	// viewer=0 sees their own hand contents...
	require.Equal(t, []string{"copper", "estate"}, snap.Players[0].Hand)
	// ...but opponent hand contents are scrubbed.
	require.Empty(t, snap.Players[1].Hand)
	require.Equal(t, int32(1), snap.Players[1].HandSize)
}

func TestPromptToProto_PutOnDeck_WithTypeFilter(t *testing.T) {
	d := &engine.Decision{
		ID:        "d1",
		PlayerIdx: 0,
		CardID:    "bureaucrat",
		Step:      0,
		Prompt:    engine.PutOnDeckPrompt{TypeFilter: []engine.CardType{engine.TypeVictory}},
	}
	pd := DecisionToProto(d)
	require.NotNil(t, pd)
	put := pd.GetPutOnDeck()
	require.NotNil(t, put)
	require.Equal(t, []pb.CardType{pb.CardType_CARD_TYPE_VICTORY}, put.TypeFilter)
}

func TestPromptToProto_PutOnDeck_EmptyFilter(t *testing.T) {
	d := &engine.Decision{
		ID:        "d1",
		PlayerIdx: 0,
		CardID:    "artisan",
		Step:      1,
		Prompt:    engine.PutOnDeckPrompt{},
	}
	pd := DecisionToProto(d)
	require.NotNil(t, pd)
	put := pd.GetPutOnDeck()
	require.NotNil(t, put)
	require.Empty(t, put.TypeFilter)
}

func TestPromptToProto_TrashFromRevealed(t *testing.T) {
	d := &engine.Decision{
		ID:        "d1",
		PlayerIdx: 1,
		CardID:    "bandit",
		Step:      0,
		Prompt:    engine.TrashFromRevealedPrompt{Cards: []engine.CardID{"silver", "gold"}},
	}
	pd := DecisionToProto(d)
	require.NotNil(t, pd)
	tfr := pd.GetTrashFromRevealed()
	require.NotNil(t, tfr)
	require.Equal(t, []string{"silver", "gold"}, tfr.Cards)
}
