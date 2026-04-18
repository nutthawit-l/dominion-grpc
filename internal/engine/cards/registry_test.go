package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestRegister_And_Lookup(t *testing.T) {
	reg := NewRegistry()
	c := &engine.Card{ID: "smithy", Name: "Smithy", Cost: 4}
	reg.Register(c)

	got, ok := reg.Lookup("smithy")
	require.True(t, ok)
	require.Same(t, c, got)
}

func TestRegister_DuplicatePanics(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&engine.Card{ID: "copper"})
	require.Panics(t, func() {
		reg.Register(&engine.Card{ID: "copper"})
	})
}

func TestLookup_MissingReturnsFalse(t *testing.T) {
	reg := NewRegistry()
	_, ok := reg.Lookup("nope")
	require.False(t, ok)
}

func TestAll_ReturnsAllRegisteredCards(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&engine.Card{ID: "copper"})
	reg.Register(&engine.Card{ID: "silver"})
	require.Len(t, reg.All(), 2)
}
