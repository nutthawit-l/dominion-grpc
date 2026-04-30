package bot_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"

	"github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1/dominionv1connect"
	"github.com/nutthawit-l/dominion-grpc/internal/bot"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/nutthawit-l/dominion-grpc/internal/engine/cards"
	"github.com/nutthawit-l/dominion-grpc/internal/service"
	"github.com/nutthawit-l/dominion-grpc/internal/store"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	lookup := func(id engine.CardID) (*engine.Card, bool) {
		return cards.DefaultRegistry.Lookup(id)
	}
	svc := service.NewGameService(store.NewMemory(), lookup)

	mux := http.NewServeMux()
	path, h := dominionv1connect.NewGameServiceHandler(svc)
	mux.Handle(path, h)

	srv := httptest.NewUnstartedServer(h2c.NewHandler(mux, &http2.Server{}))
	srv.EnableHTTP2 = true
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}

func TestBotVsBot_BigMoney(t *testing.T) {
	srv := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := bot.NewClient(srv.URL)
	b := bot.NewClient(srv.URL)

	game, err := a.CreateGame(ctx, []string{"bigmoney", "bigmoney"}, 42, nil)
	require.NoError(t, err)
	require.NotEmpty(t, game.GameId)

	grp, gctx := errgroup.WithContext(ctx)
	grp.Go(func() error { return bot.Run(gctx, a, game.GameId, 0, bot.BigMoney{}) })
	grp.Go(func() error { return bot.Run(gctx, b, game.GameId, 1, bot.BigMoney{}) })
	require.NoError(t, grp.Wait())
}

func TestBotVsBot_SmithyBM_Outperforms_BigMoney(t *testing.T) {
	if testing.Short() {
		t.Skip("integration sweep — skipped under -short")
	}

	const games = 200
	const threshold = 0.55

	srv := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	wins := 0
	for seed := int64(0); seed < games; seed++ {
		smithySeat := int(seed % 2)
		bmSeat := 1 - smithySeat

		names := make([]string, 2)
		names[smithySeat] = "smithy_bm"
		names[bmSeat] = "bigmoney"

		a := bot.NewClient(srv.URL)
		b := bot.NewClient(srv.URL)
		game, err := a.CreateGame(ctx, names, seed, []string{"smithy"})
		require.NoError(t, err)

		strategies := map[int]bot.Strategy{
			smithySeat: bot.SmithyBM{},
			bmSeat:     bot.BigMoney{},
		}

		grp, gctx := errgroup.WithContext(ctx)
		grp.Go(func() error { return bot.Run(gctx, a, game.GameId, 0, strategies[0]) })
		grp.Go(func() error { return bot.Run(gctx, b, game.GameId, 1, strategies[1]) })
		require.NoError(t, grp.Wait(), "seed=%d", seed)

		winners := finalWinners(t, srv, game.GameId)
		if len(winners) == 1 && winners[0] == smithySeat {
			wins++
		}
	}

	rate := float64(wins) / float64(games)
	require.GreaterOrEqualf(t, rate, threshold,
		"SmithyBM win rate %.2f below threshold %.2f over %d games", rate, threshold, games)
}

// Chapel+BigMoney with a supply limited to Chapel is known to run roughly
// even with pure Big Money: deck thinning helps but losing Estates costs
// VP. This sweep is a regression check that ChapelBM stays competitive,
// not that it strictly outperforms.
func TestBotVsBot_ChapelBM_CompetitiveWith_BigMoney(t *testing.T) {
	if testing.Short() {
		t.Skip("integration sweep — skipped under -short")
	}

	const games = 200
	const threshold = 0.48

	srv := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	wins := 0
	for seed := int64(0); seed < games; seed++ {
		chapelSeat := int(seed % 2)
		bmSeat := 1 - chapelSeat

		names := make([]string, 2)
		names[chapelSeat] = "chapel_bm"
		names[bmSeat] = "bigmoney"

		a := bot.NewClient(srv.URL)
		b := bot.NewClient(srv.URL)
		game, err := a.CreateGame(ctx, names, seed, []string{"chapel"})
		require.NoError(t, err)

		strategies := map[int]bot.Strategy{
			chapelSeat: bot.NewChapelBM(),
			bmSeat:     bot.BigMoney{},
		}

		grp, gctx := errgroup.WithContext(ctx)
		grp.Go(func() error { return bot.Run(gctx, a, game.GameId, 0, strategies[0]) })
		grp.Go(func() error { return bot.Run(gctx, b, game.GameId, 1, strategies[1]) })
		require.NoError(t, grp.Wait(), "seed=%d", seed)

		winners := finalWinners(t, srv, game.GameId)
		if len(winners) == 1 && winners[0] == chapelSeat {
			wins++
		}
	}

	rate := float64(wins) / float64(games)
	require.GreaterOrEqualf(t, rate, threshold,
		"ChapelBM win rate %.2f below threshold %.2f over %d games", rate, threshold, games)
}

func TestBotVsBot_RemodelBM(t *testing.T) {
	srv := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a := bot.NewClient(srv.URL)
	b := bot.NewClient(srv.URL)

	game, err := a.CreateGame(ctx, []string{"remodel_bm", "bigmoney"}, 42, []string{"remodel"})
	require.NoError(t, err)

	grp, gctx := errgroup.WithContext(ctx)
	grp.Go(func() error { return bot.Run(gctx, a, game.GameId, 0, bot.NewRemodelBM()) })
	grp.Go(func() error { return bot.Run(gctx, b, game.GameId, 1, bot.BigMoney{}) })
	require.NoError(t, grp.Wait())
}

func TestBotVsBot_WitchBM_Outperforms_BigMoney(t *testing.T) {
	if testing.Short() {
		t.Skip("integration sweep — skipped under -short")
	}

	const games = 200
	const threshold = 0.55

	srv := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	wins := 0
	for seed := int64(0); seed < games; seed++ {
		witchSeat := int(seed % 2)
		bmSeat := 1 - witchSeat

		names := make([]string, 2)
		names[witchSeat] = "witch_bm"
		names[bmSeat] = "bigmoney"

		a := bot.NewClient(srv.URL)
		b := bot.NewClient(srv.URL)
		game, err := a.CreateGame(ctx, names, seed, []string{"witch"})
		require.NoError(t, err)

		strategies := map[int]bot.Strategy{
			witchSeat: bot.NewWitchBM(),
			bmSeat:    bot.BigMoney{},
		}

		grp, gctx := errgroup.WithContext(ctx)
		grp.Go(func() error { return bot.Run(gctx, a, game.GameId, 0, strategies[0]) })
		grp.Go(func() error { return bot.Run(gctx, b, game.GameId, 1, strategies[1]) })
		require.NoError(t, grp.Wait(), "seed=%d", seed)

		winners := finalWinners(t, srv, game.GameId)
		if len(winners) == 1 && winners[0] == witchSeat {
			wins++
		}
	}

	rate := float64(wins) / float64(games)
	require.GreaterOrEqualf(t, rate, threshold,
		"WitchBM win rate %.2f below threshold %.2f over %d games", rate, threshold, games)
}

func TestBotVsBot_MilitiaBM_CompletesNormally(t *testing.T) {
	if testing.Short() {
		t.Skip("integration sweep — skipped under -short")
	}

	const games = 50

	srv := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	wins := 0
	for seed := int64(0); seed < games; seed++ {
		militiaSeat := int(seed % 2)
		bmSeat := 1 - militiaSeat

		names := make([]string, 2)
		names[militiaSeat] = "militia_bm"
		names[bmSeat] = "bigmoney"

		a := bot.NewClient(srv.URL)
		b := bot.NewClient(srv.URL)
		game, err := a.CreateGame(ctx, names, seed, []string{"militia"})
		require.NoError(t, err)

		strategies := map[int]bot.Strategy{
			militiaSeat: bot.NewMilitiaBM(),
			bmSeat:      bot.BigMoney{},
		}

		grp, gctx := errgroup.WithContext(ctx)
		grp.Go(func() error { return bot.Run(gctx, a, game.GameId, 0, strategies[0]) })
		grp.Go(func() error { return bot.Run(gctx, b, game.GameId, 1, strategies[1]) })
		require.NoError(t, grp.Wait(), "seed=%d", seed)

		winners := finalWinners(t, srv, game.GameId)
		if len(winners) == 1 && winners[0] == militiaSeat {
			wins++
		}
	}

	require.GreaterOrEqualf(t, wins, 1,
		"MilitiaBM lost every one of %d games — indicative of engine bug, not strategy weakness", games)
}

func TestBotVsBot_ThroneRoomBM_Outperforms_BigMoney(t *testing.T) {
	if testing.Short() {
		t.Skip("integration sweep — skipped under -short")
	}

	const games = 200
	const threshold = 0.55

	srv := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	wins := 0
	for seed := int64(0); seed < games; seed++ {
		trSeat := int(seed % 2)
		bmSeat := 1 - trSeat

		names := make([]string, 2)
		names[trSeat] = "throneroom_bm"
		names[bmSeat] = "bigmoney"

		a := bot.NewClient(srv.URL)
		b := bot.NewClient(srv.URL)
		kingdom := []string{
			"throne_room", "witch", "smithy", "village", "market",
			"laboratory", "festival", "council_room", "moat", "mine",
		}
		game, err := a.CreateGame(ctx, names, seed, kingdom)
		require.NoError(t, err)

		strategies := map[int]bot.Strategy{
			trSeat: bot.NewThroneRoomBM(),
			bmSeat: bot.BigMoney{},
		}

		grp, gctx := errgroup.WithContext(ctx)
		grp.Go(func() error { return bot.Run(gctx, a, game.GameId, 0, strategies[0]) })
		grp.Go(func() error { return bot.Run(gctx, b, game.GameId, 1, strategies[1]) })
		require.NoError(t, grp.Wait(), "seed=%d", seed)

		winners := finalWinners(t, srv, game.GameId)
		if len(winners) == 1 && winners[0] == trSeat {
			wins++
		}
	}

	rate := float64(wins) / float64(games)
	require.GreaterOrEqualf(t, rate, threshold,
		"ThroneRoomBM win rate %.2f below threshold %.2f over %d games", rate, threshold, games)
}

func finalWinners(t *testing.T, srv *httptest.Server, gameID string) []int {
	t.Helper()
	c := bot.NewClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := c.StreamGameEvents(ctx, gameID, 0)
	require.NoError(t, err)
	defer stream.Close()
	ev, ok := stream.Receive()
	require.True(t, ok)
	snap := ev.GetSnapshot()
	require.NotNil(t, snap)
	out := make([]int, 0, len(snap.Winners))
	for _, w := range snap.Winners {
		out = append(out, int(w))
	}
	return out
}
