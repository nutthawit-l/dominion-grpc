# dominion-grpc Phase 1a Tier 5 — Special hooks + Phase-1a closeout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the final two Base-set kingdom cards — **Gardens** (Victory whose VP is `floor(deckSize/10)`) and **Merchant** (Action with a per-turn "first Silver" trigger) — and close out Phase 1a with a 1000-game random-kingdom integration sweep over the full 26-card pool, smoke sweeps for the two new strategies, a `bot.StrategyByName` factory, and a README that declares Phase 1a complete.

**Architecture:** Two scalar fields on `PlayerState` (`MerchantBonusCharges int`, `FirstSilverPlayedThisTurn bool`) carry the Merchant trigger across plays, and `cleanupAndEndTurn` resets them at end of turn. `Silver.OnPlay` reads them and pays the bonus on its first play of a turn. `Card.IsKingdom()` is widened to accept Victory cards whose ID is not in `{estate, duchy, province}`, so Gardens auto-registers as a kingdom card. No new prompts, decisions, events, or proto messages.

**Tech Stack:** Go 1.22+, protobuf (proto3), `connectrpc.com/connect`, `github.com/stretchr/testify`, `golang.org/x/sync/errgroup`, `buf` ≥ 1.34, `golangci-lint` ≥ 1.58.

**Source spec:** [docs/superpowers/specs/2026-05-01-dominion-grpc-tier5-design.md](../specs/2026-05-01-dominion-grpc-tier5-design.md).

---

## Conventions

- **Working directory:** `/home/tie/superpower-dominion/dominion-grpc/` for every task.
- **Go module path:** `github.com/nutthawit-l/dominion-grpc`. Already configured in `go.mod`; never re-init it.
- **Commit style:** Conventional Commits — `feat(engine): …`, `feat(cards): …`, `feat(bot): …`, `chore(scope): …`, `test(scope): …`, `docs: …`. Keep one logical change per commit.
- **TDD discipline:** every behavior step writes the failing test first, runs it to **see it fail**, implements the minimum to pass, runs it again to **see it pass**, then commits. Skipping the run-to-fail step silently ships broken tests. Do not skip it.
- **No proto changes in this tier.** `make generate` should not be needed — but run it anyway at the end as a sanity check.
- **Test running:** prefer the package-targeted form when iterating, e.g. `go test ./internal/engine/cards -run TestMerchant -v`. Use `make test-short` to skip the long sweeps. Run full `make test` before claiming a task is done.
- **Lint:** `make lint` runs `buf lint` + `golangci-lint run ./...`. Keep it green at every commit.

## Task list summary

Stage A — Engine foundation (unblocks the cards):

1. `IsKingdom()` predicate fix — accept non-basic Victory cards
2. `MerchantBonusCharges` + `FirstSilverPlayedThisTurn` fields on `PlayerState` + cleanup reset

Stage B — The two cards (independent of each other after Stage A):

3. Gardens (`kingdom_gardens.go`) + scoring extension
4. Merchant (`kingdom_merchant.go`) + Silver trigger edit + TR-of-Merchant regression test

Stage C — Strategies + closeout:

5. MerchantBM + GardensBM strategies + smoke sweeps + `bot.StrategyByName` factory + `cmd/bot/main.go` refactor
6. Phase-1a closeout: full-set 1000-game sweep + README update declaring Phase 1a complete

---

## Task 1: `IsKingdom()` predicate fix — accept non-basic Victory cards

Adds the predicate change that lets Gardens auto-register as a kingdom card. **No card consumes the change yet** (Gardens doesn't exist until Task 3); this lands self-contained with synthetic-card tests.

**Files:**
- Modify: `internal/engine/card.go` — extend `IsKingdom()`.
- Modify: `internal/engine/newgame_test.go` — extend `TestIsKingdom_ActionCardsQualify` with Gardens-shaped cases.

- [ ] **Step 1: Write the failing test cases**

Open `internal/engine/newgame_test.go` and **replace** the body of `TestIsKingdom_ActionCardsQualify` with the expanded version below (preserve the existing function name and signature):

```go
func TestIsKingdom_ActionCardsQualify(t *testing.T) {
    action := &Card{ID: "x", Types: []CardType{TypeAction}}
    treasure := &Card{ID: "y", Types: []CardType{TypeTreasure}}
    estate := &Card{ID: "estate", Types: []CardType{TypeVictory}}
    duchy := &Card{ID: "duchy", Types: []CardType{TypeVictory}}
    province := &Card{ID: "province", Types: []CardType{TypeVictory}}
    curse := &Card{ID: "w", Types: []CardType{TypeCurse}}
    actionAttack := &Card{ID: "v", Types: []CardType{TypeAction, TypeAttack}}
    actionReaction := &Card{ID: "u", Types: []CardType{TypeAction, TypeReaction}}
    gardensShape := &Card{ID: "gardens", Types: []CardType{TypeVictory}}

    require.True(t, action.IsKingdom())
    require.False(t, treasure.IsKingdom())
    require.False(t, estate.IsKingdom(), "Estate is basic Victory, not kingdom")
    require.False(t, duchy.IsKingdom(), "Duchy is basic Victory, not kingdom")
    require.False(t, province.IsKingdom(), "Province is basic Victory, not kingdom")
    require.False(t, curse.IsKingdom())
    require.True(t, actionAttack.IsKingdom())
    require.True(t, actionReaction.IsKingdom())
    require.True(t, gardensShape.IsKingdom(),
        "non-basic Victory cards (Gardens-shape) ARE kingdom")
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
go test ./internal/engine -run TestIsKingdom_ActionCardsQualify -v
```

Expected: FAIL — at minimum, `gardensShape.IsKingdom()` returns false because the current predicate only accepts `TypeAction`. The Estate/Duchy/Province subasserts pass with the old predicate too (they're false for non-Action), so the diff failure will be on the `gardensShape` assertion.

- [ ] **Step 3: Implement the predicate fix**

Edit `internal/engine/card.go`. Replace the entire `IsKingdom` method body:

```go
// IsKingdom reports whether c is a kingdom card. A kingdom card is any
// card tagged TypeAction (plain actions, attacks, reactions), OR any
// non-basic Victory card (Gardens-shape). Estate / Duchy / Province
// are basic Victory and therefore NOT kingdom; treasures and curses
// are likewise basic.
func (c *Card) IsKingdom() bool {
    if c.HasType(TypeAction) {
        return true
    }
    if c.HasType(TypeVictory) {
        switch c.ID {
        case "estate", "duchy", "province":
            return false
        }
        return true
    }
    return false
}
```

- [ ] **Step 4: Run the test to confirm it passes**

```bash
go test ./internal/engine -run TestIsKingdom_ActionCardsQualify -v
```

Expected: PASS.

- [ ] **Step 5: Run the full engine package to make sure nothing else broke**

```bash
go test ./internal/engine -short
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/engine/card.go internal/engine/newgame_test.go
git commit -m "$(cat <<'EOF'
feat(engine): widen IsKingdom to accept non-basic Victory cards

Tier 5 introduces Gardens, a Victory-only kingdom card. Old predicate
required TypeAction; new predicate also accepts Victory cards whose ID
is not estate/duchy/province. No existing card behavior changes.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: `MerchantBonusCharges` + `FirstSilverPlayedThisTurn` fields on `PlayerState` + cleanup reset

Adds the per-turn scalar state Merchant and Silver will read in Tasks 3–4. **No card consumes them yet**; this lands self-contained with a cleanup-reset test.

**Files:**
- Modify: `internal/engine/state.go` — add two fields to `PlayerState`.
- Modify: `internal/engine/phases.go` — reset both fields in `cleanupAndEndTurn`.
- Modify: `internal/engine/phases_test.go` — add `TestCleanup_ResetsMerchantFields`.

- [ ] **Step 1: Write the failing test**

Append to `internal/engine/phases_test.go`:

```go
func TestCleanup_ResetsMerchantFields(t *testing.T) {
    gs := newTestState(2)
    gs.CurrentPlayer = 0
    gs.StartingPlayer = 0
    gs.Phase = PhaseCleanup
    // Pre-load the per-turn Merchant trigger fields for the active player.
    gs.Players[0].MerchantBonusCharges = 3
    gs.Players[0].FirstSilverPlayedThisTurn = true
    fillDeck(gs, 0, "copper", 5)
    fillDeck(gs, 1, "copper", 5)

    cleanupAndEndTurn(gs)

    require.Equal(t, 0, gs.Players[0].MerchantBonusCharges,
        "MerchantBonusCharges must reset to 0 at end of turn")
    require.False(t, gs.Players[0].FirstSilverPlayedThisTurn,
        "FirstSilverPlayedThisTurn must reset to false at end of turn")
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
go test ./internal/engine -run TestCleanup_ResetsMerchantFields -v
```

Expected: FAIL with `gs.Players[0].MerchantBonusCharges undefined (type PlayerState has no field or method MerchantBonusCharges)`.

- [ ] **Step 3: Add the fields to `PlayerState`**

Edit `internal/engine/state.go`. Inside the `PlayerState` struct, after the `Coins int` line, add:

```go
    // MerchantBonusCharges is incremented by Merchant.OnPlay each time
    // Merchant is played in the current turn, and drained the first
    // time Silver is played that turn. Reset to 0 by cleanupAndEndTurn.
    MerchantBonusCharges int

    // FirstSilverPlayedThisTurn flips true the first time Silver.OnPlay
    // runs in a given turn. Reset to false by cleanupAndEndTurn. Guards
    // against repeat triggers within a turn.
    FirstSilverPlayedThisTurn bool
```

- [ ] **Step 4: Confirm the package compiles**

```bash
go build ./internal/engine
```

Expected: exit 0, no output.

- [ ] **Step 5: Add the cleanup reset**

Edit `internal/engine/phases.go`. Locate the existing reset block in `cleanupAndEndTurn` (the lines that zero `Actions`, `Buys`, `Coins` for the outgoing player `px`):

```go
// Reset resources — they only apply to the current player's turn
// and are re-set below for the NEXT player.
gs.Players[px].Actions = 0
gs.Players[px].Buys = 0
gs.Players[px].Coins = 0
```

Append two more lines so the block becomes:

```go
// Reset resources — they only apply to the current player's turn
// and are re-set below for the NEXT player.
gs.Players[px].Actions = 0
gs.Players[px].Buys = 0
gs.Players[px].Coins = 0
gs.Players[px].MerchantBonusCharges = 0
gs.Players[px].FirstSilverPlayedThisTurn = false
```

- [ ] **Step 6: Run the test to confirm it passes**

```bash
go test ./internal/engine -run TestCleanup_ResetsMerchantFields -v
```

Expected: PASS.

- [ ] **Step 7: Run the full engine package to make sure nothing else broke**

```bash
go test ./internal/engine -short
```

Expected: PASS — including the existing cleanup tests.

- [ ] **Step 8: Commit**

```bash
git add internal/engine/state.go internal/engine/phases.go internal/engine/phases_test.go
git commit -m "$(cat <<'EOF'
feat(engine): add MerchantBonusCharges + FirstSilverPlayedThisTurn

Per-turn scalar fields on PlayerState backing Tier 5's Merchant trigger.
Reset by cleanupAndEndTurn alongside Actions/Buys/Coins. No card consumes
them yet; Tier 5 cards (Merchant + Silver edit) follow.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Gardens — `kingdom_gardens.go` + scoring extension

Adds the Gardens card. Now that `IsKingdom()` accepts non-basic Victory cards (Task 1), registering Gardens via `init()` automatically makes it eligible for kingdom selection. `ComputeScore` already iterates `Hand+Deck+Discard+InPlay` and calls `VictoryPoints(p)` per copy — Gardens' `VictoryPoints` reads PlayerState directly to compute `floor(n/10)`.

**Files:**
- Create: `internal/engine/cards/kingdom_gardens.go` — Gardens card definition.
- Create: `internal/engine/cards/kingdom_gardens_test.go` — per-card unit tests.
- Modify: `internal/engine/scoring_test.go` — add `TestComputeScore_GardensAcrossZones`.

- [ ] **Step 1: Write failing tests for Gardens**

Create `internal/engine/cards/kingdom_gardens_test.go`:

```go
package cards

import (
    "testing"

    "github.com/nutthawit-l/dominion-grpc/internal/engine"
    "github.com/stretchr/testify/require"
)

func TestGardens_Metadata(t *testing.T) {
    require.Equal(t, engine.CardID("gardens"), Gardens.ID)
    require.Equal(t, "Gardens", Gardens.Name)
    require.Equal(t, 4, Gardens.Cost)
    require.True(t, Gardens.HasType(engine.TypeVictory))
    require.False(t, Gardens.HasType(engine.TypeAction),
        "Gardens is Victory-only, not Action")
    require.True(t, Gardens.IsKingdom(),
        "Gardens must register as kingdom (Task 1's IsKingdom widening)")
    _, ok := DefaultRegistry.Lookup("gardens")
    require.True(t, ok, "Gardens must be registered in DefaultRegistry")
}

func TestGardens_VictoryPoints_FewerThanTen_IsZero(t *testing.T) {
    cases := []int{0, 1, 5, 9}
    for _, n := range cases {
        ps := engine.PlayerState{Hand: makeCards("copper", n)}
        require.Equalf(t, 0, Gardens.VictoryPoints(ps),
            "%d cards → expected 0 VP", n)
    }
}

func TestGardens_VictoryPoints_TenCards_IsOne(t *testing.T) {
    ps := engine.PlayerState{Hand: makeCards("copper", 10)}
    require.Equal(t, 1, Gardens.VictoryPoints(ps))
}

func TestGardens_VictoryPoints_RoundsDown(t *testing.T) {
    ps := engine.PlayerState{Hand: makeCards("copper", 19)}
    require.Equal(t, 1, Gardens.VictoryPoints(ps),
        "19 cards → 1 VP (round down)")
    ps = engine.PlayerState{Hand: makeCards("copper", 20)}
    require.Equal(t, 2, Gardens.VictoryPoints(ps))
}

func TestGardens_VictoryPoints_CountsAllZones(t *testing.T) {
    // 2 cards in each of the 5 zones = 10 total → 1 VP.
    ps := engine.PlayerState{
        Hand:     makeCards("copper", 2),
        Deck:     makeCards("copper", 2),
        Discard:  makeCards("copper", 2),
        InPlay:   makeCards("copper", 2),
        SetAside: makeCards("copper", 2),
    }
    require.Equal(t, 1, Gardens.VictoryPoints(ps),
        "Gardens must count Hand+Deck+Discard+InPlay+SetAside")
}

func TestGardens_HasNoOnPlay(t *testing.T) {
    require.Nil(t, Gardens.OnPlay,
        "Gardens is Victory-only and has no OnPlay")
}

// makeCards returns a slice of n copies of the given card ID.
func makeCards(id engine.CardID, n int) []engine.CardID {
    out := make([]engine.CardID, n)
    for i := range out {
        out[i] = id
    }
    return out
}
```

- [ ] **Step 2: Run the tests to confirm they fail**

```bash
go test ./internal/engine/cards -run TestGardens -v
```

Expected: FAIL with `undefined: Gardens` (and `makeCards` may compile fine — that's OK).

- [ ] **Step 3: Implement Gardens**

Create `internal/engine/cards/kingdom_gardens.go`:

```go
package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

// Gardens is a Victory card whose value scales with deck size:
// "Worth 1 Victory Point per 10 cards you have (round down)."
//
// Each Gardens copy in any zone independently contributes
// floor(n/10) VP, where n counts every card the player owns:
// Hand + Deck + Discard + InPlay + SetAside. Cards in trash do
// not count (trash is not part of PlayerState).
//
// SetAside is included defensively. By end-of-game cleanup it's
// empty for the active player; counting it here makes the function
// correct for any caller (debug snapshots, mid-game previews) with
// no separate "active vs other player" code path.
var Gardens = &engine.Card{
    ID:    "gardens",
    Name:  "Gardens",
    Cost:  4,
    Types: []engine.CardType{engine.TypeVictory},
    VictoryPoints: func(p engine.PlayerState) int {
        n := len(p.Hand) + len(p.Deck) + len(p.Discard) + len(p.InPlay) + len(p.SetAside)
        return n / 10
    },
}

func init() {
    DefaultRegistry.Register(Gardens)
}
```

- [ ] **Step 4: Run the tests to confirm they pass**

```bash
go test ./internal/engine/cards -run TestGardens -v
```

Expected: PASS for all six subtests.

- [ ] **Step 5: Write the failing scoring test**

Append to `internal/engine/scoring_test.go`:

```go
func TestComputeScore_Gardens_FloorsByTen(t *testing.T) {
    lookup := func(id CardID) (*Card, bool) {
        switch id {
        case "gardens":
            return &Card{
                VictoryPoints: func(p PlayerState) int {
                    n := len(p.Hand) + len(p.Deck) + len(p.Discard) + len(p.InPlay) + len(p.SetAside)
                    return n / 10
                },
            }, true
        }
        return &Card{}, true
    }
    // 18 cards across all zones; the single Gardens contributes
    // floor(18/10) = 1.
    ps := PlayerState{
        Hand:    []CardID{"gardens", "copper", "copper", "copper"},
        Deck:    []CardID{"copper", "copper", "copper", "copper"},
        Discard: []CardID{"copper", "copper", "copper", "copper"},
        InPlay:  []CardID{"copper", "copper", "copper"},
        SetAside: []CardID{"copper", "copper", "copper"},
    }
    require.Equal(t, 1, ComputeScore(ps, lookup))
}
```

- [ ] **Step 6: Run the test to confirm it passes**

```bash
go test ./internal/engine -run TestComputeScore_Gardens -v
```

Expected: PASS — `ComputeScore` already iterates the right zones (it doesn't need a code change for this test to pass; the test simply verifies the behavior we depend on).

- [ ] **Step 7: Run full engine + cards short tests**

```bash
go test ./internal/engine ./internal/engine/cards -short
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/engine/cards/kingdom_gardens.go \
        internal/engine/cards/kingdom_gardens_test.go \
        internal/engine/scoring_test.go
git commit -m "$(cat <<'EOF'
feat(cards): add Gardens — VP scales with deck size

Victory-only kingdom card; VictoryPoints returns floor(n/10) where n
counts Hand+Deck+Discard+InPlay+SetAside. Registered via init() into
DefaultRegistry; auto-eligible for random kingdoms now that IsKingdom
accepts non-basic Victory cards.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Merchant — `kingdom_merchant.go` + Silver trigger edit + TR-of-Merchant regression test

Adds the Merchant card and amends Silver's `OnPlay` to read the per-turn trigger fields. Both halves ship together because each is meaningless alone.

**Files:**
- Create: `internal/engine/cards/kingdom_merchant.go` — Merchant card definition.
- Create: `internal/engine/cards/kingdom_merchant_test.go` — per-card unit tests including TR-of-Merchant regression.
- Modify: `internal/engine/cards/basics_treasure.go` — Silver.OnPlay reads the trigger fields.
- Modify: `internal/engine/cards/basics_treasure_test.go` — extend existing tests.

- [ ] **Step 1: Write failing tests for Merchant**

Create `internal/engine/cards/kingdom_merchant_test.go`:

```go
package cards

import (
    "testing"

    "github.com/nutthawit-l/dominion-grpc/internal/engine"
    "github.com/stretchr/testify/require"
)

func TestMerchant_Metadata(t *testing.T) {
    require.Equal(t, engine.CardID("merchant"), Merchant.ID)
    require.Equal(t, "Merchant", Merchant.Name)
    require.Equal(t, 3, Merchant.Cost)
    require.True(t, Merchant.HasType(engine.TypeAction))
    _, ok := DefaultRegistry.Lookup("merchant")
    require.True(t, ok, "Merchant must be registered in DefaultRegistry")
}

func TestMerchant_OnPlay_DrawsAndAddsAction(t *testing.T) {
    gs := newTestStateForCardsWithRNG(2, 1)
    gs.Players[0].Deck = []engine.CardID{"copper", "estate", "silver"}

    Merchant.OnPlay(gs, 0)

    require.Len(t, gs.Players[0].Hand, 1, "Merchant draws 1 card")
    require.Equal(t, 1, gs.Players[0].Actions, "Merchant adds 1 Action")
}

func TestMerchant_OnPlay_IncrementsBonusCharges(t *testing.T) {
    gs := newTestStateForCardsWithRNG(2, 1)
    gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper"}

    Merchant.OnPlay(gs, 0)
    require.Equal(t, 1, gs.Players[0].MerchantBonusCharges,
        "first Merchant play → 1 charge")

    Merchant.OnPlay(gs, 0)
    require.Equal(t, 2, gs.Players[0].MerchantBonusCharges,
        "second Merchant play → 2 charges")
}

func TestMerchant_PlayedThenSilver_AddsOneCoinBonus(t *testing.T) {
    gs := newActionPhaseState()
    gs.Players[0].Hand = []engine.CardID{"merchant", "silver"}
    gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper"}

    // Play Merchant in Action phase.
    _, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "merchant"}, testLookup)
    require.NoError(t, err)

    // End Action phase; enter Buy.
    _, _, err = engine.Apply(gs, engine.EndPhase{PlayerIdx: 0}, testLookup)
    require.NoError(t, err)
    require.Equal(t, engine.PhaseBuy, gs.Phase)

    // Play Silver — should pay 2 (Silver) + 1 (Merchant bonus).
    _, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
    require.NoError(t, err)

    require.Equal(t, 3, gs.Players[0].Coins,
        "Silver (+2) + Merchant bonus (+1) = 3 coins")
    require.True(t, gs.Players[0].FirstSilverPlayedThisTurn,
        "FirstSilverPlayedThisTurn must be true after first Silver play")
}

func TestMerchant_TwoMerchants_ThenSilver_AddsTwoCoinBonus(t *testing.T) {
    gs := newActionPhaseState()
    gs.Players[0].Actions = 2
    gs.Players[0].Hand = []engine.CardID{"merchant", "merchant", "silver"}
    gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper"}

    _, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "merchant"}, testLookup)
    require.NoError(t, err)
    _, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "merchant"}, testLookup)
    require.NoError(t, err)
    _, _, err = engine.Apply(gs, engine.EndPhase{PlayerIdx: 0}, testLookup)
    require.NoError(t, err)
    _, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
    require.NoError(t, err)

    require.Equal(t, 4, gs.Players[0].Coins,
        "Silver (+2) + 2× Merchant bonus (+2) = 4 coins")
}

func TestMerchant_OnlyFirstSilverTriggersBonus(t *testing.T) {
    gs := newActionPhaseState()
    gs.Players[0].Hand = []engine.CardID{"merchant", "silver", "silver"}
    gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper"}

    _, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "merchant"}, testLookup)
    require.NoError(t, err)
    _, _, err = engine.Apply(gs, engine.EndPhase{PlayerIdx: 0}, testLookup)
    require.NoError(t, err)
    _, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
    require.NoError(t, err)
    require.Equal(t, 3, gs.Players[0].Coins, "first Silver: 2+1=3")

    _, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
    require.NoError(t, err)
    require.Equal(t, 5, gs.Players[0].Coins, "second Silver: +2 only (no bonus)")
}

func TestMerchant_SilverThenMerchant_NoBonusThisTurn(t *testing.T) {
    gs := newActionPhaseState()
    // Switch directly to Buy phase to play Silver first; Merchant lives
    // in Hand but cannot be played in Buy phase. We skip Merchant
    // entirely this turn and assert no bonus.
    gs.Phase = engine.PhaseBuy
    gs.Players[0].Hand = []engine.CardID{"silver", "silver"}

    _, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
    require.NoError(t, err)
    require.Equal(t, 2, gs.Players[0].Coins, "Silver alone: +2 (no Merchant)")
    require.True(t, gs.Players[0].FirstSilverPlayedThisTurn)

    // Now hand-poke MerchantBonusCharges to simulate "Merchant played
    // after first Silver in same turn." Per Dominion rules, this should
    // NOT retroactively award a bonus on the next Silver — the flag is
    // already set.
    gs.Players[0].MerchantBonusCharges = 1

    _, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
    require.NoError(t, err)
    require.Equal(t, 4, gs.Players[0].Coins,
        "second Silver: +2, NO bonus because flag already set")
}

func TestMerchant_ThroneRoom_DoublesBonusOnFirstSilver(t *testing.T) {
    // Regression coverage for §3.7 of the Tier 5 spec — TR(Merchant) +
    // Silver gives +$2 bonus from doubled charges. Replaces the JSON
    // replay fixture the spec originally proposed.
    gs := newActionPhaseState()
    gs.Players[0].Hand = []engine.CardID{"throne_room", "merchant", "silver"}
    gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper", "copper"}

    // Play Throne Room.
    _, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "throne_room"}, testLookup)
    require.NoError(t, err)
    require.NotNil(t, gs.PendingDecision, "TR prompts ChooseActionFromHand")

    // Resolve TR's choose-action prompt with Merchant.
    _, _, err = engine.Apply(gs, engine.ResolveDecision{
        PlayerIdx:  0,
        DecisionID: gs.PendingDecision.ID,
        Answer:     engine.CardChoiceAnswer{Card: "merchant"},
    }, testLookup)
    require.NoError(t, err)

    require.Equal(t, 2, gs.Players[0].MerchantBonusCharges,
        "TR(Merchant) accumulates 2 charges")

    // End Action phase; enter Buy.
    _, _, err = engine.Apply(gs, engine.EndPhase{PlayerIdx: 0}, testLookup)
    require.NoError(t, err)

    // Play Silver — should pay 2 (Silver) + 2 (TR-doubled Merchant bonus).
    _, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
    require.NoError(t, err)

    require.Equal(t, 4, gs.Players[0].Coins,
        "TR(Merchant) + Silver: 2 + 2 = 4 coins")
    require.True(t, gs.Players[0].FirstSilverPlayedThisTurn)
}
```

- [ ] **Step 2: Run the tests to confirm they fail**

```bash
go test ./internal/engine/cards -run TestMerchant -v
```

Expected: FAIL with `undefined: Merchant`.

- [ ] **Step 3: Implement Merchant**

Create `internal/engine/cards/kingdom_merchant.go`:

```go
package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

// Merchant is a $3 Action: "+1 Card, +1 Action. The first time you
// play a Silver this turn, +$1."
//
// The "first Silver this turn" trigger is split across two cards:
//   - Merchant.OnPlay increments PlayerState.MerchantBonusCharges.
//   - Silver.OnPlay reads charges + FirstSilverPlayedThisTurn,
//     pays the bonus once per turn.
// cleanupAndEndTurn resets both fields. See basics_treasure.go.
var Merchant = &engine.Card{
    ID:    "merchant",
    Name:  "Merchant",
    Cost:  3,
    Types: []engine.CardType{engine.TypeAction},
    OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
        events := engine.DrawCards(gs, px, 1)
        events = append(events, engine.AddActions(gs, px, 1)...)
        gs.Players[px].MerchantBonusCharges++
        return events
    },
}

func init() {
    DefaultRegistry.Register(Merchant)
}
```

- [ ] **Step 4: Run only the metadata + basic OnPlay tests to confirm Merchant compiles**

```bash
go test ./internal/engine/cards -run "TestMerchant_Metadata|TestMerchant_OnPlay_DrawsAndAddsAction|TestMerchant_OnPlay_IncrementsBonusCharges" -v
```

Expected: PASS for those three. The other Merchant tests (which exercise Silver) will still FAIL because Silver hasn't been edited yet — that's expected at this checkpoint.

- [ ] **Step 5: Edit Silver to read the trigger fields**

Edit `internal/engine/cards/basics_treasure.go`. Replace the existing `Silver` definition:

```go
// Silver is a $3 Treasure: "+$2." It also delivers Merchant's bonus —
// the first time Silver is played in a turn, any Merchants played
// earlier this turn (counted in MerchantBonusCharges) each contribute
// +$1. The bookkeeping fields live on PlayerState and are owned by
// Tier 5; cleanupAndEndTurn resets them.
var Silver = &engine.Card{
    ID:    "silver",
    Name:  "Silver",
    Cost:  3,
    Types: []engine.CardType{engine.TypeTreasure},
    OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
        events := engine.AddCoins(gs, px, 2)
        ps := &gs.Players[px]
        if !ps.FirstSilverPlayedThisTurn {
            ps.FirstSilverPlayedThisTurn = true
            if ps.MerchantBonusCharges > 0 {
                events = append(events, engine.AddCoins(gs, px, ps.MerchantBonusCharges)...)
            }
        }
        return events
    },
}
```

Leave `Copper`, `Gold`, and the `init()` block unchanged.

- [ ] **Step 6: Run all the Merchant tests to confirm they pass**

```bash
go test ./internal/engine/cards -run TestMerchant -v
```

Expected: PASS for all eight subtests, including `TestMerchant_ThroneRoom_DoublesBonusOnFirstSilver`.

- [ ] **Step 7: Extend Silver's basics tests**

Open `internal/engine/cards/basics_treasure_test.go`. Append three new tests at the end of the file:

```go
func TestSilver_OnPlay_NoMerchant_NoBonus(t *testing.T) {
    gs := newTestStateForCards(1)
    Silver.OnPlay(gs, 0)
    require.Equal(t, 2, gs.Players[0].Coins,
        "without any MerchantBonusCharges, Silver pays only 2")
    require.True(t, gs.Players[0].FirstSilverPlayedThisTurn,
        "Silver must flip the per-turn first-Silver flag")
}

func TestSilver_OnPlay_FirstTime_DrainsCharges(t *testing.T) {
    gs := newTestStateForCards(1)
    gs.Players[0].MerchantBonusCharges = 2
    Silver.OnPlay(gs, 0)
    require.Equal(t, 4, gs.Players[0].Coins,
        "first Silver pays 2 + 2 (Merchant bonus) = 4")
    require.True(t, gs.Players[0].FirstSilverPlayedThisTurn)
    // Charges remain stranded on PlayerState; only flag prevents re-fire.
}

func TestSilver_OnPlay_SecondTimeThisTurn_NoBonus(t *testing.T) {
    gs := newTestStateForCards(1)
    gs.Players[0].FirstSilverPlayedThisTurn = true
    gs.Players[0].MerchantBonusCharges = 5
    Silver.OnPlay(gs, 0)
    require.Equal(t, 2, gs.Players[0].Coins,
        "second Silver this turn must pay only +2, no bonus")
    require.True(t, gs.Players[0].FirstSilverPlayedThisTurn,
        "flag stays true (idempotent)")
}
```

The pre-existing `TestSilver_OnPlay_AddsTwoCoins` test (asserting `Coins == 2` and `len(events) == 1`) is still correct — `newTestStateForCards` produces a state with `MerchantBonusCharges == 0` and `FirstSilverPlayedThisTurn == false`, so Silver's OnPlay flips the flag but adds no bonus → 1 event, 2 coins. **Verify this stays green** in the next step.

- [ ] **Step 8: Run all Silver tests to confirm they pass**

```bash
go test ./internal/engine/cards -run TestSilver -v
```

Expected: PASS for all four subtests (`TestSilver_OnPlay_AddsTwoCoins`, plus the three new ones).

- [ ] **Step 9: Run full engine + cards short tests**

```bash
go test ./internal/engine ./internal/engine/cards -short
```

Expected: PASS — including the existing Throne Room tests, which must still pass because TR-of-Smithy / TR-of-Cellar / TR-TR-Witch don't touch the new Merchant fields.

- [ ] **Step 10: Run lint**

```bash
make lint
```

Expected: clean.

- [ ] **Step 11: Commit**

```bash
git add internal/engine/cards/kingdom_merchant.go \
        internal/engine/cards/kingdom_merchant_test.go \
        internal/engine/cards/basics_treasure.go \
        internal/engine/cards/basics_treasure_test.go
git commit -m "$(cat <<'EOF'
feat(cards): add Merchant + Silver first-play bonus trigger

Merchant: +1 Card, +1 Action, ++MerchantBonusCharges on play.
Silver: on first play of the turn, drain MerchantBonusCharges and pay
that many bonus coins; mark FirstSilverPlayedThisTurn. Includes
TR-of-Merchant regression coverage (in lieu of a JSON replay fixture
— the engine package's replay test only resolves basics).

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: MerchantBM + GardensBM strategies + smoke sweeps + `bot.StrategyByName`

Adds the two bot strategies, their unit tests, the 50-game smoke sweeps, the new `bot.StrategyByName` factory, and refactors `cmd/bot/main.go` to delegate to it.

**Files:**
- Create: `internal/bot/merchant_bm.go` — MerchantBM strategy.
- Create: `internal/bot/merchant_bm_test.go` — MerchantBM unit tests.
- Create: `internal/bot/gardens_bm.go` — GardensBM strategy.
- Create: `internal/bot/gardens_bm_test.go` — GardensBM unit tests.
- Create: `internal/bot/strategy_byname.go` — `StrategyByName` factory.
- Create: `internal/bot/strategy_byname_test.go` — round-trip tests for the factory.
- Modify: `internal/bot/strategy.go` — extend `isAction()` and `cardCost()` for `merchant` and `gardens`.
- Modify: `internal/bot/integration_test.go` — append `TestBotVsBot_MerchantBM_CompletesNormally` and `TestBotVsBot_GardensBM_CompletesNormally`.
- Modify: `cmd/bot/main.go` — replace `selectStrategy` with `bot.StrategyByName`; extend `-strategy` help text.

- [ ] **Step 1: Write the failing MerchantBM unit tests**

Create `internal/bot/merchant_bm_test.go`:

```go
package bot

import (
    "testing"

    pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
    "github.com/stretchr/testify/require"
)

func TestMerchantBM_Name(t *testing.T) {
    require.Equal(t, "merchant_bm", NewMerchantBM().Name())
}

func TestMerchantBM_PlaysMerchantWhenInHand(t *testing.T) {
    cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1,
        []string{"merchant", "copper"})
    cs.MyPlayer().Actions = 1
    a := NewMerchantBM().PickAction(cs)
    require.Equal(t, "merchant", a.GetPlayCard().CardId)
}

func TestMerchantBM_NoMerchantInHand_EndsActionPhase(t *testing.T) {
    cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1,
        []string{"copper", "estate"})
    cs.MyPlayer().Actions = 1
    a := NewMerchantBM().PickAction(cs)
    require.NotNil(t, a.GetEndPhase(),
        "no Merchant in hand → end action phase")
}

func TestMerchantBM_BuysMerchantAt3_FirstCopy(t *testing.T) {
    cs := csWithHand(0, pb.Phase_PHASE_BUY, 3, 1, nil)
    cs.Snapshot = snapshotWithSupply(map[string]int{
        "province": 8, "gold": 30, "silver": 40, "merchant": 10,
    })
    cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 3, Buys: 1}}
    a := NewMerchantBM().PickAction(cs)
    require.Equal(t, "merchant", a.GetBuyCard().CardId,
        "at coins=3 with no Merchants owned, prefer Merchant over Silver")
}

func TestMerchantBM_StopsBuyingAfterTwo(t *testing.T) {
    s := NewMerchantBM()
    s.merchants = 2
    cs := csWithHand(0, pb.Phase_PHASE_BUY, 3, 1, nil)
    cs.Snapshot = snapshotWithSupply(map[string]int{
        "province": 8, "gold": 30, "silver": 40, "merchant": 10,
    })
    cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 3, Buys: 1}}
    a := s.PickAction(cs)
    require.Equal(t, "silver", a.GetBuyCard().CardId,
        "with 2 Merchants owned, fall through to Silver")
}

func TestMerchantBM_BuysProvinceAtEight(t *testing.T) {
    cs := csWithHand(0, pb.Phase_PHASE_BUY, 8, 1, nil)
    cs.Snapshot = snapshotWithSupply(map[string]int{
        "province": 8, "gold": 30, "silver": 40, "merchant": 10,
    })
    cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 8, Buys: 1}}
    a := NewMerchantBM().PickAction(cs)
    require.Equal(t, "province", a.GetBuyCard().CardId)
}
```

- [ ] **Step 2: Write the failing GardensBM unit tests**

Create `internal/bot/gardens_bm_test.go`:

```go
package bot

import (
    "testing"

    pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
    "github.com/stretchr/testify/require"
)

func TestGardensBM_Name(t *testing.T) {
    require.Equal(t, "gardens_bm", NewGardensBM().Name())
}

func TestGardensBM_ActionPhase_AlwaysEndsPhase(t *testing.T) {
    cs := csWithHand(0, pb.Phase_PHASE_ACTION, 0, 1,
        []string{"copper", "estate"})
    a := NewGardensBM().PickAction(cs)
    require.NotNil(t, a.GetEndPhase(),
        "Gardens is Victory-only — nothing to play")
}

func TestGardensBM_BuysGardensAt4_FirstCopy(t *testing.T) {
    cs := csWithHand(0, pb.Phase_PHASE_BUY, 4, 1, nil)
    cs.Snapshot = snapshotWithSupply(map[string]int{
        "province": 8, "gold": 30, "silver": 40, "gardens": 8, "copper": 40,
    })
    cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 4, Buys: 1}}
    a := NewGardensBM().PickAction(cs)
    require.Equal(t, "gardens", a.GetBuyCard().CardId)
}

func TestGardensBM_StopsBuyingGardensAfterFour(t *testing.T) {
    s := NewGardensBM()
    s.gardens = 4
    cs := csWithHand(0, pb.Phase_PHASE_BUY, 4, 1, nil)
    cs.Snapshot = snapshotWithSupply(map[string]int{
        "province": 8, "gold": 30, "silver": 40, "gardens": 8, "copper": 40,
    })
    cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 4, Buys: 1}}
    a := s.PickAction(cs)
    // 4-cost slot empty after Gardens cap; falls through to Silver at 3+.
    require.Equal(t, "silver", a.GetBuyCard().CardId)
}

func TestGardensBM_BuysProvinceAtEight(t *testing.T) {
    cs := csWithHand(0, pb.Phase_PHASE_BUY, 8, 1, nil)
    cs.Snapshot = snapshotWithSupply(map[string]int{
        "province": 8, "gold": 30, "silver": 40, "gardens": 8, "copper": 40,
    })
    cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 8, Buys: 1}}
    a := NewGardensBM().PickAction(cs)
    require.Equal(t, "province", a.GetBuyCard().CardId)
}

func TestGardensBM_BulksWithCopper(t *testing.T) {
    // At 0 coins with buys remaining and Copper in supply, buy Copper.
    // Gardens-rush wants a fat deck, so deliberate bulking is on-strategy.
    cs := csWithHand(0, pb.Phase_PHASE_BUY, 0, 1, nil)
    cs.Snapshot = snapshotWithSupply(map[string]int{
        "province": 8, "gold": 30, "silver": 40, "gardens": 8, "copper": 40,
    })
    cs.Snapshot.Players = []*pb.PlayerView{{PlayerIdx: 0, Coins: 0, Buys: 1}}
    a := NewGardensBM().PickAction(cs)
    require.Equal(t, "copper", a.GetBuyCard().CardId)
}
```

- [ ] **Step 3: Run the strategy tests to confirm they fail**

```bash
go test ./internal/bot -run "TestMerchantBM|TestGardensBM" -v
```

Expected: FAIL with `undefined: NewMerchantBM` / `undefined: NewGardensBM`.

- [ ] **Step 4: Implement MerchantBM**

Create `internal/bot/merchant_bm.go`:

```go
package bot

import (
    pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// MerchantBM is Big Money plus Merchant. Buys up to 2 Merchants at $3
// (counter incremented on each buy), then reverts to Big Money. Plays
// Merchant whenever in hand and Actions remain. Defaults to safe
// refusal on every prompt — Merchant itself prompts no decisions.
type MerchantBM struct {
    merchants int
}

// NewMerchantBM constructs a MerchantBM strategy.
func NewMerchantBM() *MerchantBM { return &MerchantBM{} }

// Name implements Strategy.
func (m *MerchantBM) Name() string { return "merchant_bm" }

// PickAction implements Strategy.
func (m *MerchantBM) PickAction(cs *ClientState) *pb.Action {
    me := cs.MyPlayer()
    if me == nil {
        return endPhase(cs.Me)
    }
    switch cs.Phase {
    case pb.Phase_PHASE_ACTION:
        if me.Actions >= 1 && handContains(me.Hand, "merchant") {
            return playCard(cs.Me, "merchant")
        }
        return endPhase(cs.Me)

    case pb.Phase_PHASE_BUY:
        for _, c := range me.Hand {
            if isTreasure(c) {
                return playCard(cs.Me, c)
            }
        }
        if me.Buys <= 0 {
            return endPhase(cs.Me)
        }
        switch {
        case me.Coins >= 8 && supplyCount(cs.Snapshot, "province") > 0:
            return buyCard(cs.Me, "province")
        case me.Coins >= 6 && supplyCount(cs.Snapshot, "gold") > 0:
            return buyCard(cs.Me, "gold")
        case me.Coins >= 3 && m.merchants < 2 && supplyCount(cs.Snapshot, "merchant") > 0:
            m.merchants++
            return buyCard(cs.Me, "merchant")
        case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
            return buyCard(cs.Me, "silver")
        }
        return endPhase(cs.Me)
    }
    return endPhase(cs.Me)
}

// Resolve implements Strategy.
func (m *MerchantBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
    return safeRefusal(cs, d)
}
```

- [ ] **Step 5: Implement GardensBM**

Create `internal/bot/gardens_bm.go`:

```go
package bot

import (
    pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// GardensBM is a deliberate Gardens-rush strategy: bulks the deck with
// Coppers when nothing better is available, buys Gardens at $4 (up to
// 4 copies), and otherwise follows BigMoney's economy ladder. Action
// phase is always a no-op since Gardens is Victory-only.
//
// Gardens-rush in 2-player Base is a known *weak* strategy (Provinces
// drain too fast). The smoke sweep proves correctness, not strategy
// quality — `wins ≥ 1` over 50 games is the bar.
type GardensBM struct {
    gardens int
}

// NewGardensBM constructs a GardensBM strategy.
func NewGardensBM() *GardensBM { return &GardensBM{} }

// Name implements Strategy.
func (g *GardensBM) Name() string { return "gardens_bm" }

// PickAction implements Strategy.
func (g *GardensBM) PickAction(cs *ClientState) *pb.Action {
    me := cs.MyPlayer()
    if me == nil {
        return endPhase(cs.Me)
    }
    switch cs.Phase {
    case pb.Phase_PHASE_ACTION:
        return endPhase(cs.Me) // Gardens is Victory-only.

    case pb.Phase_PHASE_BUY:
        for _, c := range me.Hand {
            if isTreasure(c) {
                return playCard(cs.Me, c)
            }
        }
        if me.Buys <= 0 {
            return endPhase(cs.Me)
        }
        switch {
        case me.Coins >= 8 && supplyCount(cs.Snapshot, "province") > 0:
            return buyCard(cs.Me, "province")
        case me.Coins >= 6 && supplyCount(cs.Snapshot, "gold") > 0:
            return buyCard(cs.Me, "gold")
        case me.Coins >= 4 && g.gardens < 4 && supplyCount(cs.Snapshot, "gardens") > 0:
            g.gardens++
            return buyCard(cs.Me, "gardens")
        case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
            return buyCard(cs.Me, "silver")
        case me.Coins >= 0 && supplyCount(cs.Snapshot, "copper") > 0:
            return buyCard(cs.Me, "copper")
        }
        return endPhase(cs.Me)
    }
    return endPhase(cs.Me)
}

// Resolve implements Strategy.
func (g *GardensBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
    return safeRefusal(cs, d)
}
```

- [ ] **Step 6: Run the strategy tests to confirm they pass**

```bash
go test ./internal/bot -run "TestMerchantBM|TestGardensBM" -v
```

Expected: PASS for all sub-tests.

- [ ] **Step 7: Extend `isAction()` and `cardCost()` for the new cards**

Edit `internal/bot/strategy.go`. In the `isAction` function, extend the case list to include `merchant`:

```go
func isAction(id string) bool {
    switch id {
    case "smithy", "village", "festival", "laboratory", "market",
        "council_room", "moat", "harbinger", "vassal", "workshop",
        "moneylender", "poacher", "remodel", "mine", "artisan",
        "cellar", "chapel", "witch", "militia", "bureaucrat",
        "bandit", "throne_room", "library", "sentry", "merchant":
        return true
    }
    return false
}
```

In the `cardCost` function, add cases for `merchant` (3) and `gardens` (4). The simplest edit: extend the existing `silver, cellar, chapel, village` line to also include `merchant`, and the existing `militia, bureaucrat, ...` line to also include `gardens`. Final version:

```go
func cardCost(id string) int {
    switch id {
    case "copper", "curse":
        return 0
    case "estate", "moat":
        return 2
    case "silver", "cellar", "chapel", "village", "merchant":
        return 3
    case "harbinger", "vassal", "workshop":
        return 3
    case "militia", "bureaucrat", "moneylender", "poacher", "remodel", "smithy", "throne_room", "gardens":
        return 4
    case "mine", "witch", "bandit", "laboratory", "market", "festival", "library", "sentry":
        return 5
    case "gold", "artisan", "council_room":
        return 6
    case "duchy":
        return 5
    case "province":
        return 8
    }
    return 0
}
```

- [ ] **Step 8: Write failing tests for `bot.StrategyByName`**

Create `internal/bot/strategy_byname_test.go`:

```go
package bot

import (
    "testing"

    "github.com/stretchr/testify/require"
)

func TestStrategyByName_AllRegisteredNames(t *testing.T) {
    names := []string{
        "bigmoney", "smithy_bm", "chapel_bm", "remodel_bm",
        "witch_bm", "militia_bm", "throneroom_bm", "library_bm",
        "sentry_bm", "merchant_bm", "gardens_bm",
    }
    for _, name := range names {
        s, err := StrategyByName(name)
        require.NoErrorf(t, err, "StrategyByName(%q)", name)
        require.NotNilf(t, s, "StrategyByName(%q) returned nil strategy", name)
        require.Equalf(t, name, s.Name(),
            "round-trip mismatch for %q", name)
    }
}

func TestStrategyByName_UnknownReturnsError(t *testing.T) {
    _, err := StrategyByName("does_not_exist")
    require.Error(t, err)
}
```

- [ ] **Step 9: Run the factory tests to confirm they fail**

```bash
go test ./internal/bot -run TestStrategyByName -v
```

Expected: FAIL with `undefined: StrategyByName`.

- [ ] **Step 10: Implement `StrategyByName`**

Create `internal/bot/strategy_byname.go`:

```go
package bot

import "fmt"

// StrategyByName returns a fresh Strategy by its short name. Used by
// cmd/bot/main.go and by integration tests that pick strategies from
// a parameterized list. Adding a new strategy means adding a case
// here AND extending the closeout sweep's strategies list.
func StrategyByName(name string) (Strategy, error) {
    switch name {
    case "bigmoney":
        return BigMoney{}, nil
    case "smithy_bm":
        return SmithyBM{}, nil
    case "chapel_bm":
        return NewChapelBM(), nil
    case "remodel_bm":
        return NewRemodelBM(), nil
    case "witch_bm":
        return NewWitchBM(), nil
    case "militia_bm":
        return NewMilitiaBM(), nil
    case "throneroom_bm":
        return NewThroneRoomBM(), nil
    case "library_bm":
        return NewLibraryBM(), nil
    case "sentry_bm":
        return NewSentryBM(), nil
    case "merchant_bm":
        return NewMerchantBM(), nil
    case "gardens_bm":
        return NewGardensBM(), nil
    }
    return nil, fmt.Errorf("unknown strategy %q", name)
}
```

- [ ] **Step 11: Run the factory tests to confirm they pass**

```bash
go test ./internal/bot -run TestStrategyByName -v
```

Expected: PASS for both subtests.

- [ ] **Step 12: Refactor `cmd/bot/main.go` to use `bot.StrategyByName`**

Edit `cmd/bot/main.go`. Replace the `selectStrategy` function with a one-line delegation, AND update the `-strategy` flag's help text to include the two new names. Final form of the relevant region:

```go
strategyName := flag.String("strategy", "bigmoney",
    "strategy to use: bigmoney | smithy_bm | chapel_bm | remodel_bm | witch_bm | militia_bm | throneroom_bm | library_bm | sentry_bm | merchant_bm | gardens_bm")
```

Replace the entire `selectStrategy` function body:

```go
func selectStrategy(name string) (bot.Strategy, error) {
    return bot.StrategyByName(name)
}
```

- [ ] **Step 13: Verify `cmd/bot` still builds**

```bash
go build ./cmd/bot
```

Expected: exit 0.

- [ ] **Step 14: Run all bot package short tests**

```bash
go test ./internal/bot -short -v
```

Expected: PASS for every test (existing + new MerchantBM + GardensBM + StrategyByName).

- [ ] **Step 15: Add the smoke-sweep tests**

Append to `internal/bot/integration_test.go` (after `TestBotVsBot_SentryBM_CompletesNormally` and before `finalWinners`):

```go
func TestBotVsBot_MerchantBM_CompletesNormally(t *testing.T) {
    if testing.Short() {
        t.Skip("integration sweep — skipped under -short")
    }

    const games = 50

    srv := newTestServer(t)

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    wins := 0
    for seed := int64(0); seed < games; seed++ {
        merchSeat := int(seed % 2)
        bmSeat := 1 - merchSeat

        names := make([]string, 2)
        names[merchSeat] = "merchant_bm"
        names[bmSeat] = "bigmoney"

        a := bot.NewClient(srv.URL)
        b := bot.NewClient(srv.URL)
        game, err := a.CreateGame(ctx, names, seed, []string{"merchant"})
        require.NoError(t, err)

        strategies := map[int]bot.Strategy{
            merchSeat: bot.NewMerchantBM(),
            bmSeat:    bot.BigMoney{},
        }

        grp, gctx := errgroup.WithContext(ctx)
        grp.Go(func() error { return bot.Run(gctx, a, game.GameId, 0, strategies[0]) })
        grp.Go(func() error { return bot.Run(gctx, b, game.GameId, 1, strategies[1]) })
        require.NoError(t, grp.Wait(), "seed=%d", seed)

        winners := finalWinners(t, srv, game.GameId)
        if len(winners) == 1 && winners[0] == merchSeat {
            wins++
        }
    }

    require.GreaterOrEqualf(t, wins, 1,
        "MerchantBM lost every one of %d games — indicates engine bug, not strategy weakness", games)
}

func TestBotVsBot_GardensBM_CompletesNormally(t *testing.T) {
    if testing.Short() {
        t.Skip("integration sweep — skipped under -short")
    }

    const games = 50

    srv := newTestServer(t)

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    wins := 0
    for seed := int64(0); seed < games; seed++ {
        gardensSeat := int(seed % 2)
        bmSeat := 1 - gardensSeat

        names := make([]string, 2)
        names[gardensSeat] = "gardens_bm"
        names[bmSeat] = "bigmoney"

        a := bot.NewClient(srv.URL)
        b := bot.NewClient(srv.URL)
        game, err := a.CreateGame(ctx, names, seed, []string{"gardens"})
        require.NoError(t, err)

        strategies := map[int]bot.Strategy{
            gardensSeat: bot.NewGardensBM(),
            bmSeat:      bot.BigMoney{},
        }

        grp, gctx := errgroup.WithContext(ctx)
        grp.Go(func() error { return bot.Run(gctx, a, game.GameId, 0, strategies[0]) })
        grp.Go(func() error { return bot.Run(gctx, b, game.GameId, 1, strategies[1]) })
        require.NoError(t, grp.Wait(), "seed=%d", seed)

        winners := finalWinners(t, srv, game.GameId)
        if len(winners) == 1 && winners[0] == gardensSeat {
            wins++
        }
    }

    require.GreaterOrEqualf(t, wins, 1,
        "GardensBM lost every one of %d games — indicates engine bug, not strategy weakness", games)
}
```

- [ ] **Step 16: Run the smoke sweeps**

```bash
go test ./internal/bot -run "TestBotVsBot_MerchantBM_CompletesNormally|TestBotVsBot_GardensBM_CompletesNormally" -v
```

Expected: PASS — both sweeps complete in well under 2 minutes; both report `wins ≥ 1`. If either fails to finish a game, that's an engine bug — investigate (likely an interaction between Gardens or Merchant and the existing game-flow). If `wins == 0` for one of them, treat as an engine bug.

- [ ] **Step 17: Run `make test` to verify nothing else regressed**

```bash
make test
```

Expected: every previously-green test stays green; the two new smoke sweeps PASS. This is the one place we run the full sweeps in this task to catch interactions with other bots' tests.

- [ ] **Step 18: Run lint**

```bash
make lint
```

Expected: clean.

- [ ] **Step 19: Commit**

```bash
git add internal/bot/merchant_bm.go internal/bot/merchant_bm_test.go \
        internal/bot/gardens_bm.go internal/bot/gardens_bm_test.go \
        internal/bot/strategy_byname.go internal/bot/strategy_byname_test.go \
        internal/bot/strategy.go \
        internal/bot/integration_test.go \
        cmd/bot/main.go
git commit -m "$(cat <<'EOF'
feat(bot): add MerchantBM + GardensBM + StrategyByName factory

Two smoke-only strategies (50-game sweeps; wins>=1 acceptance bar) plus
a bot.StrategyByName factory that consolidates strategy lookup. cmd/bot
now delegates to it. isAction/cardCost extended with merchant + gardens.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Phase-1a closeout — full-set 1000-game sweep + README update

The final unit. Adds `TestBotVsBot_FullBaseSet_RandomKingdoms` driven by random kingdoms and random strategy pairs over the full registry, then updates the README to declare Phase 1a complete.

**Files:**
- Create: `internal/bot/integration_full_set_test.go` — the 1000-game sweep + helpers.
- Modify: `README.md` — declare Phase 1a complete; document the two new bots.

- [ ] **Step 1: Write the failing closeout sweep test**

Create `internal/bot/integration_full_set_test.go`:

```go
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

// allKingdomCardIDs reads every kingdom card from the cards
// DefaultRegistry. Filters via Card.IsKingdom(); auto-includes Gardens
// now that the predicate accepts non-basic Victory cards.
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
// Returns a slice with no duplicates.
func pickRandomKingdom(seed int64, pool []string, size int) []string {
    r := rand.New(rand.NewSource(seed))
    cp := append([]string(nil), pool...)
    r.Shuffle(len(cp), func(i, j int) { cp[i], cp[j] = cp[j], cp[i] })
    if size > len(cp) {
        size = len(cp)
    }
    return cp[:size]
}

// pickRandomStrategyPair returns two strategy names, possibly the same,
// chosen deterministically by seed.
func pickRandomStrategyPair(seed int64, names []string) (string, string) {
    r := rand.New(rand.NewSource(seed))
    a := names[r.Intn(len(names))]
    b := names[r.Intn(len(names))]
    return a, b
}

// runOneFullSetGame plays one game with the named strategies, asserting
// the game terminates without errors. Uses bot.StrategyByName to build
// fresh strategy instances per game (so per-game counters reset).
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
```

- [ ] **Step 2: Run the closeout sweep to confirm it works**

```bash
go test ./internal/bot -run TestBotVsBot_FullBaseSet_RandomKingdoms -v -timeout 5m
```

Expected: PASS within ~30s–90s. Every one of the 1000 sub-tests reports OK. If any sub-test panics or returns a Connect error, the closeout has caught a real bug — investigate and fix before moving on. If the timeout fires, profile: the sweep should be CPU-bound on the in-process server, not blocked.

- [ ] **Step 3: Run `make test` to confirm the full suite is green**

```bash
make test
```

Expected: PASS — every Tier 0–5 unit test, every smoke sweep, every gated sweep, the property test sweep, and the new closeout sweep. This is the gate that proves Phase 1a is done.

- [ ] **Step 4: Run `make lint`**

```bash
make lint
```

Expected: clean.

- [ ] **Step 5: Run `make generate` as a sanity check**

```bash
make generate
```

Expected: no diff to `gen/go/` (no proto changes in Tier 5).

```bash
git status
```

Expected: clean apart from whatever untracked README work remains. If `gen/go/` has unexpected changes, commit them as `chore(proto): regenerate gen/go/` before continuing.

- [ ] **Step 6: Update the README to declare Phase 1a complete**

Read the current README to understand its layout:

```bash
cat README.md | head -80
```

Then edit `README.md` to make these updates (preserve the existing structure; only the items below should change):

1. Find the project status line (likely near the top, e.g. "Phase 1a / Tier 0 complete" or similar). Replace with: **`**Status:** Phase 1a complete — all 26 Base-set kingdom cards + 7 basics implemented and integration-tested.`**

2. If there is a card-list table, mark `Gardens` and `Merchant` as implemented (✅ or equivalent). If no such table exists, add a one-line note: "Tier 5 (special hooks) adds Gardens and Merchant."

3. In the section listing `make bot ARGS="-strategy ..."` examples (or the `-strategy` flag's documentation), append `merchant_bm` and `gardens_bm` to the list.

4. Append a "Phase 1a — Done when" attestation, e.g.:

   ```
   ## Phase 1a — Done

   The Phase 1a "Done when" criterion from the parent design spec has been met:

   - All 26 Base-set kingdom cards + 7 basics implemented.
   - 1000-game bot-vs-bot integration sweep over random 10-of-26 kingdom subsets passes
     (`TestBotVsBot_FullBaseSet_RandomKingdoms`).
   - 500-seed engine property sweep enforces card conservation across every random game.

   Next phase: Phase 1b — React + Vite + TypeScript frontend over the generated
   `connect-web` client.
   ```

   Place the attestation after the existing Tier descriptions / status section, before any Contributing or License footer.

- [ ] **Step 7: Verify the README change reads correctly**

```bash
git diff README.md
```

Expected: the diff matches the four edits above. If anything wasn't in the README to begin with (e.g., no card table, no strategy list), skip that bullet — only modify what's actually there. The point is to communicate to future readers that Phase 1a is shippable.

- [ ] **Step 8: Final full-suite run**

```bash
make test
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/bot/integration_full_set_test.go README.md
git commit -m "$(cat <<'EOF'
feat(bot): add Phase-1a closeout sweep + declare Phase 1a complete

1000-game random-kingdom integration sweep over the full 26-card pool
with random strategy pairs from the registered set. Skipped under
-short. README updated to attest Phase 1a's "Done when" criterion has
been met; next phase is 1b (React/Vite frontend).

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 10: Verify the working tree is clean and the branch is ready to merge**

```bash
git status
```

Expected: `nothing to commit, working tree clean`.

```bash
git log --oneline -10
```

Expected: 6 new commits (one per Task 1–6) on top of the previous Tier 4 head; all conventional-commit-formatted; co-authored.

---

## Milestone exit checklist

Run these in sequence; every item must pass before declaring Tier 5 — and Phase 1a — complete.

- [ ] `make test` is green (includes the full suite: 500-seed property sweep, all per-strategy gated sweeps, all per-strategy smoke sweeps, the new MerchantBM/GardensBM smoke sweeps, the 1000-game closeout sweep).
- [ ] `make test-short` is green (fast iteration path).
- [ ] `make lint` is clean.
- [ ] `make generate` produces no diff (no proto changes).
- [ ] `make bot ARGS="-strategy merchant_bm -create -seed 42 -as-player 0"` and `-strategy gardens_bm` both run end-to-end against `make server`.
- [ ] `bot.StrategyByName` round-trip test names match the registered strategies one-to-one (no missing, no extra).
- [ ] README declares Phase 1a complete with the "Done when" attestation.
- [ ] Tier 5 spec's open-questions list is empty (it already is).
- [ ] All six tasks merged on `main` (or, in worktree workflow, ready for PR).
