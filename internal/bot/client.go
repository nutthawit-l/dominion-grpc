package bot

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	pb "github.com/tie/dominion-grpc/gen/go/dominion/v1"
	"github.com/tie/dominion-grpc/gen/go/dominion/v1/dominionv1connect"
)

// Client is a thin wrapper around the generated Connect client that
// hides the request/response wrapping boilerplate.
type Client struct {
	inner dominionv1connect.GameServiceClient
}

// NewClient returns a Client talking to the given base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		inner: dominionv1connect.NewGameServiceClient(http.DefaultClient, baseURL),
	}
}

// CreateGame creates a new game with the given player names and seed.
func (c *Client) CreateGame(ctx context.Context, players []string, seed int64) (*pb.CreateGameResponse, error) {
	resp, err := c.inner.CreateGame(ctx, connect.NewRequest(&pb.CreateGameRequest{
		Players: players, Seed: seed,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// SubmitAction sends an action and returns its response.
func (c *Client) SubmitAction(ctx context.Context, gameID string, a *pb.Action) error {
	_, err := c.inner.SubmitAction(ctx, connect.NewRequest(&pb.SubmitActionRequest{
		GameId: gameID, Action: a,
	}))
	return err
}

// Stream wraps a server-streaming connection for game events.
type Stream struct {
	inner *connect.ServerStreamForClient[pb.StreamGameEventsResponse]
}

// StreamGameEvents begins subscribing.
func (c *Client) StreamGameEvents(ctx context.Context, gameID string, playerIdx int) (*Stream, error) {
	s, err := c.inner.StreamGameEvents(ctx, connect.NewRequest(&pb.StreamGameEventsRequest{
		GameId: gameID, PlayerIdx: int32(playerIdx),
	}))
	if err != nil {
		return nil, err
	}
	return &Stream{inner: s}, nil
}

// Receive returns the next event or false if the stream is closed.
func (s *Stream) Receive() (*pb.StreamGameEventsResponse, bool) {
	if !s.inner.Receive() {
		return nil, false
	}
	return s.inner.Msg(), true
}

// Close releases the stream.
func (s *Stream) Close() error { return s.inner.Close() }

// Err returns the terminal error, if any.
func (s *Stream) Err() error { return s.inner.Err() }
