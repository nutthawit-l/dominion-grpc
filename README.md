# dominion-grpc

A server-authoritative Dominion implementation in Go. Uses Connect-Go over
HTTP for the RPC layer; protobuf is the single source of truth for the
contract between server and any client.

**Status:** Phase 1a / Tier 0 — basics + BigMoney. See `docs/` for design
and plans (in the dev-env repo).

## Build

    make generate    # regenerate gen/go/ from proto/
    make test        # run full test suite
    make test-short  # skip the 1000-game integration sweep
    make server      # run Connect server on :8080
    make bot         # run the standalone bot CLI

## Layout

    proto/               source of truth — protobuf contract
    gen/go/              generated Go stubs (committed; never hand-edit)
    cmd/server/          Connect-Go HTTP server entry point
    cmd/bot/             bot CLI entry point
    internal/engine/     pure Go game engine (no network, no protobuf)
    internal/service/    Connect handlers (translation layer)
    internal/store/      in-memory game registry
    internal/bot/        bot library: state reducer, strategies, run loop