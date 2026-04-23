package cards

import (
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

// registryLookup is the CardLookup function backed by DefaultRegistry.
// Used by Tier 3 attack cards that need to scan opponents' hands for
// reactions inside OnPlay (where the engine does not pass lookup in).
func registryLookup(id engine.CardID) (*engine.Card, bool) {
	return DefaultRegistry.Lookup(id)
}
