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

	game, err := a.CreateGame(ctx, []string{"bigmoney", "bigmoney"}, 42)
	require.NoError(t, err)
	require.NotEmpty(t, game.GameId)

	grp, gctx := errgroup.WithContext(ctx)
	grp.Go(func() error { return bot.Run(gctx, a, game.GameId, 0, bot.BigMoney{}) })
	grp.Go(func() error { return bot.Run(gctx, b, game.GameId, 1, bot.BigMoney{}) })
	require.NoError(t, grp.Wait())
}
