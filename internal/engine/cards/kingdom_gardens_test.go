package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestGardens_Metadata(t *testing.T) {
	require.Equal(t, engine.CardID("gardens"), Gardens.ID)
	require.Equal(t, "Gardens", Gardens.Name)
	require.Equal(t, 4, Gardens.Cost)
	require.True(t, Gardens.HasType(engine.TypeVictory))
	require.False(t, Gardens.HasType(engine.TypeAction),
		"Gardens is Victory-only, not Action")
	require.True(t, Gardens.IsKingdom(),
		"Gardens must register as kingdom (Task 1's IsKingdom widening)")
	_, ok := DefaultRegistry.Lookup("gardens")
	require.True(t, ok, "Gardens must be registered in DefaultRegistry")
}

func TestGardens_VictoryPoints_FewerThanTen_IsZero(t *testing.T) {
	cases := []int{0, 1, 5, 9}
	for _, n := range cases {
		ps := engine.PlayerState{Hand: makeCards("copper", n)}
		require.Equalf(t, 0, Gardens.VictoryPoints(ps),
			"%d cards → expected 0 VP", n)
	}
}

func TestGardens_VictoryPoints_TenCards_IsOne(t *testing.T) {
	ps := engine.PlayerState{Hand: makeCards("copper", 10)}
	require.Equal(t, 1, Gardens.VictoryPoints(ps))
}

func TestGardens_VictoryPoints_RoundsDown(t *testing.T) {
	ps := engine.PlayerState{Hand: makeCards("copper", 19)}
	require.Equal(t, 1, Gardens.VictoryPoints(ps),
		"19 cards → 1 VP (round down)")
	ps = engine.PlayerState{Hand: makeCards("copper", 20)}
	require.Equal(t, 2, Gardens.VictoryPoints(ps))
}

func TestGardens_VictoryPoints_CountsAllZones(t *testing.T) {
	// 2 cards in each of the 5 zones = 10 total → 1 VP.
	ps := engine.PlayerState{
		Hand:     makeCards("copper", 2),
		Deck:     makeCards("copper", 2),
		Discard:  makeCards("copper", 2),
		InPlay:   makeCards("copper", 2),
		SetAside: makeCards("copper", 2),
	}
	require.Equal(t, 1, Gardens.VictoryPoints(ps),
		"Gardens must count Hand+Deck+Discard+InPlay+SetAside")
}

func TestGardens_HasNoOnPlay(t *testing.T) {
	require.Nil(t, Gardens.OnPlay,
		"Gardens is Victory-only and has no OnPlay")
}

// makeCards returns a slice of n copies of the given card ID.
func makeCards(id engine.CardID, n int) []engine.CardID {
	out := make([]engine.CardID, n)
	for i := range out {
		out[i] = id
	}
	return out
}
