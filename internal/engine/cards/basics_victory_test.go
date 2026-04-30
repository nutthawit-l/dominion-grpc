package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestEstate_VictoryPoints(t *testing.T) {
	require.Equal(t, 1, Estate.VictoryPoints(engine.PlayerState{}))
	require.Equal(t, 2, Estate.Cost)
	require.True(t, Estate.HasType(engine.TypeVictory))
}

func TestDuchy_VictoryPoints(t *testing.T) {
	require.Equal(t, 3, Duchy.VictoryPoints(engine.PlayerState{}))
	require.Equal(t, 5, Duchy.Cost)
}

func TestProvince_VictoryPoints(t *testing.T) {
	require.Equal(t, 6, Province.VictoryPoints(engine.PlayerState{}))
	require.Equal(t, 8, Province.Cost)
}

func TestCurse_NegativeVP(t *testing.T) {
	require.Equal(t, -1, Curse.VictoryPoints(engine.PlayerState{}))
	require.Equal(t, 0, Curse.Cost)
	require.True(t, Curse.HasType(engine.TypeCurse))
}

func TestVictories_HaveNoOnPlay(t *testing.T) {
	require.Nil(t, Estate.OnPlay)
	require.Nil(t, Duchy.OnPlay)
	require.Nil(t, Province.OnPlay)
	require.Nil(t, Curse.OnPlay)
}
