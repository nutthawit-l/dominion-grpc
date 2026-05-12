package store

import (
	"errors"
	"sync"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var ErrGameNotFound = errors.New("store: game not found")

// Memory is a goroutine-safe in-memory game registry. Every mutation
// via WithLock holds a per-store mutex; reads via Get are lock-free
// (but callers must not mutate the returned state without going through
// WithLock).
type Memory struct {
	mu    sync.Mutex
	games map[string]*engine.GameState
	codes map[string]string // room code → game id
}

// NewMemory returns an empty registry.
func NewMemory() *Memory {
	return &Memory{
		games: map[string]*engine.GameState{},
		codes: map[string]string{},
	}
}

// Put stores a new game with its room code alias.
func (m *Memory) Put(s *engine.GameState, code string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.games[s.GameID] = s
	m.codes[code] = s.GameID
}

// GetByCode resolves a room code to a game id.
func (m *Memory) GetByCode(code string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.codes[code]
	return id, ok
}

// Get returns the state for a game id. Callers must not mutate it.
func (m *Memory) Get(id string) (*engine.GameState, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.games[id]
	return s, ok
}

// WithLock calls fn with the state for a game id, holding the registry
// mutex for the duration. fn may mutate the state.
func (m *Memory) WithLock(id string, fn func(*engine.GameState) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.games[id]
	if !ok {
		return ErrGameNotFound
	}
	return fn(s)
}
