package bot_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/nutthawit-l/dominion-grpc/internal/bot"
	"github.com/nutthawit-l/dominion-grpc/internal/engine/cards"
)

// TestBotVsBot_FullBaseSet_RandomKingdoms is the Phase-1a closeout
// sweep. Per the parent design spec's "Done when" criterion: 1000
// random games over random 10-card kingdom subsets drawn from the
// full Base-set pool and random strategy pairs from the registered
// set. Failure conditions: any panic, any Connect error, game does
// not terminate within the configured timeout, or final state is
// missing.
func TestBotVsBot_FullBaseSet_RandomKingdoms(t *testing.T) {
	if testing.Short() {
		t.Skip("Phase-1a closeout sweep — skipped under -short")
	}

	const games = 1000

	pool := allKingdomCardIDs(t)
	require.GreaterOrEqualf(t, len(pool), 26,
		"Tier 5 closeout requires all 26 kingdom cards registered; got %d", len(pool))

	strategies := []string{
		"bigmoney", "smithy_bm", "chapel_bm", "remodel_bm",
		"witch_bm", "militia_bm", "throneroom_bm", "library_bm",
		"sentry_bm", "merchant_bm", "gardens_bm",
	}

	srv := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for i := 0; i < games; i++ {
		seed := int64(i)
		kingdom := pickRandomKingdom(seed, pool, 10)
		aName, bName := pickRandomStrategyPair(seed, strategies)

		t.Run(fmt.Sprintf("seed=%d/%s_vs_%s", seed, aName, bName), func(t *testing.T) {
			runOneFullSetGame(t, ctx, srv.URL, seed, kingdom, aName, bName)
		})
	}
}

// allKingdomCardIDs reads every kingdom card from cards.DefaultRegistry.
func allKingdomCardIDs(t *testing.T) []string {
	t.Helper()
	out := []string{}
	for _, c := range cards.DefaultRegistry.All() {
		if c.IsKingdom() {
			out = append(out, string(c.ID))
		}
	}
	return out
}

// pickRandomKingdom draws size cards from pool deterministically by seed.
func pickRandomKingdom(seed int64, pool []string, size int) []string {
	r := rand.New(rand.NewSource(seed))
	cp := append([]string(nil), pool...)
	r.Shuffle(len(cp), func(i, j int) { cp[i], cp[j] = cp[j], cp[i] })
	if size > len(cp) {
		size = len(cp)
	}
	return cp[:size]
}

// pickRandomStrategyPair returns two strategy names chosen by seed.
func pickRandomStrategyPair(seed int64, names []string) (string, string) {
	r := rand.New(rand.NewSource(seed))
	a := names[r.Intn(len(names))]
	b := names[r.Intn(len(names))]
	return a, b
}

// runOneFullSetGame plays one game with the named strategies.
func runOneFullSetGame(t *testing.T, parent context.Context, url string,
	seed int64, kingdom []string, aName, bName string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	a := bot.NewClient(url)
	b := bot.NewClient(url)

	game, err := a.CreateGame(ctx, []string{aName, bName}, seed, kingdom)
	require.NoErrorf(t, err, "CreateGame seed=%d", seed)

	sa, err := bot.StrategyByName(aName)
	require.NoError(t, err)
	sb, err := bot.StrategyByName(bName)
	require.NoError(t, err)

	grp, gctx := errgroup.WithContext(ctx)
	grp.Go(func() error { return bot.Run(gctx, a, game.GameId, 0, sa) })
	grp.Go(func() error { return bot.Run(gctx, b, game.GameId, 1, sb) })
	require.NoErrorf(t, grp.Wait(), "seed=%d kingdom=%v", seed, kingdom)
}
