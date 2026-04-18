package main

import (
	"log"
	"net/http"

	"github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1/dominionv1connect"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/nutthawit-l/dominion-grpc/internal/engine/cards"
	"github.com/nutthawit-l/dominion-grpc/internal/service"
	"github.com/nutthawit-l/dominion-grpc/internal/store"
)

func main() {
	lookup := func(id engine.CardID) (*engine.Card, bool) {
		return cards.DefaultRegistry.Lookup(id)
	}
	svc := service.NewGameService(store.NewMemory(), lookup)

	mux := http.NewServeMux()
	path, h := dominionv1connect.NewGameServiceHandler(svc)
	mux.Handle(path, h)

	log.Println("dominion-grpc server listening on :8080")
	if err := http.ListenAndServe(":8080", h2cMux(mux)); err != nil {
		log.Fatal(err)
	}
}
