# dominion-grpc — Tier 5 Design Spec (Special hooks + Phase-1a closeout)

**Status:** Brainstorming complete. All sections approved by user. Next step is to invoke `superpowers:writing-plans`.
**Date:** 2026-05-01
**Author of brainstorm:** Claude (Opus 4.7) with user
**Parent spec:** [2026-04-12-dominion-grpc-design.md](2026-04-12-dominion-grpc-design.md) — Tier 5 definition in Section 5.
**Scope reminder:** This spec covers **Phase 1a / Tier 5** — the final two kingdom cards (Gardens, Merchant) plus the **Phase-1a closeout** work that the parent spec implies as the milestone exit: a 1000-game random-kingdom sweep over the full 26-card pool, a regression replay fixture, and a README update. After this tier merges, all 26 Base-set kingdom cards + 7 basics are implemented.

---

## Confirmed decisions (locked-in choices from the brainstorm)

| Topic | Decision |
|---|---|
| **Scope** | Both cards (Gardens, Merchant) plus Phase-1a closeout in one tier |
| **Merchant trigger** | Per-turn scalars on `PlayerState` (`MerchantBonusCharges int`, `FirstSilverPlayedThisTurn bool`); Silver's `OnPlay` reads them; `cleanupAndEndTurn` resets them |
| **Gardens kingdom-card recognition** | Expand `Card.IsKingdom()` to accept Victory cards whose ID is not in `{estate, duchy, province}`. No new field, no new type tag. |
| **Bot strategies** | `MerchantBM` and `GardensBM` are both **smoke-only** (50 games vs `BigMoney`, terminate + conservation + `wins ≥ 1`) |
| **Phase-1a closeout sweep** | New `TestBotVsBot_FullBaseSet_RandomKingdoms` — 1000 games, random 10-of-26 kingdom subsets, random strategy pairs from the full registered set, `-short` skip |
| **No proto changes** | No new prompts, decisions, events, or messages; `buf breaking` is trivially clean |
| **No property-test change** | New state fields are scalars, not card slices; conservation is unaffected |
| **Replay fixture** | One new JSON: TR-of-Merchant followed by Silver gives +$2 (regression guard for charges-double-on-replay) |
| **Milestone sequencing** | Tier 0 > Tier 1 > Tier 2 > Tier 3 > Tier 4 > **Tier 5** (Phase 1a complete) > Phase 1b |

---

## Section 1 — Engine changes

Two targeted additions, both contained.

### 1.1 Per-turn Merchant trigger fields on `PlayerState`

```go
type PlayerState struct {
    // ... existing zones, Actions, Buys, Coins, SetAside ...

    // MerchantBonusCharges is incremented by Merchant.OnPlay each time
    // Merchant is played in the current turn, and drained the first time
    // Silver is played that turn. Reset to 0 by cleanupAndEndTurn.
    MerchantBonusCharges int

    // FirstSilverPlayedThisTurn flips true the first time Silver.OnPlay
    // runs in a given turn. Reset to false by cleanupAndEndTurn. Guards
    // against repeat triggers within a turn.
    FirstSilverPlayedThisTurn bool
}
```

Both fields are **scalars**, not card slices, so the card-conservation property test is unaffected. The fields belong to the active player whose turn it is — they describe per-turn state for that player only — and are reset at end-of-turn for the player whose turn just ended.

### 1.2 `cleanupAndEndTurn` reset

`internal/engine/phases.go` — append two lines to the per-turn reset block (alongside the existing `Actions = 0`, `Buys = 0`, `Coins = 0`):

```go
gs.Players[px].MerchantBonusCharges = 0
gs.Players[px].FirstSilverPlayedThisTurn = false
```

These reset for the **outgoing** player (whose turn just ended), matching how Actions/Buys/Coins are zeroed there. The incoming player's fields are already zero (initial values), and remain so until that player plays a Merchant or Silver.

### 1.3 `IsKingdom()` predicate fix

`internal/engine/card.go`:

```go
func (c *Card) IsKingdom() bool {
    if c.HasType(TypeAction) {
        return true
    }
    // Non-basic Victory cards (e.g., Gardens) are kingdom too.
    // Estate / Duchy / Province are basic Victory and excluded here.
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

Two new test cases in `card_test.go`:

- A Gardens-shaped fake card (Victory, non-basic ID) returns `IsKingdom() == true`.
- Estate / Duchy / Province return `IsKingdom() == false`.

This change unblocks `resolveKingdom()` (which rejects non-kingdom IDs in custom kingdoms) and `discoverKingdom()` (which auto-populates random kingdoms), letting both honor Gardens.

### 1.4 What is *not* added

- **No new event kinds.** Merchant's bonus is internal to Silver's `OnPlay` and surfaces only through the existing `EventCoinsAdded` event (with the bonus added on a separate call so the event count reflects the Merchant trigger).
- **No new card field.** The trigger is scalar state on `PlayerState`, not a hook on `Card`.
- **No generic "on card played" subscription.** Silver checks `MerchantBonusCharges` directly. YAGNI for one card.
- **No new prompts or decisions.** Both Tier 5 cards run inside `OnPlay` / `VictoryPoints` only.

---

## Section 2 — Card implementations

### 2.1 Gardens — `kingdom_gardens.go`, cost $4, Victory

```go
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

func init() { DefaultRegistry.Register(Gardens) }
```

`SetAside` is included in the count defensively. By end-of-game cleanup it is empty for the active player; including it makes the function correct for any caller (e.g., debug snapshots, mid-game scoring previews) without a separate "active vs other player" code path. Each Gardens copy in any zone contributes `floor(n/10)` independently because `ComputeScore` calls `VictoryPoints(p)` once per copy.

**Edge cases.**

- Player with 0..9 cards: 0 VP.
- Player with 10..19 cards: 1 VP each Gardens.
- Player with 20 cards and 2 Gardens: 2 × 2 = 4 VP from Gardens.
- Cards in trash do **not** count (trash is not part of `PlayerState`).

### 2.2 Merchant — `kingdom_merchant.go`, cost $3, Action

```go
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

func init() { DefaultRegistry.Register(Merchant) }
```

Note: incrementing `MerchantBonusCharges` does **not** emit a dedicated event. The bonus surfaces later, when Silver fires, as a normal `EventCoinsAdded`.

### 2.3 Silver trigger — `basics_treasure.go` edit

```go
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

A short comment in the file explains the coupling: *Silver checks the per-turn Merchant trigger; the bookkeeping fields live on `PlayerState` and are owned by Tier 5.*

Copper and Gold are unchanged.

**Edge cases (with their tests in §3).**

- **No Merchant played**: `MerchantBonusCharges == 0`; Silver fires the flag flip but adds 0 bonus coins (no extra `EventCoinsAdded` event).
- **One Merchant, then Silver**: +1 bonus coin. Subsequent Silvers in the same turn add nothing extra.
- **Two Merchants, then Silver**: +2 bonus coins (`MerchantBonusCharges == 2` at the time of Silver's first play).
- **Silver, then Merchant in same turn**: Merchant's `OnPlay` still increments `MerchantBonusCharges` (engine doesn't gate this), but `FirstSilverPlayedThisTurn` is already true, so the next Silver this turn doesn't trigger. Per Dominion rules, you only get the bonus if Merchant is played *before* the first Silver. Charges left "stranded" in `MerchantBonusCharges` are flushed at cleanup.
- **Throne-Room-of-Merchant**: Throne Room replays Merchant's `OnPlay`; charges go 0→1→2; the next Silver this turn awards +$2.
- **Throne-Room-of-Silver**: Silver's `OnPlay` runs twice. First play flips the flag and pays bonus. Second play sees the flag set, pays no bonus (correct — only the *first* Silver triggers).
- **Multi-turn play across Throne / Vassal etc.** Cleanup resets both fields; next turn starts fresh.

### 2.4 Registration

`Gardens` and `Merchant` each have `init()` registering into `DefaultRegistry`. Once registered, `discoverKingdom()` (driven by `RegisterKingdomLister` from `cards/registry.go`) picks them up automatically because the patched `IsKingdom()` accepts both.

---

## Section 3 — Testing strategy

### 3.1 Engine unit tests — per card

`internal/engine/cards/kingdom_gardens_test.go`:

- `Metadata` (cost, types, name, registered).
- `VictoryPoints_FewerThanTen_IsZero` (0..9 cards → 0 VP).
- `VictoryPoints_TenCards_IsOne`.
- `VictoryPoints_NineteenCards_IsOne` (rounds down).
- `VictoryPoints_TwentyCards_IsTwo`.
- `VictoryPoints_CountsAllZones` — distribute 10 cards across Hand/Deck/Discard/InPlay/SetAside; expect 1 VP.
- `VictoryPoints_TrashDoesNotCount`.

`internal/engine/cards/kingdom_merchant_test.go`:

- `Metadata`.
- `OnPlay_DrawsAndAddsAction_NoBonusYet` — +1 card, +1 action; `MerchantBonusCharges` becomes 1; no Silver played → no extra coin event.
- `OnPlay_IncrementsMerchantBonusCharges` — multiple plays accumulate charges (use direct `Merchant.OnPlay` calls or Throne Room).
- `Played_ThenSilver_AddsOneCoin` — full Apply round-trip: play Merchant, switch to Buy phase, play Silver → coins reflect 2 (Silver) + 1 (Merchant bonus).
- `TwoMerchants_ThenSilver_AddsTwoCoins`.
- `Merchant_ThenSilverThenSilver_OnlyFirstSilverTriggersBonus` — second Silver adds 2 (no bonus).
- `Silver_ThenMerchant_NoBonusThisTurn` — flag set; charges accumulated by post-Silver Merchant don't fire.
- `Cleanup_ResetsCharges_AndFirstSilverFlag` — turn rollover starts fresh; verify both fields zeroed for the outgoing player.

`internal/engine/cards/basics_treasure_test.go` (extend):

- `Silver_OnPlay_NoMerchant_NoBonus`.
- `Silver_OnPlay_FlipsFirstSilverFlag`.
- `Silver_OnPlay_SecondTimeThisTurn_NoFlagFlip_NoBonus` — flag stays true; no extra coins; charges left stranded.

`internal/engine/cards/kingdom_throne_room_test.go` (one new case):

- `ThroneRoom_Merchant_DoublesBonusCharges` — TR(Merchant) → 2 charges; subsequent Silver → +$2.

### 3.2 Engine-level tests

`internal/engine/card_test.go` (extend):

- `IsKingdom_GardensQualifies` — Victory card with non-basic ID returns true.
- `IsKingdom_BasicVictoryNotKingdom` — estate/duchy/province all false.

`internal/engine/scoring_test.go` (extend):

- `ComputeScore_GardensAcrossZones` — Gardens in InPlay; PlayerState holds 18 cards across all zones → 1 VP from Gardens.

`internal/engine/phases_test.go` (extend):

- `Cleanup_ResetsMerchantFields` — set `MerchantBonusCharges = 3` and `FirstSilverPlayedThisTurn = true` for the active player; trigger cleanup; assert both zeroed.

### 3.3 Property test

**No change.** The new `PlayerState` fields are scalars (one int, one bool); `totalCards` does not read them; the existing 500-seed sweep continues to verify card conservation. With Gardens and Merchant added to the kingdom pool, the random-action sweep automatically exercises Merchant + Silver via the existing `randomLegalAction` helper (which already plays Coppers/Silvers/Golds in buy phase).

### 3.4 Bot strategy unit tests

`internal/bot/merchant_bm_test.go`:

- `Buy_PriorityOrder` — Province ≥8 > Gold ≥6 > Merchant if coins=3 and owned ≤ 2 > Silver ≥3 > Estate near end.
- `Play_PlaysMerchantInActionPhase`.
- `Play_NoMerchantInHand_EndsActionPhase`.

`internal/bot/gardens_bm_test.go`:

- `Buy_PriorityOrder` — Province ≥8 > Gold ≥6 > Gardens if coins=4 and owned ≤ 4 > Silver ≥3 > Copper if coins=0 (deliberate bulking).
- `Play_NoActionsToPlay_EndsActionPhase` — Gardens is Victory-only.

### 3.5 Bot smoke sweeps

`internal/bot/integration_test.go` (extend):

- `TestBotVsBot_MerchantBM_CompletesNormally` — 50 games vs `BigMoney`, alternating seats, all terminate ≤ 100 turns, conservation holds, `wins ≥ 1`. Skipped under `-short`.
- `TestBotVsBot_GardensBM_CompletesNormally` — same shape.

### 3.6 Phase-1a closeout sweep

`internal/bot/integration_full_set_test.go` (new file):

```go
func TestBotVsBot_FullBaseSet_RandomKingdoms(t *testing.T) {
    if testing.Short() {
        t.Skip("Phase-1a closeout sweep is slow; use go test -short to skip")
    }
    const games = 1000
    srv := newTestServer(t)
    defer srv.Close()

    pool := allKingdomCardIDs(t)              // every IsKingdom() card in DefaultRegistry
    require.GreaterOrEqual(t, len(pool), 26)  // Tiers 0–5 must all be registered

    strategies := []string{
        "bigmoney", "smithy_bm", "chapel_bm", "remodel_bm",
        "witch_bm", "militia_bm", "throneroom_bm", "library_bm",
        "sentry_bm", "merchant_bm", "gardens_bm",
    }

    for i := 0; i < games; i++ {
        seed := int64(i)
        kingdom := pickRandomKingdom(seed, pool, 10)
        a, b := pickRandomStrategyPair(seed, strategies)
        t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
            runOneGame(t, srv.URL, seed, kingdom, a, b)
        })
    }
}
```

Helpers `pickRandomKingdom`, `pickRandomStrategyPair`, `allKingdomCardIDs`, and `runOneGame` live in the same file or a shared `internal/bot/integration_helpers_test.go`.

**Failure conditions:**

- Any panic.
- Any Connect error (deterministic strategies must never submit illegal moves).
- Game does not terminate within 100 turns.
- Final state violates card conservation (re-checked at end-of-game).

**Runtime budget:** under 60 seconds on CI; engine is fast and the server runs in-process. If a future change pushes past 60 seconds, drop to 100 games on PRs and run 1000 nightly per parent spec §7A.5.

The pool is **derived**, not hardcoded. The test reads every `IsKingdom()` card from `DefaultRegistry` and uses that set; this auto-includes Gardens (now that `IsKingdom()` is fixed) without a hardcoded list, and prevents drift if a future tier adds cards.

### 3.7 Replay fixture

One new JSON under `internal/engine/testdata/replays/`:

- `tier5_throne_merchant_then_silver.json` — fixed seed, kingdom containing Throne Room + Merchant + Silver-supply, action log: TR → Merchant → Silver. Expected final: player coins reflect 2 (Silver) + 2 (TR-doubled Merchant bonus) = 4 from that play sequence. Locks in the charges-double-on-replay path against future regressions.

### 3.8 Runtime budget after Tier 5

Existing per-strategy sweeps:

| Sweep | Games |
|---|---|
| `BigMoney` (single) | 1 |
| `SmithyBM` outperforms (gated) | 200 |
| `ChapelBM` competitive | 50 |
| `RemodelBM` smoke | 50 |
| `WitchBM` outperforms (gated) | 200 |
| `MilitiaBM` smoke | 50 |
| `ThroneRoomBM` outperforms (gated) | 200 |
| `LibraryBM` smoke | 50 |
| `SentryBM` smoke | 50 |
| `MerchantBM` smoke (NEW) | 50 |
| `GardensBM` smoke (NEW) | 50 |
| **Phase-1a closeout (NEW)** | **1000** |
| **Total** | **~1951** |

Should finish in well under 90 seconds on the dev box; revisit the budget if not.

---

## Section 4 — Service / proto / bot infrastructure changes

### 4.1 Proto

**No changes.** Tier 5 introduces no new prompts, decisions, events, or messages. `buf breaking --against` is trivially clean.

### 4.2 Service layer

**No translate.go changes.** No new prompts to map. The bot strategies submit existing `PlayCard` and `BuyCard` actions; the service layer routes them through unchanged.

### 4.3 Bot library

`internal/bot/strategy.go` — extend `StrategyByName`:

```go
case "merchant_bm": return &MerchantBM{}, nil
case "gardens_bm":  return &GardensBM{}, nil
```

`safeRefusal` defaults: **no changes**, since no new prompts are introduced.

`cmd/bot/main.go` — add `merchant_bm` and `gardens_bm` to the `-strategy` help text.

### 4.4 Bot strategies

Both follow the existing `Strategy` interface and the file-per-strategy pattern.

**`internal/bot/merchant_bm.go`:**

- Action phase: play Merchant if in hand and `Actions ≥ 1`; otherwise end the phase.
- Buy phase: play all treasures in hand first, then:
  - Province if coins ≥ 8.
  - Gold if coins ≥ 6.
  - Merchant if coins = 3 and owned-Merchant ≤ 2 (don't over-buy; one or two charges per turn is the realistic ceiling for BigMoney-shape decks).
  - Silver if coins ≥ 3.
  - Estate near end (≤ 4 Provinces left).
- Resolve: delegate every prompt to `safeRefusal` (Merchant strategy never plays cards that prompt).

**`internal/bot/gardens_bm.go`:**

- Action phase: nothing to play (Gardens is Victory-only); always ends the phase.
- Buy phase: play all treasures in hand first, then:
  - Province if coins ≥ 8 (still relevant — Gardens-rush in 2p needs to also race Province depletion).
  - Gold if coins ≥ 6.
  - Gardens if coins = 4 and owned-Gardens ≤ 4.
  - Silver if coins ≥ 3.
  - Copper if coins = 0 and Buys remain (deliberate deck-bulking — the strategy *wants* a fat deck so Gardens score climbs).
- Resolve: delegate every prompt to `safeRefusal`.

---

## Section 5 — Documentation / README

`dominion-grpc/README.md` is updated as part of unit #6 of the work plan:

- Status line changes from "Phase 1a / Tier 0 complete" (currently outdated even for Tier 4) to "Phase 1a complete (Tiers 0–5)."
- Card-list table updated to mark Gardens and Merchant implemented; all 26 kingdom cards now show ✅.
- `make bot ARGS="-strategy ..."` example list extended with `merchant_bm` and `gardens_bm`.
- Phase-1a "Done when" criterion checked: 1000-game bot-vs-bot integration sweep across all 26 cards passes.
- "Next phase" note points to Phase 1b (React + Vite frontend).

---

## Section 6 — Scope boundaries

**Ships in this tier:**

1. Engine: `MerchantBonusCharges int` and `FirstSilverPlayedThisTurn bool` fields on `PlayerState`; reset in `cleanupAndEndTurn`.
2. Engine: `Card.IsKingdom()` predicate expanded to include non-basic Victory cards.
3. Cards: `Gardens` (`kingdom_gardens.go`), `Merchant` (`kingdom_merchant.go`), each with full unit tests.
4. Basics: `Silver` gets the trigger check on `OnPlay`; `Copper` and `Gold` unchanged.
5. Throne-Room-of-Merchant test case in `kingdom_throne_room_test.go`.
6. Bot strategies: `MerchantBM` and `GardensBM` (both smoke-only).
7. Bot smoke sweeps: 50-game `MerchantBM` and `GardensBM` integration tests.
8. Phase-1a closeout sweep: `TestBotVsBot_FullBaseSet_RandomKingdoms` — 1000 games with random 10-of-26 kingdoms and random strategy pairs over the registered set.
9. Replay fixture: `tier5_throne_merchant_then_silver.json`.
10. README update declaring Phase 1a complete.

**Explicitly NOT in this tier:**

- Generic "on card played" event-subscription system. Silver's direct `MerchantBonusCharges` check suffices for one card.
- New `TypeKingdomVictory` marker type or `Kingdom bool` field on `Card`. Expanded `IsKingdom()` predicate suffices for Base.
- New prompts, decisions, events, or proto messages.
- Property-test changes (the new state fields are scalars, conservation unaffected).
- Smarter Gardens or Merchant strategies (Gardens-rush hybrid, Workshop+Gardens combo, Merchant + Silver-buy timing optimization). Smoke sweeps prove correctness, not strategy quality. Gated done-criterion sweeps for Tier 5 strategies were considered and rejected during brainstorming (smoke-only chosen).
- Removing or merging any existing per-strategy sweep. The new 1000-game closeout is additive.
- Phase 1b (React frontend) and Phase 2 (auth, lobby, persistence). Out of phase as before.

---

## Section 7 — Work ordering

Six reviewable units. Each PR lands green on `main` independently.

### Stage A — Engine foundation

1. **`IsKingdom()` predicate fix + tests.** `card.go` predicate change; `card_test.go` cases for Gardens-shape and basic-Victory. No card consumes the relaxed predicate yet (Gardens not registered until #3). Standalone PR.
2. **`MerchantBonusCharges` + `FirstSilverPlayedThisTurn` fields on `PlayerState` + cleanup reset.** New scalars on `PlayerState`; two new lines in `cleanupAndEndTurn`. Direct test in `phases_test.go`: simulate a turn end with both fields set, assert both zeroed. No card consumes them yet.

### Stage B — Cards (independent after Stage A)

3. **Gardens.** `kingdom_gardens.go` + `kingdom_gardens_test.go` + `scoring_test.go` extension. Registers via `init()`. After this PR Gardens is in the registry, `IsKingdom()` accepts it, and scoring includes it.
4. **Merchant + Silver trigger.** `kingdom_merchant.go` + `kingdom_merchant_test.go` + `basics_treasure.go` Silver edit + `basics_treasure_test.go` extension + new Throne-Room-of-Merchant case in `kingdom_throne_room_test.go`. Both halves of the trigger ship together because each is meaningless alone.

### Stage C — Bots + closeout

5. **MerchantBM + GardensBM strategies + smoke sweeps.** `merchant_bm.go`, `gardens_bm.go`, both unit tests, `StrategyByName` registration, `cmd/bot/main.go` help text, two 50-game smoke sweeps in `internal/bot/integration_test.go`.
6. **Phase-1a closeout: full-set sweep + replay fixture + README.** New `internal/bot/integration_full_set_test.go` running 1000 games over random 10-of-26 kingdoms with random strategy pairs. New replay fixture `tier5_throne_merchant_then_silver.json`. README updated to reflect Phase 1a complete.

### Dependency graph

```
#1 IsKingdom fix ──┐
                   ├── #3 Gardens ───────┐
                   │                     ├── #5 Bots + smoke sweeps ── #6 Closeout
#2 PlayerState ────┴── #4 Merchant ──────┘
   fields + reset
```

### Milestone exit criteria

- All six units closed and merged.
- `make test` green on `main`, including the two 50-game smoke sweeps and the 1000-game closeout sweep.
- `make generate` and `make lint` clean. `buf breaking` clean (proto unchanged).
- `make bot ARGS="-strategy merchant_bm"` and `gardens_bm` run end-to-end against `make server`.
- Replay fixture passes: TR-of-Merchant then Silver = +$2 bonus from charges (4 total coins from the play sequence).
- README declares Phase 1a complete; all 26 kingdom cards + 7 basics implemented.

---

## Open questions

None remaining. All design choices were settled during brainstorming and are recorded in the "Confirmed decisions" table above.

---

## Next step

Invoke `superpowers:writing-plans` to produce the implementation plan from this spec. The plan will follow the same structure as the Tier 4 plan: numbered tasks with checkbox steps, TDD discipline (test → run-to-fail → implement → run-to-pass → commit), and one Conventional-Commit per task.
