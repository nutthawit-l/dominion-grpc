package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

// Registry holds card definitions by ID. It is not safe for concurrent
// registration — all cards should be registered at package init time
// from a single goroutine.
type Registry struct {
	byID map[engine.CardID]*engine.Card
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{byID: map[engine.CardID]*engine.Card{}}
}

// Register adds c to the registry. Panics if c.ID is already present.
func (r *Registry) Register(c *engine.Card) {
	if _, exists := r.byID[c.ID]; exists {
		panic(fmt.Sprintf("dominion: duplicate card ID %q", c.ID))
	}
	r.byID[c.ID] = c
}

// Lookup returns the card for the given ID, if any.
func (r *Registry) Lookup(id engine.CardID) (*engine.Card, bool) {
	c, ok := r.byID[id]
	return c, ok
}

// All returns every registered card in an unspecified order.
func (r *Registry) All() []*engine.Card {
	out := make([]*engine.Card, 0, len(r.byID))
	for _, c := range r.byID {
		out = append(out, c)
	}
	return out
}

// DefaultRegistry is the package-global registry populated via init()
// by each card file. Tests that need a fresh registry should call
// NewRegistry() directly.
var DefaultRegistry = NewRegistry()

func init() {
	engine.RegisterKingdomLister(func() []*engine.Card {
		return DefaultRegistry.All()
	})
}
