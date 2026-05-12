package store

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestMemory_PutAndGet(t *testing.T) {
	m := NewMemory()
	s := &engine.GameState{GameID: "g1"}
	m.Put(s, "ATEST")

	got, ok := m.Get("g1")
	require.True(t, ok)
	require.Same(t, s, got)
}

func TestMemory_GetMissing(t *testing.T) {
	m := NewMemory()
	_, ok := m.Get("nope")
	require.False(t, ok)
}

func TestMemory_WithLock(t *testing.T) {
	m := NewMemory()
	m.Put(&engine.GameState{GameID: "g", Turn: 0}, "BTEST")

	err := m.WithLock("g", func(s *engine.GameState) error {
		s.Turn = 5
		return nil
	})
	require.NoError(t, err)

	got, _ := m.Get("g")
	require.Equal(t, 5, got.Turn)
}

func TestMemory_WithLock_Missing(t *testing.T) {
	m := NewMemory()
	err := m.WithLock("nope", func(*engine.GameState) error { return nil })
	require.Error(t, err)
}
