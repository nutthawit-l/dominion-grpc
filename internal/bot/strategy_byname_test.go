package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStrategyByName_AllRegisteredNames(t *testing.T) {
	names := []string{
		"bigmoney", "smithy_bm", "chapel_bm", "remodel_bm",
		"witch_bm", "militia_bm", "throneroom_bm", "library_bm",
		"sentry_bm", "merchant_bm", "gardens_bm",
	}
	for _, name := range names {
		s, err := StrategyByName(name)
		require.NoErrorf(t, err, "StrategyByName(%q)", name)
		require.NotNilf(t, s, "StrategyByName(%q) returned nil strategy", name)
		require.Equalf(t, name, s.Name(),
			"round-trip mismatch for %q", name)
	}
}

func TestStrategyByName_UnknownReturnsError(t *testing.T) {
	_, err := StrategyByName("does_not_exist")
	require.Error(t, err)
}

func TestIsVictory_IncludesGardens(t *testing.T) {
	require.True(t, isVictory("gardens"))
	require.True(t, isVictory("estate"))
	require.True(t, isVictory("duchy"))
	require.True(t, isVictory("province"))
	require.False(t, isVictory("smithy"))
	require.False(t, isVictory("copper"))
}
