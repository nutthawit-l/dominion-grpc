# dominion-grpc — Tier 3 Design Spec (Attacks + reactions)

**Status:** Brainstorming complete. All sections approved by user. Next step is to invoke `superpowers:writing-plans`.
**Date:** 2026-04-22
**Author of brainstorm:** Claude (Opus 4.7) with user
**Parent spec:** [2026-04-12-dominion-grpc-design.md](2026-04-12-dominion-grpc-design.md) — Tier 3 definition in Section 5.
**Scope reminder:** This spec covers **Phase 1a / Tier 3** only: five cards introducing attacks and reactions, the reaction system (`OnReaction` + `Trigger`), an attack-resolution helper, three new deck-reveal primitives, extensions for one existing prompt plus one new prompt, WitchBM and MilitiaBM strategies, and the corresponding proto changes.

---

## Confirmed decisions (locked-in choices from the brainstorm)

| Topic | Decision |
|---|---|
| **Scope** | All 5 cards — Moat, Militia, Bureaucrat, Witch, Bandit |
| **Moat reaction trigger** | Automatic — engine auto-blocks when a reaction is in hand; no prompt |
| **Per-victim decision queue** | `Decision.Context` carries `remaining_victims` — same pattern Tier 2 uses for inter-step state |
| **Bot strategies** | WitchBM (done-criterion gate) + MilitiaBM (secondary, no gate) |
| **New prompts** | Reuse `DiscardFromHandPrompt`; extend `PutOnDeckPrompt` with `TypeFilter`; add `TrashFromRevealedPrompt`. No new answer types |
| **Reaction hook** | New `OnReaction` field on `Card` + a `Trigger` struct (parent-spec shape) |
| **Done-criterion** | 200 games, ≥0.55 threshold, seats alternated by seed parity |
| **Forced/null per-victim decisions** | Skip the prompt entirely — auto-apply forced effect, emit events, advance to next victim |
| **Milestone sequencing** | Tier 0 > Tier 1 > Tier 2 > **Tier 3** > Tier 4 > Phase 1b > Tier 5 |

---

## Section 1 — Reaction system (engine changes)

The core new machinery. Three pieces: a `Trigger` type, an `OnReaction` field on `Card`, and an attack-resolution helper.

### 1.1 `Trigger` struct

```go
type TriggerKind int

const (
    TriggerUnknown TriggerKind = iota
    TriggerAttackPlayed
)

type Trigger struct {
    Kind     TriggerKind
    Attacker PlayerIdx
    CardID   CardID    // the attack card being played
}
```

One kind for Tier 3. The struct is designed to extend (future triggers: `TriggerCardGained` for Watchtower, `TriggerAnyBuy`, etc.) without breaking existing reactions.

### 1.2 `Card.OnReaction` field

```go
type Card struct {
    ID            CardID
    Name          string
    Cost          int
    Types         []CardType
    OnPlay        func(gs *GameState, px PlayerIdx) []Event
    VictoryPoints func(p PlayerState) int
    OnResolve     func(gs *GameState, px PlayerIdx, d *Decision, answer Answer, lookup CardLookup) ([]Event, error)
    OnReaction    func(gs *GameState, victim PlayerIdx, trigger Trigger) (blocks bool, events []Event)  // NEW
}
```

Cards without reaction behavior leave `OnReaction` nil (all existing cards).

### 1.3 Attack-resolution helper

A single new function that every attack card calls instead of looping over victims by hand:

```go
// ResolveAttackVictims builds the ordered list of victims an attack applies
// to, running each victim's reactions inline. Returns:
//   - victims: the seats that did NOT block (in turn order, starting from
//     the seat after attacker).
//   - events: one EventAttackPlayed plus one EventReactionTriggered per
//     triggered reaction.
//
// Does NOT create decisions — the caller decides per-victim whether a
// decision is needed and sets up the first one. The remaining victims
// are handed back to the caller to stash in Decision.Context.
func ResolveAttackVictims(gs *GameState, attacker PlayerIdx, cardID CardID, lookup CardLookup) (victims []PlayerIdx, events []Event)
```

Internally it uses `EachOtherPlayer` iteration order, scans each victim's hand for any card with a non-nil `OnReaction`, calls `OnReaction` with the trigger, and records the victim as blocked if any reaction returns `blocks = true`. Emits `EventAttackPlayed` once and `EventReactionTriggered{PlayerIdx: victim, CardID: reactionCard}` per trigger.

### 1.4 New events

```go
const (
    // existing...
    EventAttackPlayed       // emitted once per attack-card play
    EventReactionTriggered  // emitted once per reaction that fires
    EventCardRevealed       // Bureaucrat's "reveal hand" / Bandit's top-2 reveal
)
```

### 1.5 New primitive: `RevealFromDeck` (no-discard variant)

Tier 2's `RevealAndDiscardFromDeck` is not reusable for Bandit, which reveals top 2 and then trashes/discards selectively. New primitive:

```go
// RevealFromDeck reveals the top n cards WITHOUT moving them anywhere.
// Returns revealed IDs to the caller. If the deck is empty, the discard
// is shuffled into the deck first (same shuffle rule as DrawCards).
// Caller is responsible for moving each revealed card to its final zone.
func RevealFromDeck(gs *GameState, px PlayerIdx, n int) (revealed []CardID, events []Event)
```

The caller gets back a slice of revealed IDs that are **still on top of the deck** (physically). The caller then uses `PutOnDeck` / `TrashFromDeck` / `DiscardFromDeck` to move them. Two paired helpers for Bandit's flow:

```go
// TrashFromDeck removes a specific card from the top region of the deck
// (the region the caller just revealed) and moves it to trash.
func TrashFromDeck(gs *GameState, px PlayerIdx, card CardID) []Event

// DiscardFromDeck removes a specific card from the top region of the
// deck and moves it to discard.
func DiscardFromDeck(gs *GameState, px PlayerIdx, card CardID) []Event
```

Both surface in `primitives.go` alongside their siblings to keep the engine's "here's what the card toolkit is" story clean.

### 1.6 New context keys

```go
const (
    // existing: CtxKeyTrashedCost, CtxKeyCard
    CtxKeyAttacker         ContextKey = "attacker"
    CtxKeyRemainingVictims ContextKey = "remaining_victims"
    CtxKeyRevealedCards    ContextKey = "revealed_cards"  // Bandit: cards still on top of deck awaiting decision
)
```

### 1.7 Decision.PlayerIdx already is the victim

`Apply`'s `applyResolveDecision` already dispatches answers from `d.PlayerIdx`, not `gs.CurrentPlayer` (Tier 2 Section 1.8). No changes needed for cross-player attack decisions.

---

## Section 2 — Card implementations

All 5 cards, each in its own file under `internal/engine/cards/`.

### 2.1 Moat — `kingdom_moat.go`, cost $2, Action-Reaction

```
Types: [TypeAction, TypeReaction]

OnPlay:    DrawCards(gs, px, 2)

OnReaction: func(gs, victim, trigger) (blocks, events) {
    if trigger.Kind != TriggerAttackPlayed { return false, nil }
    return true, []Event{{Kind: EventReactionTriggered, PlayerIdx: victim, CardID: "moat"}}
}
```

No decision. OnReaction is the only new-hook user in this tier.

### 2.2 Witch — `kingdom_witch.go`, cost $5, Action-Attack

*+2 Cards. Each other player gains a Curse.*

```
OnPlay:  DrawCards(gs, px, 2)
         victims, events := ResolveAttackVictims(gs, px, "witch", lookup)
         for _, v := range victims:
             events = append(events, GainCard(gs, v, "curse", GainToDiscard)...)
         return events
```

No decisions, no OnResolve needed. If the Curse pile is empty, `GainCard` is already a no-op — natural behavior.

### 2.3 Militia — `kingdom_militia.go`, cost $4, Action-Attack

*+2 Coins. Each other player discards down to 3 cards in hand.*

```
OnPlay:  AddCoins(gs, px, 2)
         victims, events := ResolveAttackVictims(gs, px, "militia", lookup)
         // Filter to victims with > 3 cards in hand (others have no decision)
         realVictims := filter(victims, |v| len(Hand(v)) > 3)
         if len(realVictims) == 0: return events
         first, rest := realVictims[0], realVictims[1:]
         n := len(Hand(first)) - 3
         events = append(events, RequestDecision(gs, first, "militia", 0,
             DiscardFromHandPrompt{Min: n, Max: n},
             Context{CtxKeyAttacker: px, CtxKeyRemainingVictims: rest})...)
         return events

OnResolve: cards := answer.(CardListAnswer).Cards
           validate len(cards) == len(Hand(px)) - 3   // else error
           events := DiscardFromHand(gs, px, cards)
           remaining := d.Context[CtxKeyRemainingVictims].([]PlayerIdx)
           realRemaining := filter(remaining, |v| len(Hand(v)) > 3)
           if len(realRemaining) == 0: return events
           next, rest := realRemaining[0], realRemaining[1:]
           n := len(Hand(next)) - 3
           events = append(events, RequestDecision(gs, next, "militia", 0,
               DiscardFromHandPrompt{Min: n, Max: n},
               Context{CtxKeyAttacker: d.Context[CtxKeyAttacker], CtxKeyRemainingVictims: rest})...)
           return events
```

Step stays `0` because each victim is their own independent decision — multi-step would mean "this same victim has a second question," which isn't the case here.

### 2.4 Bureaucrat — `kingdom_bureaucrat.go`, cost $4, Action-Attack

*Gain a Silver onto your deck. Each other player reveals a Victory card from their hand and puts it on their deck (or reveals a hand with no Victory cards).*

```
OnPlay:  events := GainCard(gs, px, "silver", GainToDeck)
         victims, attackEvents := ResolveAttackVictims(gs, px, "bureaucrat", lookup)
         events = append(events, attackEvents...)
         events = append(events, processBureaucratQueue(gs, victims, px, lookup)...)
         return events

processBureaucratQueue(gs, victims, attacker, lookup):
    for i, v := range victims:
        victories := victoriesInHand(v, lookup)
        switch len(victories):
        case 0:
            events = append(events, EventCardRevealed{PlayerIdx: v, ...whole hand...})
            continue           // no change to deck, advance to next victim
        case 1:
            events = append(events, PutOnDeck(gs, v, victories)...)
            continue           // forced; no decision
        default:
            // 2+ victories — real choice; create decision and park the rest
            rest := victims[i+1:]
            events = append(events, RequestDecision(gs, v, "bureaucrat", 0,
                PutOnDeckPrompt{TypeFilter: [TypeVictory]},
                Context{CtxKeyAttacker: attacker, CtxKeyRemainingVictims: rest})...)
            return events
    return events

OnResolve: chosen := answer.(CardListAnswer).Cards  // exactly 1 Victory
           validate len(chosen)==1 && isVictory && inHand
           events := PutOnDeck(gs, px, chosen)
           remaining := d.Context[CtxKeyRemainingVictims].([]PlayerIdx)
           attacker := d.Context[CtxKeyAttacker].(PlayerIdx)
           events = append(events, processBureaucratQueue(gs, remaining, attacker, lookup)...)
           return events
```

Both `OnPlay` and `OnResolve` use the same helper — the only loop is "advance through remaining victims, skipping the 0- and 1-Victory cases inline, parking the queue when a real choice appears."

### 2.5 Bandit — `kingdom_bandit.go`, cost $5, Action-Attack

*Gain a Gold. Each other player reveals the top 2 cards of their deck, trashes a revealed Treasure other than Copper, and discards the rest.*

```
OnPlay:  events := GainCard(gs, px, "gold", GainToDiscard)
         victims, attackEvents := ResolveAttackVictims(gs, px, "bandit", lookup)
         events = append(events, attackEvents...)
         events = append(events, processBanditQueue(gs, victims, px, lookup)...)
         return events

processBanditQueue(gs, victims, attacker, lookup):
    for i, v := range victims:
        revealed, revealEvents := RevealFromDeck(gs, v, 2)
        events = append(events, revealEvents...)
        events = append(events, EventCardRevealed{...revealed...})

        treasures := filter(revealed, |c| isTreasure(c, lookup) && c != "copper")
        switch len(treasures):
        case 0:
            for _, c := range revealed:
                events = append(events, DiscardFromDeck(gs, v, c)...)
            continue
        case 1:
            events = append(events, TrashFromDeck(gs, v, treasures[0])...)
            for _, c := range revealed where c != treasures[0]:
                events = append(events, DiscardFromDeck(gs, v, c)...)
            continue
        default:
            // 2 non-Copper treasures — real choice; park the rest
            rest := victims[i+1:]
            events = append(events, RequestDecision(gs, v, "bandit", 0,
                TrashFromRevealedPrompt{Cards: treasures},
                Context{
                    CtxKeyAttacker:         attacker,
                    CtxKeyRemainingVictims: rest,
                    CtxKeyRevealedCards:    revealed,
                })...)
            return events
    return events

OnResolve: choice := answer.(CardChoiceAnswer)
           revealed := d.Context[CtxKeyRevealedCards].([]CardID)
           validate choice.Card is in revealed and is a non-Copper Treasure
           events := TrashFromDeck(gs, px, choice.Card)
           for _, c := range revealed where c != choice.Card:
               events = append(events, DiscardFromDeck(gs, px, c)...)
           remaining := d.Context[CtxKeyRemainingVictims].([]PlayerIdx)
           attacker := d.Context[CtxKeyAttacker].(PlayerIdx)
           events = append(events, processBanditQueue(gs, remaining, attacker, lookup)...)
           return events
```

### 2.6 Registry

All 5 cards self-register via `init()` into `DefaultRegistry`, same pattern as Tier 1/2.

---

## Section 3 — Proto & engine decision-type changes

Tiny — just one additive field + one new prompt message + one engine prompt struct.

### 3.1 Extend `PutOnDeckPrompt`

**Proto (`game.proto`):**

```protobuf
message PutOnDeckPrompt {
    repeated CardType type_filter = 1;  // NEW — empty = any card
}
```

**Engine (`decision.go`):**

```go
type PutOnDeckPrompt struct {
    TypeFilter []CardType
}
```

Existing Artisan usage is unchanged — an empty/nil `TypeFilter` means "any card," same behavior as before. Bureaucrat passes `TypeFilter: [TypeVictory]`.

### 3.2 New `TrashFromRevealedPrompt`

**Proto (`game.proto`):**

```protobuf
message Decision {
    // existing fields 1-4

    oneof prompt {
        DiscardFromHandPrompt    discard_from_hand    = 5;
        TrashFromHandPrompt      trash_from_hand      = 6;
        GainFromSupplyPrompt     gain_from_supply     = 7;
        ChooseFromDiscardPrompt  choose_from_discard  = 8;
        PutOnDeckPrompt          put_on_deck          = 9;
        MayPlayActionPrompt      may_play_action      = 10;
        TrashFromRevealedPrompt  trash_from_revealed  = 11;  // NEW
    }
}

message TrashFromRevealedPrompt {
    repeated string cards = 1;  // the revealed cards the player may choose from
}
```

**Engine (`decision.go`):**

```go
type TrashFromRevealedPrompt struct {
    Cards []CardID
}

func (TrashFromRevealedPrompt) isPrompt() {}
```

Used only by Bandit. Answered with `CardChoiceAnswer` (existing — no new answer type).

### 3.3 No other proto changes

- Answer oneof: unchanged. `CardListAnswer`, `CardChoiceAnswer`, `YesNoAnswer` cover everything in Tier 3.
- `GainDest`: unchanged. `GAIN_DEST_DECK` already exists from Tier 2.
- `CardType`: unchanged. `CARD_TYPE_ATTACK` and `CARD_TYPE_REACTION` already exist from the Tier 0 proto.
- Event messages: none of the engine's new events (`EventAttackPlayed`, `EventReactionTriggered`, `EventCardRevealed`) cross the proto boundary in this tier — the service layer already rebuilds a full snapshot after each `SubmitAction`, so the stream carries the updated state. A future tier that wants typed attack/reaction events can add them then.

### 3.4 Service-layer translation

Two additions to `translate.go`:

- `PromptToProto`: new case for `TrashFromRevealedPrompt`; extended case for `PutOnDeckPrompt` copying `TypeFilter`.
- `AnswerFromProto`: unchanged — Bandit uses `CardChoiceAnswer`, Bureaucrat uses `CardListAnswer`, both already handled.

No `SubmitAction` or snapshot scrubbing changes. Bandit's revealed-from-deck cards are included in the prompt itself (public info — the attack revealed them), so no `PlayerView` shape change is needed.

### 3.5 Backwards compatibility

`buf breaking` passes — all changes are additive (one new field on an existing message, one new oneof variant, one new message). No field number reuse, no type change.

---

## Section 4 — Bot strategies

Two new strategies plus safe-refusal updates for the new prompts.

### 4.1 WitchBM — done-criterion strategy

**Buy policy:**
- Buy 1 Witch on turns 3–4 when `Coins >= 5` (first window it's affordable).
- After owning 1 Witch, never buy another.
- Otherwise BigMoney: Province at $8, Gold at $6, Silver at $3.

**Action phase — play policy:**
- Play Witch if in hand and `Actions >= 1`. No chaining.

**Resolve policy:**
- WitchBM owns only 1 Witch; it's not attacked by anything in a WitchBM-vs-BigMoney sweep (BigMoney never buys attacks), so WitchBM's Resolve is essentially `safeRefusal`. If a future integration sweep throws Militia/Bureaucrat/Bandit at it, safe refusal handles those correctly (see 4.3).

**One-Witch justification:** 1 Witch is the known-optimal count for WitchBM — the attack's value is in the Curse distribution, not in the per-turn draw. Additional Witches displace treasure buys without increasing Curse output.

### 4.2 MilitiaBM — secondary strategy, no done-criterion gate

**Buy policy:**
- Buy 1 Militia on turns 3–4 when `Coins >= 4`.
- After owning 1 Militia, never buy another.
- Otherwise BigMoney.

**Action phase — play policy:**
- Play Militia if in hand and `Actions >= 1`.

**Resolve policy:**
- **As attacker's opponent (discard-down-to-3 victim):** rank hand by "keep value" (`Gold > Silver > Copper` for treasures; Estates and Curses lowest). Sort descending, keep top 3, discard the rest.
- **Other prompt types:** `safeRefusal`.

### 4.3 Safe-refusal updates

`safeRefusal` gains cases for the new/extended prompts:

| Prompt | Safe refusal |
|---|---|
| `DiscardFromHandPrompt` (Militia forces Min=Max=n>0) | Pick any N cards — first N from hand. Already covered by existing "Min > 0" case. |
| `PutOnDeckPrompt{TypeFilter: [TypeVictory]}` (Bureaucrat) | Pick the first hand card matching the filter. Lowest-VP Victory if tie. |
| `TrashFromRevealedPrompt` (Bandit) | Pick the cheapest listed card (the prompt only ever lists non-Copper Treasures, so "cheapest" = Silver over Gold — gives up the less valuable one). |

BigMoney, SmithyBM, ChapelBM, RemodelBM use safe refusal for all Tier 3 prompts. No changes to their code — the `safeRefusal` function is shared.

### 4.4 Strategy registration

`bot.StrategyByName` gains two cases:

```go
case "witch_bm":    return WitchBM{}
case "militia_bm":  return MilitiaBM{}
```

### 4.5 `cmd/bot/main.go`

No structural change — the `-strategy` flag already dispatches through `StrategyByName`. Users can now run `make bot ARGS="-strategy witch_bm ..."`.

---

## Section 5 — Testing strategy

### 5.1 Engine unit tests — per card

One test file per card in `internal/engine/cards/`:

| File | Key cases |
|---|---|
| `kingdom_moat_test.go` | OnPlay draws 2; OnReaction returns `(true, …)` for `TriggerAttackPlayed`; returns `(false, nil)` for other trigger kinds; `Types` contains both `TypeAction` and `TypeReaction` |
| `kingdom_witch_test.go` | +2 cards to attacker; each non-blocked opponent gains a Curse from supply; Moat-holding opponents are skipped; Curse pile empty → effect is no-op for that opponent; 3-player variant (two opponents both get Curses when non-blocked) |
| `kingdom_militia_test.go` | +2 coins; opponent with ≤3 cards is skipped entirely; opponent with >3 gets `DiscardFromHandPrompt{Min:n,Max:n}`; multi-opponent queue (resolve opp1 → opp2 gets prompted); Moat blocks; wrong discard count → error |
| `kingdom_bureaucrat_test.go` | Silver gained to deck; Silver pile empty → no gain, attack still runs; opp with 0 Victories → `EventCardRevealed` only; opp with 1 Victory → auto-PutOnDeck, no prompt; opp with 2+ → `PutOnDeckPrompt{TypeFilter:[TypeVictory]}`; wrong type answer → error; Moat blocks |
| `kingdom_bandit_test.go` | Gold gained to discard; Gold pile empty → no gain; opp with 0 non-Copper Treasures revealed → auto-discard both; 1 non-Copper Treasure → auto-trash, other discarded; 2 → `TrashFromRevealedPrompt`; deck+discard empty → reveals 0; deck empty but discard has 1 → shuffles, reveals 1; answer must be in revealed list → error |

Each file also covers the game-ended guard and the phase guard (can't play in buy phase).

### 5.2 Primitive + helper tests

| Target | Key tests |
|---|---|
| `ResolveAttackVictims` | 2-player no-reaction; 4-player iteration order from seat `(attacker+1)` wrapping; one blocker (Moat) reported in returned events but excluded from `victims`; all opponents block → empty `victims` |
| `RevealFromDeck` | Reveals top n; shuffles discard if deck empty; deck+discard empty → returns empty, no error; cards remain on top of deck after return |
| `TrashFromDeck` | Moves specified top-region card to trash; card not in top region → error |
| `DiscardFromDeck` | Moves specified top-region card to discard; card not in top region → error |
| `OnReaction` dispatch | Engine calls `OnReaction` on every hand card with non-nil `OnReaction`; cards with `OnReaction == nil` are skipped; multiple reactions in one hand → any `blocks=true` blocks |

### 5.3 Decision system tests (in `apply_test.go`)

- Per-victim decisions: resolve from non-current player is accepted (already passes from Tier 2's deciding-player bypass — re-assert here with an attack-in-flight fixture).
- Attacker cannot take other actions while per-victim decisions are pending (blocked by existing `ErrDecisionPending`).
- Victim A resolves → victim B's decision becomes pending with step=0 and context containing the updated `remaining_victims` slice.

### 5.4 Bot strategy tests

- `witch_bm_test.go`: buys Witch on turns 3–4 when `Coins >= 5`; never a second Witch; plays Witch with `Actions >= 1`; Resolve is safeRefusal for all prompt types.
- `militia_bm_test.go`: buys Militia on turns 3–4 when `Coins >= 4`; plays Militia with `Actions >= 1`; Militia discard-down-to-3 keeps `Gold > Silver > Copper` over Estates/Curses; falls back to safeRefusal for other prompts.
- Safe refusal: extended to cover `TrashFromRevealedPrompt` (picks cheapest in list) and `PutOnDeckPrompt` with `TypeFilter` (picks first matching card).

### 5.5 Done-criterion sweep

`internal/bot/integration_test.go` gains:

```go
func TestBotVsBot_WitchBM_Outperforms_BigMoney(t *testing.T) {
    if testing.Short() { t.Skip("integration sweep") }

    const games = 200
    const threshold = 0.55
    wins := 0
    srv := newTestServer(t)
    defer srv.Close()

    for seed := int64(0); seed < games; seed++ {
        witchSeat := int(seed % 2)
        bmSeat := 1 - witchSeat
        strategies := []string{"", ""}
        strategies[witchSeat] = "witch_bm"
        strategies[bmSeat] = "bigmoney"

        final := runOneGame(t, srv.URL, seed, []string{"witch"}, strategies)
        if len(final.Winners) == 1 && int(final.Winners[0]) == witchSeat {
            wins++
        }
    }
    rate := float64(wins) / float64(games)
    require.GreaterOrEqualf(t, rate, threshold,
        "WitchBM win rate %.2f below threshold %.2f over %d games", rate, threshold, games)
}
```

Alternating seats. Expected true win rate ~0.70–0.80; 0.55 floor is safe against starting-player noise.

### 5.6 MilitiaBM integration test (no statistical gate)

`TestBotVsBot_MilitiaBM_CompletesNormally`: 50 games at fixed seeds, MilitiaBM vs BigMoney with `["militia"]` kingdom. Asserts every game terminates, card-conservation holds, and MilitiaBM wins at least some games (`wins >= 1` — sanity check, not a strength claim). This exercises the per-victim discard flow end-to-end.

### 5.7 Tier 2-style integration sweep

The existing 1000-game random-kingdom sweep extends its kingdom pool to include Tier 3 cards (`witch`, `militia`, `bureaucrat`, `bandit`, `moat`). Since only WitchBM and MilitiaBM buy attack cards, games where neither side is WitchBM/MilitiaBM won't exercise Witch/Bureaucrat/Bandit from the supply — that's fine; the sweep's purpose is crash-free termination and conservation invariants, which unit tests back up for the attack-card paths.

### 5.8 Property tests

The existing 500-seed property sweep (`properties_test.go`) continues to pass with Tier 3 cards in the registry. No new properties added; no RandomActionBot in this tier (deferred, matching Tier 1/2 decisions).

### 5.9 Runtime budget

- `make test-short` stays under 3s (adds ~5 unit test files).
- `make test` adds the 200-game WitchBM sweep (~10s expected) + 50-game MilitiaBM sweep (~3s). Total additional ~15s, well inside the 60s budget.

---

## Section 6 — Scope boundaries

**Ships in this tier:**

1. Reaction system: `Card.OnReaction` field, `Trigger` struct, `TriggerKind` enum with `TriggerAttackPlayed`.
2. `ResolveAttackVictims` helper — reaction-check + victim-list builder.
3. New primitives: `RevealFromDeck`, `TrashFromDeck`, `DiscardFromDeck`.
4. New events: `EventAttackPlayed`, `EventReactionTriggered`, `EventCardRevealed`.
5. New context keys: `CtxKeyAttacker`, `CtxKeyRemainingVictims`, `CtxKeyRevealedCards`.
6. Five kingdom cards: Moat, Militia, Bureaucrat, Witch, Bandit.
7. Proto changes: `PutOnDeckPrompt.type_filter` (additive), new `TrashFromRevealedPrompt` message + engine struct.
8. Service-layer `translate.go` updates for the two prompt changes.
9. Bot strategies: WitchBM (done-criterion), MilitiaBM (secondary).
10. Safe-refusal updates for the two new prompt shapes.
11. Done-criterion test: 200-game WitchBM vs BigMoney sweep.
12. MilitiaBM smoke sweep (50 games, termination + conservation, `wins >= 1`).
13. Tier 3 cards added to the random-kingdom integration sweep pool.

**Explicitly NOT in this tier:**

- Typed attack/reaction events over the stream (Tier 3 ships them as engine-internal only; proto-level typed events deferred).
- RandomActionBot / random-action property coverage.
- Throne Room, Library, Sentry (Tier 4).
- Gardens, Merchant (Tier 5).
- Persistence, CI pipeline, React frontend.
- Reaction cards beyond Moat (Watchtower, etc. — not Base set).
- Prompt-based reactions (Tier 3's Moat is auto-reveal only).
- Trigger kinds beyond `TriggerAttackPlayed`.

---

## Section 7 — Work ordering

Twelve reviewable units. Ordered so each PR lands green on top of `main` without depending on unlanded work.

### Stage A — Engine foundation (unblocks the cards)

1. **`OnReaction` field + `Trigger` struct.** Card struct change + unit test that reaction hook is called on hand scan. No consumers yet.
2. **`RevealFromDeck`, `TrashFromDeck`, `DiscardFromDeck` primitives.** Three primitives with unit tests. No consumers yet.
3. **`ResolveAttackVictims` helper + new event kinds.** Builds on #1; returns victim list + events. Unit tests for iteration order, Moat blocking, all-blocked case.
4. **Extend `PutOnDeckPrompt` with `TypeFilter`.** Proto field + engine struct field. Artisan behavior unchanged (empty filter). Service translate update. Test: Artisan still passes; new test with filter.
5. **New `TrashFromRevealedPrompt`.** Proto message + engine struct + translate update + safe-refusal case in bot. No card uses it yet.

### Stage B — The five cards (Witch and Moat first; rest in any order)

6. **Moat** (#1 consumer). `kingdom_moat.go` + test.
7. **Witch** (#3 consumer). `kingdom_witch.go` + test (uses `ResolveAttackVictims`; no decisions).
8. **Militia.** `kingdom_militia.go` + test (reuses `DiscardFromHandPrompt`; multi-victim queue).
9. **Bureaucrat** (#4 consumer). `kingdom_bureaucrat.go` + test (uses extended `PutOnDeckPrompt`).
10. **Bandit** (#2 + #5 consumer). `kingdom_bandit.go` + test (uses `RevealFromDeck` + `TrashFromRevealedPrompt`).

### Stage C — Strategies + sweeps

11. **WitchBM + done-criterion sweep.** Strategy + unit tests + `StrategyByName` registration + 200-game `TestBotVsBot_WitchBM_Outperforms_BigMoney`.
12. **MilitiaBM + smoke sweep + random-kingdom pool update.** Strategy + unit tests + `StrategyByName` registration + 50-game `TestBotVsBot_MilitiaBM_CompletesNormally` + extending the random-kingdom sweep pool to include `witch, militia, bureaucrat, bandit, moat`.

### Dependency graph

```
#1 OnReaction hook ─┐
                    ├── #3 ResolveAttackVictims ──┐
#2 Reveal/Trash/    │                              │
  DiscardFromDeck ──┤                              ├── #6 Moat
                    │                              ├── #7 Witch ── #11 WitchBM + sweep
#4 PutOnDeckPrompt  │                              ├── #8 Militia ── #12 MilitiaBM + sweep
  TypeFilter ───────┤                              ├── #9 Bureaucrat
                    │                              └── #10 Bandit
#5 TrashFromRevealed┘
  Prompt
```

### Milestone exit criteria

- All 12 issues closed.
- `make test` green on `main`, including the 200-game WitchBM sweep and 50-game MilitiaBM sweep.
- `make generate` and `make lint` clean.
- `buf breaking` passes.
- README's "Quick start" snippet works with `-strategy witch_bm` and `-strategy militia_bm`.

---

## Open questions

None currently blocking. Items deferred by decision during brainstorming:

- Typed attack/reaction events over the proto stream — deferred until a consumer (frontend, replay viewer) needs them.
- RandomActionBot strategy and random-action property-sweep coverage — still deferred from Tier 1/2.
- Prompt-based reactions (Watchtower-style) — not in Base set; the `OnReaction` signature can be extended when a future tier needs it.

---

## Next step

Invoke `superpowers:writing-plans` against this spec to produce the step-by-step implementation plan matching the 12-PR structure in Section 7.
