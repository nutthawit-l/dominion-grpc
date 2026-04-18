package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/nutthawit-l/dominion-grpc/internal/bot"
)

func main() {
	server := flag.String("server", "http://localhost:8080", "Connect server base URL")
	gameID := flag.String("game", "", "game ID to join (required unless --create)")
	create := flag.Bool("create", false, "create a new bot-only game before joining")
	asPlayer := flag.Int("as-player", 0, "player index this bot plays")
	strategyName := flag.String("strategy", "bigmoney", "strategy to use (bigmoney)")
	seed := flag.Int64("seed", 1, "game seed (only used with --create)")
	flag.Parse()

	strat, err := selectStrategy(*strategyName)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	c := bot.NewClient(*server)

	id := *gameID
	if *create {
		resp, err := c.CreateGame(ctx, []string{"bigmoney", "bigmoney"}, *seed)
		if err != nil {
			log.Fatalf("create: %v", err)
		}
		id = resp.GameId
		fmt.Printf("created game %s\n", id)
	}
	if id == "" {
		log.Fatal("--game or --create is required")
	}

	if err := bot.Run(ctx, c, id, *asPlayer, strat); err != nil {
		log.Fatalf("run: %v", err)
	}
}

func selectStrategy(name string) (bot.Strategy, error) {
	switch name {
	case "bigmoney":
		return bot.BigMoney{}, nil
	}
	return nil, fmt.Errorf("unknown strategy %q", name)
}
