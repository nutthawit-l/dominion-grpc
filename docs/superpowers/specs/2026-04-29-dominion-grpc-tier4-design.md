# dominion-grpc — Tier 4 Design Spec (Complex multi-step)

**Status:** Brainstorming complete. All sections approved by user. Next step is to invoke `superpowers:writing-plans`.
**Date:** 2026-04-29
**Author of brainstorm:** Claude (Sonnet 4.6) with user
**Parent spec:** [2026-04-12-dominion-grpc-design.md](2026-04-12-dominion-grpc-design.md) — Tier 4 definition in Section 5.
**Scope reminder:** This spec covers **Phase 1a / Tier 4** only: three kingdom cards introducing recursion (Throne Room), a set-aside zone with a per-card decision loop (Library), and a three-step decision flow (Sentry). It adds a small pending-plays mechanism on `GameState`, a new `SetAside` zone on `PlayerState`, four new prompt types, three bot strategies (one with a done-criterion gate), and the corresponding proto changes.

---

## Confirmed decisions (locked-in choices from the brainstorm)

| Topic | Decision |
|---|---|
| **Scope** | All three cards — Throne Room, Library, Sentry — in one tier |
| **Throne Room recursion** | `PendingPlays []PendingPlay` stack on `GameState`, popped by `Apply` whenever no decision is pending; LIFO unwinding handles arbitrary nesting (TR-of-TR-of-X) |
| **Library set-aside zone** | New `SetAside []CardID` field on `PlayerState`; included in card-conservation invariant; cleaned up by Library's `OnResolve` and defensively at end-of-turn |
| **Sentry decision flow** | Three sequential decisions (trash → discard → reorder), both top cards revealed up front into `SetAside`; `ReorderCardsPrompt` skipped when ≤ 1 card remains |
| **Throne Room prompt** | New `ChooseActionFromHandPrompt`; engine skips silently when hand has no Action card |
| **Bot strategies** | `ThroneRoomBM` gated (≥ 0.55 win rate over 200 games vs `BigMoney`), `LibraryBM` and `SentryBM` smoke-only |
| **TR-TR-Witch correctness** | 4 Curses on opponent (parent spec misstated as 2 — corrected here) |
| **New answer types** | None — all four new prompts reuse existing answers (`CardChoiceAnswer`, `YesNoAnswer`, `CardListAnswer`) |
| **Milestone sequencing** | Tier 0 > Tier 1 > Tier 2 > Tier 3 > **Tier 4** > Phase 1b > Tier 5 |

---

## Section 1 — Engine changes

Three additions to `internal/engine/`. None of these reshape Tier 2's decision system; they extend it.

### 1.1 `PendingPlays` stack on `GameState`

```go
type GameState struct {
    // ... existing fields ...
    PendingPlays []PendingPlay
}

type PendingPlay struct {
    PlayerIdx PlayerIdx
    CardID    CardID
    Source    CardID // e.g., "throne_room" — for events / observability
}
```

`PendingPlays` is a LIFO stack. Throne Room's `OnResolve` pushes one entry per replay it owes; `Apply` pops entries one at a time and runs them via `PlayCardInPlace`.

### 1.2 New helper: `PlayCardInPlace`

```go
// PlayCardInPlace runs a card's OnPlay without moving it between zones.
// The caller must have already placed the card in InPlay (or arranged
// equivalent state). Used by:
//   - Throne Room's OnResolve, for the first play of the doubled card.
//   - Apply's pending-play unwinder, for queued replays.
func PlayCardInPlace(gs *GameState, px PlayerIdx, cardID CardID,
                    lookup CardLookup) ([]Event, error)
```

Distinct from Tier 2's `PlayCardFromZone` (Vassal's helper), which moves a card from `Hand`/`Discard` → `InPlay` first. `PlayCardInPlace` emits `EventCardPlayed` and calls `c.OnPlay(gs, px)`.

### 1.3 `Apply`-side pending-play unwinding

After every `ResolveDecision` whose `OnResolve` returns successfully and leaves no new `PendingDecision`, `Apply` checks `gs.PendingPlays`. If non-empty:

1. Pop the **last** entry (LIFO).
2. Emit `EventCardThroned` (using `Source` for context) followed by `EventCardPlayed`.
3. Call `PlayCardInPlace` for the popped card.
4. If that play sets a new `PendingDecision`, stop unwinding and return — the next `ResolveDecision` will resume.
5. Otherwise, repeat from step 1 until the stack is empty.

This loop is what makes `TR(TR(X))` work: each Throne Room contributes its own continuation; LIFO order interleaves the inner replays before the outer ones, matching Dominion semantics.

### 1.4 `SetAside` zone on `PlayerState`

```go
type PlayerState struct {
    // ... existing zones: Hand, Deck, Discard, InPlay ...
    SetAside []CardID
}
```

Used by:
- **Library**: holds Actions the player chooses to set aside during draw-to-7. Discarded en masse when Library's loop exits.
- **Sentry**: holds the top 2 cards of deck during the trash/discard/reorder decision sequence. Empty when Sentry's `OnResolve` finishes.

The card-conservation property test (`internal/engine/properties_test.go`) is updated:

```go
total = sum(Hand + Deck + Discard + InPlay + SetAside) over players
      + sum(Supply.Piles) + len(Trash)
```

End-of-turn cleanup defensively flushes `SetAside → Discard` for the player whose turn just ended. In practice both Library and Sentry leave `SetAside` empty before completing, so the defensive sweep is belt-and-suspenders.

### 1.5 New events

Two engine-level event kinds in `state.go`:

- `EventCardThroned` — emitted when `Apply` pops a pending play (i.e., a Throne Room replay). Carries the doubled `CardID` and `Source = "throne_room"`.
- `EventCardSetAside` — emitted when Library moves an Action from hand to `SetAside`.

Per Tier 3 precedent, these stay engine-internal; the proto event stream still emits state snapshots and generic `action_applied` events. No proto event changes.

### 1.6 Context keys

No genuinely new keys are required.
- `CtxKeyCard` (existing) is reused by Library's `OnResolve` to remember the just-drawn card across the prompt.
- Sentry uses no context keys — its state lives in `SetAside` and is read directly from `PlayerState`.

---

## Section 2 — Card implementations

All three cards land in `internal/engine/cards/` and register via `init()` into `DefaultRegistry`.

### 2.1 Throne Room — `kingdom_throne_room.go`, cost $4, Action

```
OnPlay:
  - Scan hand for cards with TypeAction. If none → no-op (no decision).
  - RequestDecision(ChooseActionFromHandPrompt{}, step=0).

OnResolve (step 0):
  - C := answer.(CardChoiceAnswer).Card
  - Move C from Hand → InPlay (emit EventCardPlayed for first play).
  - Push PendingPlay{px, C, Source: "throne_room"} onto gs.PendingPlays.
  - Call C.OnPlay(gs, px) — first play. May set its own PendingDecision.
```

After `OnResolve` returns, `Apply` checks `PendingPlays`. If the first play set no decision, `Apply` immediately pops the queued second play and runs it. If the first play set a decision, the second play stays queued until the first's chained decisions all resolve.

**Edge case — TR-of-TR.** Outer TR's `OnResolve` picks inner TR, pushes "inner-TR-again", runs inner TR's first `OnPlay`, which prompts `ChooseActionFromHand`. Player picks card X. Inner TR's `OnResolve` pushes "X-again" onto the stack and plays X first. Stack now has `[outer-pushed inner-TR-again, inner-pushed X-again]`. LIFO unwinding runs X again, then inner TR again (which prompts again), then X again, then... — yielding **4 plays of the eventually-chosen Action card**. For TR-TR-Witch this means **4 Curses per opponent**.

**Edge case — no Action in hand.** `OnPlay` skips the prompt entirely; Throne Room is a no-op. This matches Mine/Moneylender's "skip prompt when nothing eligible" pattern.

### 2.2 Library — `kingdom_library.go`, cost $5, Action

Loop-based via a private `libraryStep(gs, px) []Event` helper called from both `OnPlay` and `OnResolve`:

```
libraryStep:
  Loop:
    - if len(Hand) >= 7:
        Move SetAside → Discard. Return (no decision).
    - if Deck empty AND Discard empty:
        Move SetAside → Discard. Return (no decision).
    - DrawCards(gs, px, 1). drawn := last card in Hand.
    - if drawn.HasType(TypeAction):
        RequestDecision(SetAsideActionPrompt{Card: drawn},
                        step=0,
                        ctx={CtxKeyCard: drawn})
        Return (waits for player).
    - else: keep silently and continue loop.

OnPlay:
  Return libraryStep(gs, px).

OnResolve (step 0):
  yn := answer.(YesNoAnswer)
  drawn := ctx[CtxKeyCard].(CardID)
  if yn.Yes:
    Move drawn from end-of-Hand → SetAside (emit EventCardSetAside).
  // else: keep in hand; no movement.
  Return libraryStep(gs, px).
```

**Edge cases.**
- Hand size already ≥ 7 at `OnPlay`: loop exits immediately, no decision, no draws.
- Deck and Discard both empty mid-loop: terminates with hand size < 7; `SetAside` still discards.
- Non-Action drawn: kept silently, loop continues without a decision round-trip.

### 2.3 Sentry — `kingdom_sentry.go`, cost $5, Action

Reveals top 2 of deck into `SetAside` (reused as the "inspecting" zone), then runs three sequential decisions.

```
OnPlay:
  events := DrawCards(1) ++ AddActions(1)
  // Move top ≤ 2 cards Deck → SetAside (reshuffle once if needed).
  revealed := peelToSetAside(gs, px, 2)
  if len(revealed) == 0:
    return events  // empty deck+discard, done.
  RequestDecision(TrashFromRevealedPrompt{Cards: revealed}, step=0)
  return events

OnResolve step 0 (trash):
  toTrash := answer.(CardListAnswer).Cards
  Move chosen cards SetAside → Trash.
  remaining := current SetAside.
  if len(remaining) == 0: return.
  RequestDecision(DiscardFromRevealedPrompt{Cards: remaining}, step=1)

OnResolve step 1 (discard):
  toDiscard := answer.(CardListAnswer).Cards
  Move chosen cards SetAside → Discard.
  remaining := current SetAside.
  if len(remaining) == 0: return.
  if len(remaining) == 1:
    Move SetAside → top of Deck. Return.
  RequestDecision(ReorderCardsPrompt{Cards: remaining}, step=2)

OnResolve step 2 (reorder):
  ordered := answer.(CardListAnswer).Cards  // ordered semantics.
  Move SetAside cards in `ordered` order to top of Deck.
```

**Conservation invariant.** During Sentry's resolution, the 2 revealed cards live in `SetAside`; the property test sums include `SetAside`, so total cards stay constant.

**Edge cases.**
- Deck + Discard both empty: reveal 0, no decisions.
- Deck has only 1 card after reshuffle: reveal 1; trash/discard prompts handle 0..1; reorder skipped.
- All revealed trashed: skip discard and reorder steps; SetAside is empty.

### 2.4 No new answer types

All three cards reuse Tier 2's existing answers:

| Prompt | Answer |
|---|---|
| `ChooseActionFromHandPrompt` | `CardChoiceAnswer` (single card; engine guarantees ≥ 1 legal choice) |
| `SetAsideActionPrompt` | `YesNoAnswer` |
| `TrashFromRevealedPrompt` | `CardListAnswer` (existing, Tier 3) |
| `DiscardFromRevealedPrompt` | `CardListAnswer` |
| `ReorderCardsPrompt` | `CardListAnswer` (interpreted as ordered) |

### 2.5 Registry

```go
func init() {
    DefaultRegistry.Register(ThroneRoom)
    DefaultRegistry.Register(Library)
    DefaultRegistry.Register(Sentry)
}
```

---

## Section 3 — Proto & service-layer changes

All proto changes are additive; `buf breaking --against` against `main` stays clean.

### 3.1 New prompt messages (`proto/dominion/v1/game.proto`)

Added as new variants inside the existing `Decision.prompt` oneof, alongside Tier 2/3 prompts:

```protobuf
message ChooseActionFromHandPrompt {
  // No fields — type filter (Action) is implicit and engine-enforced.
}

message SetAsideActionPrompt {
  string card_id = 1;  // the just-drawn Action card.
}

message DiscardFromRevealedPrompt {
  repeated string cards = 1;  // candidate cards from SetAside.
}

message ReorderCardsPrompt {
  repeated string cards = 1;  // cards to reorder; in Tier 4 always length 2.
}
```

### 3.2 No new answer messages

All four prompts use existing Tier 2 answer types: `CardChoiceAnswer`, `YesNoAnswer`, and `CardListAnswer` (re-interpreted as ordered for `ReorderCardsPrompt`).

### 3.3 Engine prompt structs (`internal/engine/decision.go`)

Mirror the proto messages:

```go
type ChooseActionFromHandPrompt struct{}
func (ChooseActionFromHandPrompt) isPrompt() {}

type SetAsideActionPrompt struct{ Card CardID }
func (SetAsideActionPrompt) isPrompt() {}

type DiscardFromRevealedPrompt struct{ Cards []CardID }
func (DiscardFromRevealedPrompt) isPrompt() {}

type ReorderCardsPrompt struct{ Cards []CardID }
func (ReorderCardsPrompt) isPrompt() {}
```

### 3.4 Service-layer translation (`internal/service/translate.go`)

Add four cases each in `promptToProto` and `promptFromProto`. Pure mapping — no behavior changes.

### 3.5 Bot safe-refusal (`internal/bot/strategy.go`)

Defaults for the four new prompts (used when a strategy doesn't override):

| Prompt | Safe-refusal default |
|---|---|
| `ChooseActionFromHandPrompt` | First Action in `ClientState.MyHand` (engine guarantees ≥ 1 exists) |
| `SetAsideActionPrompt` | `YesNoAnswer{Yes: false}` (keep in hand) |
| `DiscardFromRevealedPrompt` | `CardListAnswer{Cards: nil}` (discard nothing) |
| `ReorderCardsPrompt` | `CardListAnswer{Cards: <revealed list as-is>}` (identity reorder) |

### 3.6 Backwards compatibility

Adding fields inside an existing oneof is binary-safe with `buf breaking --against`. Pre-Tier-4 clients receiving a Tier 4 prompt see an empty `Decision.prompt` oneof — the bot's `unknownPromptError` path triggers a safe refusal, which is fine for forward-compat tests.

---

## Section 4 — Bot strategies

Three strategies under `internal/bot/`. All implement the existing `Strategy` interface. Only `ThroneRoomBM` is gated; the other two are smoke-only.

### 4.1 `ThroneRoomBM` — gated done-criterion strategy

File: `throneroom_bm.go`. Implements the canonical TR + Witch combo.

**Buy priority** (descending):

1. Province if coins ≥ 8.
2. Witch if coins ≥ 5 and Witch supply > 0.
3. Gold if coins ≥ 6.
4. Throne Room if coins = 4 and we own ≥ 1 Witch (otherwise nothing profitable to throne).
5. Silver if coins ≥ 3.
6. Estate near game end (≤ 4 Provinces left).

**Play priority** (Action phase):

1. If hand has Throne Room **and** at least one other Action → play Throne Room.
   - Throne Room's `ChooseActionFromHandPrompt`: prefer Witch, fall back to any other Action.
2. Else if hand has Witch → play Witch.
3. Else if hand has any other Action → play it.

Witch's victims auto-gain Curses; no per-victim decision is involved on the attacker side.

**Done-criterion gate** (mirrors Tier 3's `WitchBM`):

- 200 games against `BigMoney`, alternating seats by seed parity.
- Threshold: `wins / total ≥ 0.55`.
- Test: `TestBotVsBot_ThroneRoomBM_Outperforms_BigMoney` — `-short` skips it.
- Standard kingdom for the gate test: `[throne_room, witch, smithy, village, market, lab, festival, council_room, moat, mine]` (10 cards including TR + Witch + filler).

### 4.2 `LibraryBM` — smoke strategy

File: `library_bm.go`. BigMoney + buy Library.

**Buy priority**:
1. Province if coins ≥ 8.
2. Gold if coins ≥ 6.
3. Library if coins ≥ 5 and owned-Library count ≤ 2.
4. Silver if coins ≥ 3.
5. Estate near end.

**Play**: play Library if in hand. `SetAsideActionPrompt` defaults to refusal (`Yes: false`, keep in hand) — simple and correct for a BigMoney-shape deck where a drawn Action is rare and worth keeping.

**Smoke sweep**: 50 games vs `BigMoney`, termination + conservation checks, `wins ≥ 1`. Test: `TestBotVsBot_LibraryBM_CompletesNormally`.

### 4.3 `SentryBM` — smoke strategy

File: `sentry_bm.go`. BigMoney + buy Sentry for deck thinning.

**Buy priority**:
1. Province if coins ≥ 8.
2. Gold if coins ≥ 6.
3. Sentry if coins = 5 and owned-Sentry count ≤ 1.
4. Silver if coins ≥ 3.
5. Estate near end.

**Play**: play Sentry if in hand. Prompts:
- `TrashFromRevealedPrompt` → trash any Curse / Estate revealed.
- `DiscardFromRevealedPrompt` → discard nothing extra.
- `ReorderCardsPrompt` → identity order.

**Smoke sweep**: 50 games vs `BigMoney`, termination + conservation checks, `wins ≥ 1`. Test: `TestBotVsBot_SentryBM_CompletesNormally`.

### 4.4 Strategy registration

`internal/bot/strategy.go` — extend `StrategyByName`:

```go
case "throneroom_bm": return &ThroneRoomBM{}, nil
case "library_bm":    return &LibraryBM{}, nil
case "sentry_bm":     return &SentryBM{}, nil
```

`cmd/bot/main.go` — add the three names to the `-strategy` help text.

### 4.5 Random-kingdom integration sweep

`internal/bot/integration_test.go` — extend the random-kingdom card pool to include Throne Room, Library, Sentry alongside the 15 existing kingdom cards. Sweep size unchanged.

---

## Section 5 — Testing strategy

### 5.1 Engine unit tests — per card

`internal/engine/cards/kingdom_throne_room_test.go`:
- `Metadata` (cost, types, name).
- `OnPlay_NoActionInHand_NoDecision`.
- `OnPlay_OneActionInHand_RequestsChoice`.
- `OnResolve_PlaysCardTwice_NoDecisionCard` — Throne Smithy → +6 cards total, 1 PendingPlay popped.
- `OnResolve_FirstPlaySetsDecision_SecondPlayQueued` — Throne Cellar → first play prompts discard; second play is on `PendingPlays`.
- `Recursive_TR_TR_Witch` — 4 Curses on opponent verifies LIFO unwinding.
- `Recursive_TR_TR_NoOtherActions` — no-op chain still terminates cleanly.

`internal/engine/cards/kingdom_library_test.go`:
- `Metadata`.
- `OnPlay_HandAlreadySeven_NoDraw_NoDecision`.
- `OnPlay_DrawsNonAction_ContinuesWithoutDecision`.
- `OnPlay_DrawsAction_RequestsSetAsidePrompt`.
- `OnResolve_SetsAside_LoopsAgain`.
- `OnResolve_KeepsInHand_LoopsAgain`.
- `Loop_StopsAtSevenWithSetAside` — final hand size = 7 after discarding SetAside.
- `Loop_StopsWhenDeckAndDiscardEmpty`.
- `SetAside_DiscardedAtEnd`.

`internal/engine/cards/kingdom_sentry_test.go`:
- `Metadata`.
- `OnPlay_DrawsAndAddsAction_RevealsTwo`.
- `OnPlay_EmptyDeckAndDiscard_NoDecisions`.
- `OnPlay_OnlyOneCardLeft_RevealsOne`.
- `Step0_TrashAll_NoFurtherDecisions`.
- `Step0_TrashNone_AdvancesToDiscardStep`.
- `Step1_DiscardSome_AdvancesToReorderIfTwoLeft`.
- `Step1_OneLeftAfterTrashAndDiscard_PutsBackNoReorder`.
- `Step2_Reorder_PutsBackInOrder`.
- `SetAside_EmptyAtEnd`.

### 5.2 Engine apply / decision tests

`internal/engine/apply_test.go` extensions:
- `Apply_UnwindsPendingPlays_AfterResolve` — stub card pushes a `PendingPlay`; `Apply` pops and runs after `OnResolve`.
- `Apply_LIFO_Order` — push two PendingPlays, verify second-pushed runs first.
- `Apply_DoesNotPopWhilePendingDecisionSet`.

`internal/engine/decision_test.go`: parse-tests for the four new prompts (engine struct ↔ proto round-trip).

### 5.3 Property test extension

`internal/engine/properties_test.go` updated:

```go
total = sum(Hand + Deck + Discard + InPlay + SetAside) over players
      + sum(Supply.Piles) + len(Trash)
```

The 500-seed sweep already exercises random legal actions; with TR/Library/Sentry mixed into the kingdom pool, `SetAside` flux is automatically covered.

### 5.4 Bot strategy unit tests

- `internal/bot/throneroom_bm_test.go` — buy priority, play priority, `ChooseActionFromHandPrompt` picks Witch over Smithy.
- `internal/bot/library_bm_test.go` — buy priority, play, default refusal on `SetAsideActionPrompt`.
- `internal/bot/sentry_bm_test.go` — buy priority, trash-Curse / Estate logic, discard-none, identity reorder.

### 5.5 Bot done-criterion sweep

`internal/bot/integration_test.go`:
- `TestBotVsBot_ThroneRoomBM_Outperforms_BigMoney` — 200 games, `wins/total ≥ 0.55`, alternating seats. Skipped under `-short`.

### 5.6 Bot smoke sweeps

Same file:
- `TestBotVsBot_LibraryBM_CompletesNormally` — 50 games, all terminate, conservation holds, `wins ≥ 1`. Skipped under `-short`.
- `TestBotVsBot_SentryBM_CompletesNormally` — 50 games, same shape. Skipped under `-short`.

### 5.7 Random-kingdom integration sweep update

Extend the random-kingdom test pool with `throne_room, library, sentry`. The sweep already exercises termination + conservation; the new cards just become eligible kingdoms.

### 5.8 Replay fixture (`internal/engine/testdata/replays/`)

One new JSON fixture exercising TR-of-TR-of-Witch on a fixed seed — locks in the 4-Curse outcome as a regression guard against pending-plays stack regressions.

### 5.9 Runtime budget

Tier 3 contributed 200 + 50 = 250 integration games. Tier 4 adds 200 + 50 + 50 = 300 more. Total `make test` (non-short) integration sweep: ~550 games. Should still finish in < 30s on the dev box; revisit if not.

---

## Section 6 — Scope boundaries

**Ships in this tier:**

1. Engine: `PendingPlays` stack on `GameState`, `PlayCardInPlace` helper, `Apply`-side unwinding logic.
2. Engine: `SetAside` zone on `PlayerState`; conservation property updated; defensive end-of-turn cleanup.
3. Engine: `EventCardThroned`, `EventCardSetAside` event kinds.
4. Engine: 4 new prompt structs (`ChooseActionFromHandPrompt`, `SetAsideActionPrompt`, `DiscardFromRevealedPrompt`, `ReorderCardsPrompt`).
5. Three kingdom cards: Throne Room, Library, Sentry.
6. Proto: 4 new prompt messages added to the `Decision.prompt` oneof.
7. Service-layer translation updates for the four new prompts.
8. Bot strategies: `ThroneRoomBM` (gated), `LibraryBM` (smoke), `SentryBM` (smoke).
9. Safe-refusal defaults for the four new prompts.
10. Random-kingdom integration sweep extended with the three new cards.
11. Replay fixture for TR-TR-Witch (4 Curses).

**Explicitly NOT in this tier:**

- Tier 5 cards: Gardens (deck-size VP), Merchant (per-turn first-Silver trigger).
- Generic `ChooseFromHandPrompt` abstraction — `ChooseActionFromHandPrompt` is intentionally specific (YAGNI).
- Generic `Zones map[ZoneKey][]CardID` on `PlayerState` — `SetAside` is a single named field.
- Proto-level typed events for thrones / set-asides (engine-internal events only; same posture Tier 3 took).
- Reaction-system changes — Throne Room of an attack still triggers Moat reactions per attack play, but nothing new in Moat / Tier 3 is needed.
- Library "may" semantics for non-Action draws (Library only sets aside Actions per Base-set rules; the prompt only fires on Action draws).
- Persistence, Phase 1b frontend, auth / lobby — out of phase as before.

---

## Section 7 — Work ordering

Ten reviewable units, ordered so each PR lands green on `main` without depending on unlanded work.

### Stage A — Engine foundation (unblocks the cards)

1. **`PendingPlays` stack + `PlayCardInPlace` + `Apply` unwinding.** New field on `GameState`, new helper, new pop-loop in `Apply`. Unit tests for LIFO order, decision-pending guard, no-op when stack empty. No card consumes it yet.
2. **`SetAside` zone + conservation update.** New field on `PlayerState`; `properties_test.go` updated. End-of-turn cleanup defensively flushes `SetAside → Discard`. Unit tests for cleanup. No card consumes it yet.
3. **Four new prompts (engine + proto + translate + safe-refusal).** Engine structs, proto messages added to `Decision.prompt` oneof, `translate.go` cases, bot safe-refusal defaults. No card consumes them yet. `buf breaking` clean.

### Stage B — The three cards (any order after Stage A)

4. **Throne Room** (consumes #1, #3 `ChooseActionFromHandPrompt`). `kingdom_throne_room.go` + tests including TR-of-TR-of-Witch.
5. **Library** (consumes #2, #3 `SetAsideActionPrompt`). `kingdom_library.go` + tests including hand-already-7 / both-empty edge cases.
6. **Sentry** (consumes #2, #3 `DiscardFromRevealedPrompt` + `ReorderCardsPrompt`, reuses Tier 3's `TrashFromRevealedPrompt`). `kingdom_sentry.go` + tests including 0/1/2-card-remaining branches.

### Stage C — Strategies + sweeps

7. **ThroneRoomBM + done-criterion sweep.** `throneroom_bm.go` + unit tests + `StrategyByName` registration + 200-game `TestBotVsBot_ThroneRoomBM_Outperforms_BigMoney`.
8. **LibraryBM + smoke sweep.** `library_bm.go` + unit tests + registration + 50-game smoke.
9. **SentryBM + smoke sweep.** `sentry_bm.go` + unit tests + registration + 50-game smoke.
10. **Random-kingdom pool extension + replay fixture.** Add TR / Library / Sentry to the random-kingdom test pool. Add JSON replay fixture for TR-TR-Witch.

### Dependency graph

```
#1 PendingPlays + PlayCardInPlace ──┐
                                    ├── #4 Throne Room ── #7 ThroneRoomBM + sweep
#3 New prompts (4) ─────────────────┤
                                    ├── #5 Library ────── #8 LibraryBM + sweep
#2 SetAside zone ───────────────────┤
                                    └── #6 Sentry ─────── #9 SentryBM + sweep

#10 Random-kingdom pool + replay fixture
   (depends on #4 + #5 + #6 + #7 + #8 + #9)
```

### Milestone exit criteria

- All 10 units closed.
- `make test` green on `main`, including the 200-game ThroneRoomBM sweep and the two 50-game smoke sweeps.
- `make generate` and `make lint` clean.
- `buf breaking` passes against `main`.
- `make bot ARGS="-strategy throneroom_bm"`, `library_bm`, `sentry_bm` all run end-to-end against `make server`.
- TR-TR-Witch replay fixture passes — 4 Curses on opponent.

---

## Open questions

None remaining. All design choices were settled during brainstorming and are recorded in the "Confirmed decisions" table above.

---

## Next step

Invoke `superpowers:writing-plans` to produce the implementation plan from this spec. The plan will follow the same structure as the Tier 3 plan: numbered tasks with checkbox steps, TDD discipline (test → run-to-fail → implement → run-to-pass → commit), and one Conventional-Commit per task.
