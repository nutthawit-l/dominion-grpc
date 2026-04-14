package engine

import "math/rand"

// shuffleCards shuffles a slice of CardID in place using the given
// per-game RNG. All shuffles in the engine must go through this
// function so game state is reproducible from seed + action log.
func shuffleCards(r *rand.Rand, cards []CardID) {
	r.Shuffle(len(cards), func(i, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})
}
