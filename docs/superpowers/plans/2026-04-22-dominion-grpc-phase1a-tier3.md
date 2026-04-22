# Phase 1a / Tier 3 Implementation Plan — Attacks + reactions

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the reaction system (`Card.OnReaction` + `Trigger`), an attack-resolution helper (`ResolveAttackVictims`), three deck-reveal primitives (`RevealFromDeck`, `TrashFromDeck`, `DiscardFromDeck`), one extended prompt (`PutOnDeckPrompt.TypeFilter`) and one new prompt (`TrashFromRevealedPrompt`). Implement five kingdom cards — Moat, Witch, Militia, Bureaucrat, Bandit. Ship WitchBM (done-criterion, ≥0.55 win rate) and MilitiaBM (secondary smoke sweep) bot strategies.

**Architecture:** Attacks iterate opponents via `EachOtherPlayer` ordering through a new helper that runs each victim's reactions inline and returns the non-blocked victims. Per-victim decisions (Militia discard, Bureaucrat reveal-Victory, Bandit trash-from-revealed) use `Decision.Context` to carry the remaining-victims queue across `OnResolve` cycles — same pattern Tier 2 uses for `CtxKeyTrashedCost`. Moat's reaction is automatic: when a victim has any card with a non-nil `OnReaction` in hand at the moment of attack, that card's `OnReaction` runs and can block. Forced/null per-victim cases skip the prompt entirely and auto-apply (matching Tier 2's Moneylender/Poacher/Harbinger precedent).

**Tech Stack:** Go 1.22+, Connect-Go RPC (`connectrpc.com/connect`), protobuf via `buf`, testing via stdlib `testing` + `github.com/stretchr/testify/require`, concurrency via `golang.org/x/sync/errgroup`.

**Working directory:** All commands below run from `/home/tie/superpower-dominion/dominion-grpc/` unless explicitly noted.

**Design source:** [docs/superpowers/specs/2026-04-22-dominion-grpc-tier3-design.md](../specs/2026-04-22-dominion-grpc-tier3-design.md)

---

## File Structure

Files created or modified in this plan:

| Stage | File | Purpose |
|---|---|---|
| A | `dominion-grpc/internal/engine/types.go` | Add `TriggerKind` enum + `Trigger` struct |
| A | `dominion-grpc/internal/engine/card.go` | Add `OnReaction` field to `Card` |
| A | `dominion-grpc/internal/engine/state.go` | Add `EventAttackPlayed`, `EventReactionTriggered`, `EventCardRevealed` event kinds |
| A | `dominion-grpc/internal/engine/decision.go` | Add `CtxKeyAttacker`, `CtxKeyRemainingVictims`, `CtxKeyRevealedCards` context keys; add `TrashFromRevealedPrompt`; add `TypeFilter` field to `PutOnDeckPrompt` |
| A | `dominion-grpc/internal/engine/decision_test.go` | Tests for new context keys + new prompt type |
| A | `dominion-grpc/internal/engine/primitives.go` | Add `RevealFromDeck`, `TrashFromDeck`, `DiscardFromDeck` |
| A | `dominion-grpc/internal/engine/primitives_test.go` | Tests for the three new primitives |
| A | `dominion-grpc/internal/engine/attack.go` | New file: `ResolveAttackVictims` helper |
| A | `dominion-grpc/internal/engine/attack_test.go` | New file: tests for reaction scan + victim list |
| A | `dominion-grpc/proto/dominion/v1/game.proto` | Add `TypeFilter` to `PutOnDeckPrompt`; add new `TrashFromRevealedPrompt` message + oneof variant |
| A | `dominion-grpc/gen/go/dominion/v1/*.go` | Regenerated via `make generate` |
| A | `dominion-grpc/internal/service/translate.go` | Update `PromptToProto` for new prompt + extended field |
| A | `dominion-grpc/internal/service/translate_test.go` | Tests for new translations |
| B | `dominion-grpc/internal/engine/cards/kingdom_moat.go` | Moat card |
| B | `dominion-grpc/internal/engine/cards/kingdom_moat_test.go` | Moat tests |
| B | `dominion-grpc/internal/engine/cards/kingdom_witch.go` | Witch card |
| B | `dominion-grpc/internal/engine/cards/kingdom_witch_test.go` | Witch tests |
| B | `dominion-grpc/internal/engine/cards/kingdom_militia.go` | Militia card |
| B | `dominion-grpc/internal/engine/cards/kingdom_militia_test.go` | Militia tests |
| B | `dominion-grpc/internal/engine/cards/kingdom_bureaucrat.go` | Bureaucrat card |
| B | `dominion-grpc/internal/engine/cards/kingdom_bureaucrat_test.go` | Bureaucrat tests |
| B | `dominion-grpc/internal/engine/cards/kingdom_bandit.go` | Bandit card |
| B | `dominion-grpc/internal/engine/cards/kingdom_bandit_test.go` | Bandit tests |
| C | `dominion-grpc/internal/bot/strategy.go` | Update `safeRefusal` for `PutOnDeckPrompt` filter + `TrashFromRevealedPrompt`; update `cardCost` for Tier 3 costs |
| C | `dominion-grpc/internal/bot/witch_bm.go` | WitchBM strategy |
| C | `dominion-grpc/internal/bot/witch_bm_test.go` | WitchBM unit tests |
| C | `dominion-grpc/internal/bot/militia_bm.go` | MilitiaBM strategy |
| C | `dominion-grpc/internal/bot/militia_bm_test.go` | MilitiaBM unit tests |
| C | `dominion-grpc/internal/bot/integration_test.go` | WitchBM done-criterion sweep + MilitiaBM smoke sweep |
| C | `dominion-grpc/cmd/bot/main.go` | Wire `witch_bm` / `militia_bm` into `-strategy` flag |

---

## Stage A: Engine foundation

### Task 1: `OnReaction` hook on `Card` + `Trigger` struct

**Files:**
- Modify: `dominion-grpc/internal/engine/types.go`
- Modify: `dominion-grpc/internal/engine/card.go`
- Modify: `dominion-grpc/internal/engine/state.go` (new event kinds)

- [ ] **Step 1: Write the failing test for the reaction hook dispatch shape**

Create `dominion-grpc/internal/engine/card_test.go` (new file):

```go
package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCard_OnReaction_NilByDefault(t *testing.T) {
	c := &Card{ID: "nothing_special", Name: "X", Cost: 0}
	require.Nil(t, c.OnReaction, "cards without a reaction must leave OnReaction nil")
}

func TestTrigger_AttackPlayedKind(t *testing.T) {
	tr := Trigger{Kind: TriggerAttackPlayed, Attacker: PlayerIdx(0), CardID: "witch"}
	require.Equal(t, TriggerAttackPlayed, tr.Kind)
	require.Equal(t, PlayerIdx(0), tr.Attacker)
	require.Equal(t, CardID("witch"), tr.CardID)
}

func TestCard_OnReaction_CallableReturnsBlocksAndEvents(t *testing.T) {
	called := false
	c := &Card{
		ID: "fake_reaction", Name: "X", Cost: 2,
		OnReaction: func(gs *GameState, victim PlayerIdx, trigger Trigger) (bool, []Event) {
			called = true
			return true, []Event{{Kind: EventReactionTriggered, PlayerIdx: victim, CardID: "fake_reaction"}}
		},
	}
	gs := NewTestStateWithRNG("t", 1, []PlayerState{{Name: "a"}})
	blocks, ev := c.OnReaction(gs, 0, Trigger{Kind: TriggerAttackPlayed, Attacker: 0, CardID: "witch"})
	require.True(t, called)
	require.True(t, blocks)
	require.Len(t, ev, 1)
	require.Equal(t, EventReactionTriggered, ev[0].Kind)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/engine/ -run "TestCard_OnReaction|TestTrigger_" -v`
Expected: FAIL — `OnReaction`, `Trigger`, `TriggerAttackPlayed`, `EventReactionTriggered` all undefined.

- [ ] **Step 3: Add `TriggerKind` and `Trigger` to `types.go`**

In `dominion-grpc/internal/engine/types.go`, append after the existing `GainDest` block:

```go
// TriggerKind identifies what event fired a reaction.
type TriggerKind int

const (
	TriggerUnknown TriggerKind = iota
	TriggerAttackPlayed
)

// Trigger describes why a card's OnReaction is being invoked.
type Trigger struct {
	Kind     TriggerKind
	Attacker PlayerIdx
	CardID   CardID // the attack card being played
}
```

- [ ] **Step 4: Add `OnReaction` field to `Card`**

In `dominion-grpc/internal/engine/card.go`, add a new field to the `Card` struct after `OnResolve`:

```go
// OnReaction is called when an event fires that a reaction card can
// respond to. Returns (blocks, events): blocks=true means the reaction
// intercepted the triggering effect (e.g., Moat blocks an attack).
// Nil for cards without reaction behavior.
OnReaction func(gs *GameState, victim PlayerIdx, trigger Trigger) (blocks bool, events []Event)
```

Final `Card` struct:

```go
type Card struct {
	ID    CardID
	Name  string
	Cost  int
	Types []CardType

	OnPlay func(gs *GameState, px PlayerIdx) []Event

	VictoryPoints func(p PlayerState) int

	OnResolve func(gs *GameState, px PlayerIdx, d *Decision, answer Answer, lookup CardLookup) ([]Event, error)

	OnReaction func(gs *GameState, victim PlayerIdx, trigger Trigger) (blocks bool, events []Event)
}
```

- [ ] **Step 5: Add new `EventKind` values to `state.go`**

In `dominion-grpc/internal/engine/state.go`, extend the `EventKind` const block — add three new values at the end:

```go
const (
	EventUnknown EventKind = iota
	EventCardDrawn
	EventCardDiscarded
	EventCardPlayed
	EventCardGained
	EventCardTrashed
	EventCoinsAdded
	EventBuysAdded
	EventActionsAdded
	EventPhaseChanged
	EventTurnStarted
	EventGameEnded
	EventDecisionRequested
	EventCardPutOnDeck
	EventAttackPlayed      // NEW — emitted once per attack-card play
	EventReactionTriggered // NEW — emitted once per reaction that fires
	EventCardRevealed      // NEW — Bureaucrat reveal hand / Bandit top-2 reveal
)
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/engine/ -run "TestCard_OnReaction|TestTrigger_" -v`
Expected: PASS.

Run: `go build ./...`
Expected: success (no compile errors from the new `Card` field — all existing cards leave `OnReaction` nil).

- [ ] **Step 7: Commit**

```bash
git add internal/engine/card.go internal/engine/card_test.go internal/engine/types.go internal/engine/state.go
git commit -m "$(cat <<'EOF'
feat(engine): add OnReaction hook on Card + Trigger struct

Introduces the reaction-system primitives the Tier 3 spec calls for:
OnReaction function field on Card, Trigger + TriggerKind for classifying
why a reaction fires, and three new EventKind values (EventAttackPlayed,
EventReactionTriggered, EventCardRevealed). No card uses any of these
yet — Moat is the first consumer in a later task.
EOF
)"
```

---

### Task 2: Reveal / trash / discard-from-deck primitives

**Files:**
- Modify: `dominion-grpc/internal/engine/primitives.go`
- Modify: `dominion-grpc/internal/engine/primitives_test.go`

- [ ] **Step 1: Write the failing test for `RevealFromDeck`**

Append to `dominion-grpc/internal/engine/primitives_test.go`:

```go
func TestRevealFromDeck_TopCardsRemainOnDeck(t *testing.T) {
	gs := NewTestStateWithRNG("t", 1, []PlayerState{{
		Name: "a",
		Deck: []CardID{"copper", "silver", "gold"}, // gold is top
	}})

	revealed, events := RevealFromDeck(gs, 0, 2)

	require.Equal(t, []CardID{"gold", "silver"}, revealed)
	require.Equal(t, []CardID{"copper", "silver", "gold"}, gs.Players[0].Deck,
		"revealed cards must remain physically on top of the deck")
	require.Empty(t, events, "RevealFromDeck emits no events by itself; caller emits EventCardRevealed")
}

func TestRevealFromDeck_ShufflesDiscardWhenDeckEmpty(t *testing.T) {
	gs := NewTestStateWithRNG("t", 1, []PlayerState{{
		Name:    "a",
		Discard: []CardID{"estate", "copper"},
	}})

	revealed, _ := RevealFromDeck(gs, 0, 2)

	require.Len(t, revealed, 2)
	require.Empty(t, gs.Players[0].Discard, "discard must be moved into deck before reveal")
	require.Len(t, gs.Players[0].Deck, 2)
}

func TestRevealFromDeck_DeckAndDiscardEmpty_ReturnsEmpty(t *testing.T) {
	gs := NewTestStateWithRNG("t", 1, []PlayerState{{Name: "a"}})
	revealed, _ := RevealFromDeck(gs, 0, 2)
	require.Empty(t, revealed)
}

func TestRevealFromDeck_RequestMoreThanAvailable_ReturnsWhatExists(t *testing.T) {
	gs := NewTestStateWithRNG("t", 1, []PlayerState{{
		Name: "a",
		Deck: []CardID{"copper"},
	}})
	revealed, _ := RevealFromDeck(gs, 0, 2)
	require.Equal(t, []CardID{"copper"}, revealed)
}

func TestTrashFromDeck_RemovesCardAndEmitsEvent(t *testing.T) {
	gs := NewTestStateWithRNG("t", 1, []PlayerState{{
		Name: "a",
		Deck: []CardID{"copper", "silver", "gold"},
	}})

	events := TrashFromDeck(gs, 0, "gold")

	require.Equal(t, []CardID{"copper", "silver"}, gs.Players[0].Deck)
	require.Equal(t, []CardID{"gold"}, gs.Trash)
	require.Len(t, events, 1)
	require.Equal(t, EventCardTrashed, events[0].Kind)
	require.Equal(t, CardID("gold"), events[0].CardID)
}

func TestTrashFromDeck_CardAbsent_NoOp(t *testing.T) {
	gs := NewTestStateWithRNG("t", 1, []PlayerState{{
		Name: "a",
		Deck: []CardID{"copper"},
	}})
	events := TrashFromDeck(gs, 0, "gold")
	require.Empty(t, events)
	require.Equal(t, []CardID{"copper"}, gs.Players[0].Deck)
	require.Empty(t, gs.Trash)
}

func TestDiscardFromDeck_RemovesCardAndEmitsEvent(t *testing.T) {
	gs := NewTestStateWithRNG("t", 1, []PlayerState{{
		Name: "a",
		Deck: []CardID{"copper", "silver", "gold"},
	}})

	events := DiscardFromDeck(gs, 0, "silver")

	require.Equal(t, []CardID{"copper", "gold"}, gs.Players[0].Deck)
	require.Equal(t, []CardID{"silver"}, gs.Players[0].Discard)
	require.Len(t, events, 1)
	require.Equal(t, EventCardDiscarded, events[0].Kind)
}

func TestDiscardFromDeck_CardAbsent_NoOp(t *testing.T) {
	gs := NewTestStateWithRNG("t", 1, []PlayerState{{
		Name: "a",
		Deck: []CardID{"copper"},
	}})
	events := DiscardFromDeck(gs, 0, "gold")
	require.Empty(t, events)
	require.Equal(t, []CardID{"copper"}, gs.Players[0].Deck)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -run "TestRevealFromDeck|TestTrashFromDeck|TestDiscardFromDeck" -v`
Expected: FAIL — all three primitives undefined.

- [ ] **Step 3: Implement `RevealFromDeck`, `TrashFromDeck`, `DiscardFromDeck`**

Append to `dominion-grpc/internal/engine/primitives.go`:

```go
// RevealFromDeck reveals the top n cards WITHOUT moving them. If the
// deck is empty it shuffles the discard into the deck first. Returns
// the revealed card IDs in reveal order (top-of-deck first). The cards
// remain physically on top of the deck; the caller is responsible for
// moving each one to its final zone via TrashFromDeck / DiscardFromDeck
// / PutOnDeck.
func RevealFromDeck(gs *GameState, px PlayerIdx, n int) ([]CardID, []Event) {
	ps := &gs.Players[px]
	if len(ps.Deck) < n {
		if len(ps.Discard) > 0 {
			// Move discard into deck and shuffle, then continue.
			ps.Deck = append(ps.Deck, ps.Discard...)
			ps.Discard = nil
			shuffleCards(gs.rng, ps.Deck)
		}
	}
	available := len(ps.Deck)
	if available > n {
		available = n
	}
	if available == 0 {
		return nil, nil
	}
	revealed := make([]CardID, 0, available)
	// Top of deck is end of slice. Reveal from end downward.
	for i := 0; i < available; i++ {
		revealed = append(revealed, ps.Deck[len(ps.Deck)-1-i])
	}
	return revealed, nil
}

// TrashFromDeck moves a specific card from the top region of the deck
// to the trash. If the card is not present in the deck, this is a no-op
// (no events, no error).
func TrashFromDeck(gs *GameState, px PlayerIdx, card CardID) []Event {
	ps := &gs.Players[px]
	idx := indexOf(ps.Deck, card)
	if idx < 0 {
		return nil
	}
	ps.Deck = append(ps.Deck[:idx], ps.Deck[idx+1:]...)
	gs.Trash = append(gs.Trash, card)
	return []Event{{Kind: EventCardTrashed, PlayerIdx: px, CardID: card}}
}

// DiscardFromDeck moves a specific card from the top region of the deck
// to the player's discard pile. No-op if the card is not present.
func DiscardFromDeck(gs *GameState, px PlayerIdx, card CardID) []Event {
	ps := &gs.Players[px]
	idx := indexOf(ps.Deck, card)
	if idx < 0 {
		return nil
	}
	ps.Deck = append(ps.Deck[:idx], ps.Deck[idx+1:]...)
	ps.Discard = append(ps.Discard, card)
	return []Event{{Kind: EventCardDiscarded, PlayerIdx: px, CardID: card}}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/ -run "TestRevealFromDeck|TestTrashFromDeck|TestDiscardFromDeck" -v`
Expected: PASS.

Run: `go test ./internal/engine/...`
Expected: all existing tests still pass.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/primitives.go internal/engine/primitives_test.go
git commit -m "$(cat <<'EOF'
feat(engine): add RevealFromDeck, TrashFromDeck, DiscardFromDeck

Three new primitives used by Bandit in Tier 3. RevealFromDeck returns
the top n cards without moving them; TrashFromDeck/DiscardFromDeck
move a named card from the top region of the deck to trash/discard.
Shuffle-on-empty semantics match DrawCards.
EOF
)"
```

---

### Task 3: `ResolveAttackVictims` helper

**Files:**
- Create: `dominion-grpc/internal/engine/attack.go`
- Create: `dominion-grpc/internal/engine/attack_test.go`

- [ ] **Step 1: Write the failing test for `ResolveAttackVictims`**

Create `dominion-grpc/internal/engine/attack_test.go`:

```go
package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeAttack and fakeReaction are helpers for wiring a lookup that the
// tests in this file can control without registering real cards.
type fakeAttack struct{ id CardID }

func fakeLookupWith(cards ...*Card) CardLookup {
	byID := map[CardID]*Card{}
	for _, c := range cards {
		byID[c.ID] = c
	}
	return func(id CardID) (*Card, bool) {
		c, ok := byID[id]
		return c, ok
	}
}

func TestResolveAttackVictims_TwoPlayerNoReactions(t *testing.T) {
	gs := NewTestStateWithRNG("t", 1, []PlayerState{
		{Name: "a"}, {Name: "b", Hand: []CardID{"copper", "estate"}},
	})
	lookup := fakeLookupWith(
		&Card{ID: "copper", Name: "Copper"},
		&Card{ID: "estate", Name: "Estate"},
	)

	victims, events := ResolveAttackVictims(gs, 0, "witch", lookup)

	require.Equal(t, []PlayerIdx{1}, victims)
	require.Len(t, events, 1)
	require.Equal(t, EventAttackPlayed, events[0].Kind)
	require.Equal(t, PlayerIdx(0), events[0].PlayerIdx)
	require.Equal(t, CardID("witch"), events[0].CardID)
}

func TestResolveAttackVictims_FourPlayerIterationOrder(t *testing.T) {
	gs := NewTestStateWithRNG("t", 1, []PlayerState{
		{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"},
	})
	lookup := fakeLookupWith()

	victims, _ := ResolveAttackVictims(gs, 1, "witch", lookup) // attacker = seat 1

	require.Equal(t, []PlayerIdx{2, 3, 0}, victims,
		"iteration must start at (attacker+1) and wrap")
}

func TestResolveAttackVictims_MoatLikeBlockerSkipsVictim(t *testing.T) {
	blocker := &Card{
		ID: "blocker", Name: "Blocker",
		OnReaction: func(gs *GameState, victim PlayerIdx, trigger Trigger) (bool, []Event) {
			return true, []Event{{Kind: EventReactionTriggered, PlayerIdx: victim, CardID: "blocker"}}
		},
	}
	gs := NewTestStateWithRNG("t", 1, []PlayerState{
		{Name: "a"},
		{Name: "b", Hand: []CardID{"blocker", "copper"}},
		{Name: "c"},
	})
	lookup := fakeLookupWith(blocker, &Card{ID: "copper", Name: "Copper"})

	victims, events := ResolveAttackVictims(gs, 0, "witch", lookup)

	require.Equal(t, []PlayerIdx{2}, victims, "blocked victim (seat 1) must be excluded")
	require.Contains(t, events, Event{Kind: EventReactionTriggered, PlayerIdx: 1, CardID: "blocker"})
}

func TestResolveAttackVictims_AllBlocked_EmptyVictimList(t *testing.T) {
	blocker := &Card{
		ID: "blocker", Name: "Blocker",
		OnReaction: func(gs *GameState, victim PlayerIdx, trigger Trigger) (bool, []Event) {
			return true, nil
		},
	}
	gs := NewTestStateWithRNG("t", 1, []PlayerState{
		{Name: "a"},
		{Name: "b", Hand: []CardID{"blocker"}},
	})
	lookup := fakeLookupWith(blocker)

	victims, _ := ResolveAttackVictims(gs, 0, "witch", lookup)
	require.Empty(t, victims)
}

func TestResolveAttackVictims_NonReactionTriggerIgnored(t *testing.T) {
	nonBlocker := &Card{
		ID: "non_blocker", Name: "X",
		OnReaction: func(gs *GameState, victim PlayerIdx, trigger Trigger) (bool, []Event) {
			return false, nil // defines OnReaction but chooses not to block
		},
	}
	gs := NewTestStateWithRNG("t", 1, []PlayerState{
		{Name: "a"},
		{Name: "b", Hand: []CardID{"non_blocker"}},
	})
	lookup := fakeLookupWith(nonBlocker)

	victims, _ := ResolveAttackVictims(gs, 0, "witch", lookup)
	require.Equal(t, []PlayerIdx{1}, victims)
}

// Silence unused-struct warning on fakeAttack since it's part of the shared
// helper surface for upcoming card tests.
var _ = fakeAttack{}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/engine/ -run TestResolveAttackVictims -v`
Expected: FAIL — `ResolveAttackVictims` undefined.

- [ ] **Step 3: Implement `ResolveAttackVictims` in `attack.go`**

Create `dominion-grpc/internal/engine/attack.go`:

```go
package engine

// ResolveAttackVictims iterates opponents in turn order (next seat from
// attacker, wrapping), scans each victim's hand for cards with a
// non-nil OnReaction, calls each such reaction, and collects the set
// of victims that were NOT blocked.
//
// Returns:
//   - victims: seats that did not block, in turn order
//   - events: one EventAttackPlayed followed by one event per reaction
//     event returned by any card's OnReaction
//
// Does NOT create decisions. Callers (attack cards) decide per-victim
// whether a prompt is needed and stash the remaining victims in the
// decision's Context under CtxKeyRemainingVictims.
func ResolveAttackVictims(gs *GameState, attacker PlayerIdx, cardID CardID, lookup CardLookup) ([]PlayerIdx, []Event) {
	events := []Event{{Kind: EventAttackPlayed, PlayerIdx: attacker, CardID: cardID}}
	trigger := Trigger{Kind: TriggerAttackPlayed, Attacker: attacker, CardID: cardID}

	n := PlayerIdx(len(gs.Players))
	var victims []PlayerIdx
	for step := PlayerIdx(1); step < n; step++ {
		victim := (attacker + step) % n
		blocked := false
		for _, handCardID := range gs.Players[victim].Hand {
			card, ok := lookup(handCardID)
			if !ok || card.OnReaction == nil {
				continue
			}
			blocks, ev := card.OnReaction(gs, victim, trigger)
			events = append(events, ev...)
			if blocks {
				blocked = true
			}
		}
		if !blocked {
			victims = append(victims, victim)
		}
	}
	return victims, events
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/engine/ -run TestResolveAttackVictims -v`
Expected: PASS (all 5 subtests).

Run: `go test ./internal/engine/...`
Expected: all tests green.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/attack.go internal/engine/attack_test.go
git commit -m "$(cat <<'EOF'
feat(engine): add ResolveAttackVictims helper

Iterates opponents in turn order (next seat, wrapping), calls each hand
card's OnReaction against a TriggerAttackPlayed trigger, and returns
the non-blocked victim list plus collected events. Attack cards in
Tier 3 use this helper to build their per-victim queue.
EOF
)"
```

---

### Task 4: Extend `PutOnDeckPrompt` with `TypeFilter`

**Files:**
- Modify: `dominion-grpc/proto/dominion/v1/game.proto`
- Modify: `dominion-grpc/internal/engine/decision.go`
- Modify: `dominion-grpc/internal/engine/decision_test.go`
- Modify: `dominion-grpc/internal/service/translate.go`
- Modify: `dominion-grpc/internal/service/translate_test.go`
- Regenerate: `dominion-grpc/gen/go/dominion/v1/*`

- [ ] **Step 1: Write the failing engine test**

Append to `dominion-grpc/internal/engine/decision_test.go`:

```go
func TestPutOnDeckPrompt_EmptyFilter_MeansAnyCard(t *testing.T) {
	p := PutOnDeckPrompt{}
	require.Empty(t, p.TypeFilter, "default zero-value TypeFilter must be empty")
}

func TestPutOnDeckPrompt_WithTypeFilter(t *testing.T) {
	p := PutOnDeckPrompt{TypeFilter: []CardType{TypeVictory}}
	require.Equal(t, []CardType{TypeVictory}, p.TypeFilter)
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/engine/ -run TestPutOnDeckPrompt -v`
Expected: FAIL — `PutOnDeckPrompt` has no `TypeFilter` field.

- [ ] **Step 3: Add `TypeFilter` to engine struct**

In `dominion-grpc/internal/engine/decision.go`, replace the existing `PutOnDeckPrompt` struct (currently empty) with:

```go
// PutOnDeckPrompt asks the player to put one card from hand on top of
// their deck. TypeFilter narrows which hand cards are legal choices;
// an empty/nil filter means "any card."
type PutOnDeckPrompt struct {
	TypeFilter []CardType
}

func (PutOnDeckPrompt) isPrompt() {}
```

- [ ] **Step 4: Run engine tests**

Run: `go test ./internal/engine/ -run TestPutOnDeckPrompt -v`
Expected: PASS.

Run: `go test ./internal/engine/cards/ -run TestArtisan -v`
Expected: PASS — Artisan does not set `TypeFilter`, and empty filter is the "any card" default.

- [ ] **Step 5: Add `type_filter` field to proto `PutOnDeckPrompt`**

In `dominion-grpc/proto/dominion/v1/game.proto`, locate `message PutOnDeckPrompt {}` and replace with:

```proto
message PutOnDeckPrompt {
    repeated CardType type_filter = 1;  // empty = any card
}
```

- [ ] **Step 6: Regenerate proto code**

Run: `make generate`
Expected: regenerates `gen/go/dominion/v1/*.go` with the new field.

- [ ] **Step 7: Update service translation for the new field**

In `dominion-grpc/internal/service/translate.go`, locate the `PutOnDeck` case inside `PromptToProto` (the function that maps engine prompts to proto). Update it to copy `TypeFilter`:

```go
case engine.PutOnDeckPrompt:
	proto := &pb.PutOnDeckPrompt{}
	for _, t := range p.TypeFilter {
		proto.TypeFilter = append(proto.TypeFilter, cardTypeToProto(t))
	}
	pd.Prompt = &pb.Decision_PutOnDeck{PutOnDeck: proto}
```

If `cardTypeToProto` does not already exist in `translate.go`, search for an existing case that uses it (e.g., the `TrashFromHandPrompt` case already copies a `TypeFilter`) — reuse that helper.

- [ ] **Step 8: Add a translation test**

Append to `dominion-grpc/internal/service/translate_test.go`:

```go
func TestPromptToProto_PutOnDeck_WithTypeFilter(t *testing.T) {
	d := &engine.Decision{
		ID:        "d1",
		PlayerIdx: 0,
		CardID:    "bureaucrat",
		Step:      0,
		Prompt:    engine.PutOnDeckPrompt{TypeFilter: []engine.CardType{engine.TypeVictory}},
	}
	pd := service.DecisionToProto(d)
	require.NotNil(t, pd)
	put := pd.GetPutOnDeck()
	require.NotNil(t, put)
	require.Equal(t, []pb.CardType{pb.CardType_CARD_TYPE_VICTORY}, put.TypeFilter)
}

func TestPromptToProto_PutOnDeck_EmptyFilter(t *testing.T) {
	d := &engine.Decision{
		ID:        "d1",
		PlayerIdx: 0,
		CardID:    "artisan",
		Step:      1,
		Prompt:    engine.PutOnDeckPrompt{},
	}
	pd := service.DecisionToProto(d)
	require.NotNil(t, pd)
	put := pd.GetPutOnDeck()
	require.NotNil(t, put)
	require.Empty(t, put.TypeFilter)
}
```

If the test file's imports do not already include `pb` and `service`, add them. The existing Artisan-using test(s) in this file establish the pattern.

- [ ] **Step 9: Run service tests**

Run: `go test ./internal/service/ -run TestPromptToProto_PutOnDeck -v`
Expected: PASS.

Run: `go test ./...`
Expected: all green. No Artisan regression; no other tests broken.

- [ ] **Step 10: Commit**

```bash
git add proto/dominion/v1/game.proto gen/go/dominion/v1/ internal/engine/decision.go internal/engine/decision_test.go internal/service/translate.go internal/service/translate_test.go
git commit -m "$(cat <<'EOF'
feat(proto): extend PutOnDeckPrompt with optional type_filter

Additive field. Empty filter preserves Artisan's existing "any card"
semantics; Bureaucrat (coming next) will pass TypeFilter=[VICTORY].
Updates PromptToProto to plumb the filter through and adds a
translation test covering both filter and no-filter paths.
EOF
)"
```

---

### Task 5: New `TrashFromRevealedPrompt`

**Files:**
- Modify: `dominion-grpc/proto/dominion/v1/game.proto`
- Modify: `dominion-grpc/internal/engine/decision.go`
- Modify: `dominion-grpc/internal/engine/decision_test.go`
- Modify: `dominion-grpc/internal/service/translate.go`
- Modify: `dominion-grpc/internal/service/translate_test.go`
- Modify: `dominion-grpc/internal/engine/decision.go` (new context keys)

- [ ] **Step 1: Write failing engine tests**

Append to `dominion-grpc/internal/engine/decision_test.go`:

```go
func TestTrashFromRevealedPrompt_Shape(t *testing.T) {
	p := TrashFromRevealedPrompt{Cards: []CardID{"silver", "gold"}}
	require.Equal(t, []CardID{"silver", "gold"}, p.Cards)
	var _ Prompt = p // compile-time check that it implements Prompt
}

func TestContextKeys_NewInTier3(t *testing.T) {
	require.Equal(t, ContextKey("attacker"), CtxKeyAttacker)
	require.Equal(t, ContextKey("remaining_victims"), CtxKeyRemainingVictims)
	require.Equal(t, ContextKey("revealed_cards"), CtxKeyRevealedCards)
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/engine/ -run "TestTrashFromRevealedPrompt|TestContextKeys_NewInTier3" -v`
Expected: FAIL — both types and keys undefined.

- [ ] **Step 3: Add the new prompt type + context keys**

In `dominion-grpc/internal/engine/decision.go`:

Extend the context-key const block:

```go
const (
	CtxKeyTrashedCost      ContextKey = "trashed_cost"
	CtxKeyCard             ContextKey = "card"
	CtxKeyAttacker         ContextKey = "attacker"
	CtxKeyRemainingVictims ContextKey = "remaining_victims"
	CtxKeyRevealedCards    ContextKey = "revealed_cards"
)
```

Add the new prompt type near the other prompt definitions:

```go
// TrashFromRevealedPrompt asks the player to choose one card to trash
// from a pre-revealed list (e.g., Bandit's top-2 deck reveal).
type TrashFromRevealedPrompt struct {
	Cards []CardID
}

func (TrashFromRevealedPrompt) isPrompt() {}
```

- [ ] **Step 4: Run engine tests**

Run: `go test ./internal/engine/ -run "TestTrashFromRevealedPrompt|TestContextKeys_NewInTier3" -v`
Expected: PASS.

- [ ] **Step 5: Add new proto message + oneof variant**

In `dominion-grpc/proto/dominion/v1/game.proto`, extend `message Decision`'s `oneof prompt` with a new variant:

```proto
message Decision {
    string id = 1;
    int32 player_idx = 2;
    string card_id = 3;
    int32 step = 4;

    oneof prompt {
        DiscardFromHandPrompt discard_from_hand = 5;
        TrashFromHandPrompt trash_from_hand = 6;
        GainFromSupplyPrompt gain_from_supply = 7;
        ChooseFromDiscardPrompt choose_from_discard = 8;
        PutOnDeckPrompt put_on_deck = 9;
        MayPlayActionPrompt may_play_action = 10;
        TrashFromRevealedPrompt trash_from_revealed = 11;
    }
}
```

And add the new message definition below the existing prompt messages:

```proto
message TrashFromRevealedPrompt {
    repeated string cards = 1;  // the revealed cards the player may choose from
}
```

- [ ] **Step 6: Regenerate proto code**

Run: `make generate`
Expected: regenerates `gen/go/dominion/v1/*.go` with the new message.

- [ ] **Step 7: Update `PromptToProto` for the new prompt**

In `dominion-grpc/internal/service/translate.go`, add a case inside `PromptToProto`:

```go
case engine.TrashFromRevealedPrompt:
	proto := &pb.TrashFromRevealedPrompt{}
	for _, c := range p.Cards {
		proto.Cards = append(proto.Cards, string(c))
	}
	pd.Prompt = &pb.Decision_TrashFromRevealed{TrashFromRevealed: proto}
```

- [ ] **Step 8: Add a translation test**

Append to `dominion-grpc/internal/service/translate_test.go`:

```go
func TestPromptToProto_TrashFromRevealed(t *testing.T) {
	d := &engine.Decision{
		ID:        "d1",
		PlayerIdx: 1,
		CardID:    "bandit",
		Step:      0,
		Prompt:    engine.TrashFromRevealedPrompt{Cards: []engine.CardID{"silver", "gold"}},
	}
	pd := service.DecisionToProto(d)
	require.NotNil(t, pd)
	tfr := pd.GetTrashFromRevealed()
	require.NotNil(t, tfr)
	require.Equal(t, []string{"silver", "gold"}, tfr.Cards)
}
```

- [ ] **Step 9: Run tests**

Run: `go test ./internal/service/ -run TestPromptToProto_TrashFromRevealed -v`
Expected: PASS.

Run: `go test ./...`
Expected: all green.

Run: `make lint`
Expected: clean. Specifically `buf lint` and `buf breaking` pass (additive changes only).

- [ ] **Step 10: Commit**

```bash
git add proto/dominion/v1/game.proto gen/go/dominion/v1/ internal/engine/decision.go internal/engine/decision_test.go internal/service/translate.go internal/service/translate_test.go
git commit -m "$(cat <<'EOF'
feat(proto): add TrashFromRevealedPrompt + Tier 3 context keys

New oneof variant on Decision for Bandit's "pick which non-Copper
Treasure from the revealed top-2 to trash" flow. Answered with the
existing CardChoiceAnswer. Three new context keys (CtxKeyAttacker,
CtxKeyRemainingVictims, CtxKeyRevealedCards) support the per-victim
decision queue the attack cards will use.
EOF
)"
```

---

## Stage B: The five cards

Cards land in this order to respect the dependency graph: Moat first (the first `OnReaction` consumer), then Witch (the simplest attack — no decisions), then the decision-heavy attacks.

### Task 6: Moat

**Files:**
- Create: `dominion-grpc/internal/engine/cards/kingdom_moat.go`
- Create: `dominion-grpc/internal/engine/cards/kingdom_moat_test.go`

- [ ] **Step 1: Write failing tests**

Create `dominion-grpc/internal/engine/cards/kingdom_moat_test.go`:

```go
package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestMoat_Metadata(t *testing.T) {
	require.Equal(t, "Moat", Moat.Name)
	require.Equal(t, 2, Moat.Cost)
	require.True(t, Moat.HasType(engine.TypeAction))
	require.True(t, Moat.HasType(engine.TypeReaction))
}

func TestMoat_OnPlay_DrawsTwo(t *testing.T) {
	gs := newTestStateForCards(1)
	gs.Players[0].Deck = []engine.CardID{"copper", "silver", "gold"}

	events := Moat.OnPlay(gs, 0)

	require.Len(t, gs.Players[0].Hand, 2)
	require.Len(t, events, 2)
}

func TestMoat_OnReaction_AttackTrigger_Blocks(t *testing.T) {
	gs := newTestStateForCards(2)
	tr := engine.Trigger{Kind: engine.TriggerAttackPlayed, Attacker: 0, CardID: "witch"}

	blocks, events := Moat.OnReaction(gs, 1, tr)

	require.True(t, blocks)
	require.Len(t, events, 1)
	require.Equal(t, engine.EventReactionTriggered, events[0].Kind)
	require.Equal(t, engine.PlayerIdx(1), events[0].PlayerIdx)
	require.Equal(t, engine.CardID("moat"), events[0].CardID)
}

func TestMoat_OnReaction_UnknownTrigger_DoesNotBlock(t *testing.T) {
	gs := newTestStateForCards(2)
	tr := engine.Trigger{Kind: engine.TriggerUnknown, Attacker: 0, CardID: "x"}

	blocks, events := Moat.OnReaction(gs, 1, tr)

	require.False(t, blocks)
	require.Empty(t, events)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/engine/cards/ -run TestMoat -v`
Expected: FAIL — `Moat` undefined.

- [ ] **Step 3: Implement Moat**

Create `dominion-grpc/internal/engine/cards/kingdom_moat.go`:

```go
package cards

import (
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Moat = &engine.Card{
	ID:    "moat",
	Name:  "Moat",
	Cost:  2,
	Types: []engine.CardType{engine.TypeAction, engine.TypeReaction},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		return engine.DrawCards(gs, px, 2)
	},
	OnReaction: func(gs *engine.GameState, victim engine.PlayerIdx, trigger engine.Trigger) (bool, []engine.Event) {
		if trigger.Kind != engine.TriggerAttackPlayed {
			return false, nil
		}
		return true, []engine.Event{{
			Kind:      engine.EventReactionTriggered,
			PlayerIdx: victim,
			CardID:    "moat",
		}}
	},
}

func init() {
	DefaultRegistry.Register(Moat)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/engine/cards/ -run TestMoat -v`
Expected: PASS.

Run: `go test ./...`
Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/cards/kingdom_moat.go internal/engine/cards/kingdom_moat_test.go
git commit -m "$(cat <<'EOF'
feat(cards): add Moat — +2 cards, blocks attacks via OnReaction

First Tier 3 card. OnReaction blocks any TriggerAttackPlayed; other
trigger kinds are left unblocked (future reaction cards handle them).
EOF
)"
```

---

### Task 7: Witch

**Files:**
- Create: `dominion-grpc/internal/engine/cards/kingdom_witch.go`
- Create: `dominion-grpc/internal/engine/cards/kingdom_witch_test.go`

- [ ] **Step 1: Write failing tests**

Create `dominion-grpc/internal/engine/cards/kingdom_witch_test.go`:

```go
package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestWitch_Metadata(t *testing.T) {
	require.Equal(t, "Witch", Witch.Name)
	require.Equal(t, 5, Witch.Cost)
	require.True(t, Witch.HasType(engine.TypeAction))
	require.True(t, Witch.HasType(engine.TypeAttack))
}

func TestWitch_OnPlay_DrawsTwo_EachOpponentGainsCurse(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"witch"}
	gs.Players[0].Deck = []engine.CardID{"copper", "copper"}
	gs.Supply.Piles["curse"] = 10

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "witch"}, testLookup)
	require.NoError(t, err)
	require.Len(t, gs.Players[0].Hand, 2, "attacker drew 2")
	require.Contains(t, gs.Players[1].Discard, engine.CardID("curse"))
	require.Equal(t, 9, gs.Supply.Piles["curse"])
}

func TestWitch_OnPlay_MoatBlocksCurseGain(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"witch"}
	gs.Players[1].Hand = []engine.CardID{"moat"}
	gs.Supply.Piles["curse"] = 10

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "witch"}, testLookup)
	require.NoError(t, err)
	require.NotContains(t, gs.Players[1].Discard, engine.CardID("curse"),
		"Moat-holding opponent must not receive a Curse")
	require.Equal(t, 10, gs.Supply.Piles["curse"])
}

func TestWitch_OnPlay_EmptyCursePile_NoEffect(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"witch"}
	gs.Supply.Piles["curse"] = 0

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "witch"}, testLookup)
	require.NoError(t, err)
	require.NotContains(t, gs.Players[1].Discard, engine.CardID("curse"))
}

func TestWitch_OnPlay_ThreePlayer_BothOpponentsGainCurses(t *testing.T) {
	gs := newTestStateForCards(3)
	gs.Phase = engine.PhaseAction
	gs.CurrentPlayer = 0
	gs.Players[0].Actions = 1
	gs.Players[0].Hand = []engine.CardID{"witch"}
	gs.Supply.Piles["curse"] = 10

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "witch"}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[1].Discard, engine.CardID("curse"))
	require.Contains(t, gs.Players[2].Discard, engine.CardID("curse"))
	require.Equal(t, 8, gs.Supply.Piles["curse"])
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/engine/cards/ -run TestWitch -v`
Expected: FAIL — `Witch` undefined.

- [ ] **Step 3: Add a package-level `registryLookup` helper**

Witch's `OnPlay` needs a `CardLookup` to scan opponents' hands for reactions, but `Card.OnPlay`'s signature only provides `(gs, px)`. Rather than extending the signature — a cross-cutting change affecting every existing card — capture `DefaultRegistry` at the cards-package level. Every card that might be in an opponent's hand is registered there, so the lookup is complete for reaction-scanning purposes.

Create `dominion-grpc/internal/engine/cards/registry_lookup.go`:

```go
package cards

import (
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

// registryLookup is the CardLookup function backed by DefaultRegistry.
// Used by Tier 3 attack cards that need to scan opponents' hands for
// reactions inside OnPlay (where the engine does not pass lookup in).
func registryLookup(id engine.CardID) (*engine.Card, bool) {
	return DefaultRegistry.Lookup(id)
}
```

- [ ] **Step 4: Implement Witch**

Create `dominion-grpc/internal/engine/cards/kingdom_witch.go`:

```go
package cards

import (
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Witch = &engine.Card{
	ID:    "witch",
	Name:  "Witch",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction, engine.TypeAttack},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.DrawCards(gs, px, 2)
		victims, attackEvents := engine.ResolveAttackVictims(gs, px, "witch", registryLookup)
		events = append(events, attackEvents...)
		for _, v := range victims {
			events = append(events, engine.GainCard(gs, v, "curse", engine.GainToDiscard)...)
		}
		return events
	},
}

func init() {
	DefaultRegistry.Register(Witch)
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/engine/cards/ -run TestWitch -v`
Expected: PASS.

Run: `go test ./...`
Expected: all green. Moat's test still passes (Moat itself isn't an attacker, so it doesn't use `registryLookup`).

- [ ] **Step 6: Commit**

```bash
git add internal/engine/cards/registry_lookup.go internal/engine/cards/kingdom_witch.go internal/engine/cards/kingdom_witch_test.go
git commit -m "$(cat <<'EOF'
feat(cards): add Witch — +2 cards, each other player gains a Curse

Also introduces a package-level registryLookup helper for attack cards
that need to scan opponents' hands for reactions inside OnPlay. Moat
auto-blocks via the existing OnReaction path; empty Curse pile
silently no-ops per GainCard semantics.
EOF
)"
```

---

### Task 8: Militia

**Files:**
- Create: `dominion-grpc/internal/engine/cards/kingdom_militia.go`
- Create: `dominion-grpc/internal/engine/cards/kingdom_militia_test.go`

- [ ] **Step 1: Write failing tests**

Create `dominion-grpc/internal/engine/cards/kingdom_militia_test.go`:

```go
package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestMilitia_Metadata(t *testing.T) {
	require.Equal(t, "Militia", Militia.Name)
	require.Equal(t, 4, Militia.Cost)
	require.True(t, Militia.HasType(engine.TypeAction))
	require.True(t, Militia.HasType(engine.TypeAttack))
}

func TestMilitia_OnPlay_AddsCoinsAndPromptsOpponent(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"militia"}
	gs.Players[1].Hand = []engine.CardID{"copper", "copper", "estate", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "militia"}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 2, gs.Players[0].Coins)

	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, engine.PlayerIdx(1), gs.PendingDecision.PlayerIdx)
	p, ok := gs.PendingDecision.Prompt.(engine.DiscardFromHandPrompt)
	require.True(t, ok)
	require.Equal(t, 2, p.Min) // 5-card hand, discard to 3
	require.Equal(t, 2, p.Max)
}

func TestMilitia_OnPlay_OpponentWithThreeOrFewer_NoDecision(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"militia"}
	gs.Players[1].Hand = []engine.CardID{"copper", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "militia"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
}

func TestMilitia_Resolve_OpponentDiscardsChosenCards(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"militia"}
	gs.Players[1].Hand = []engine.CardID{"copper", "copper", "estate", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "militia"}, testLookup)
	require.NoError(t, err)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardListAnswer{Cards: []engine.CardID{"estate", "estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Len(t, gs.Players[1].Hand, 3)
	require.Contains(t, gs.Players[1].Discard, engine.CardID("estate"))
}

func TestMilitia_Resolve_WrongCount_Error(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"militia"}
	gs.Players[1].Hand = []engine.CardID{"copper", "copper", "estate", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "militia"}, testLookup)
	require.NoError(t, err)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardListAnswer{Cards: []engine.CardID{"estate"}}, // only 1, need 2
	}, testLookup)
	require.Error(t, err)
}

func TestMilitia_OnPlay_MoatBlocksOpponent(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"militia"}
	gs.Players[1].Hand = []engine.CardID{"moat", "copper", "copper", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "militia"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision, "Moat-holding opponent must not be prompted")
	require.Equal(t, 2, gs.Players[0].Coins, "+2 coins still happens for the attacker")
}

func TestMilitia_ThreePlayer_QueuesBothOpponents(t *testing.T) {
	gs := newTestStateForCards(3)
	gs.Phase = engine.PhaseAction
	gs.CurrentPlayer = 0
	gs.Players[0].Actions = 1
	gs.Players[0].Hand = []engine.CardID{"militia"}
	gs.Players[1].Hand = []engine.CardID{"copper", "copper", "estate", "estate", "silver"}
	gs.Players[2].Hand = []engine.CardID{"copper", "copper", "gold", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "militia"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, engine.PlayerIdx(1), gs.PendingDecision.PlayerIdx, "opp at seat 1 is prompted first")

	// Resolve opp 1
	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardListAnswer{Cards: []engine.CardID{"estate", "estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision, "opp 2 must now be prompted")
	require.Equal(t, engine.PlayerIdx(2), gs.PendingDecision.PlayerIdx)

	// Resolve opp 2
	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  2,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardListAnswer{Cards: []engine.CardID{"copper", "copper"}},
	}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision, "all opponents resolved")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/engine/cards/ -run TestMilitia -v`
Expected: FAIL — `Militia` undefined.

- [ ] **Step 3: Implement Militia**

Create `dominion-grpc/internal/engine/cards/kingdom_militia.go`:

```go
package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Militia = &engine.Card{
	ID:    "militia",
	Name:  "Militia",
	Cost:  4,
	Types: []engine.CardType{engine.TypeAction, engine.TypeAttack},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.AddCoins(gs, px, 2)
		victims, attackEvents := engine.ResolveAttackVictims(gs, px, "militia", registryLookup)
		events = append(events, attackEvents...)
		events = append(events, militiaQueue(gs, px, victims)...)
		return events
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		cards := answer.(engine.CardListAnswer).Cards
		expected := len(gs.Players[px].Hand) - 3
		if expected < 0 {
			expected = 0
		}
		if len(cards) != expected {
			return nil, fmt.Errorf("militia: must discard exactly %d cards, got %d", expected, len(cards))
		}
		events := engine.DiscardFromHand(gs, px, cards)
		attacker := d.Context[engine.CtxKeyAttacker].(engine.PlayerIdx)
		remaining := d.Context[engine.CtxKeyRemainingVictims].([]engine.PlayerIdx)
		events = append(events, militiaQueue(gs, attacker, remaining)...)
		return events, nil
	},
}

// militiaQueue advances through the victim list, skipping victims whose
// hand is already at or below 3 cards and parking a decision on the
// first victim who needs to discard. Returns only the RequestDecision
// event (if any); caller accumulates additional events.
func militiaQueue(gs *engine.GameState, attacker engine.PlayerIdx, victims []engine.PlayerIdx) []engine.Event {
	for i, v := range victims {
		handSize := len(gs.Players[v].Hand)
		if handSize <= 3 {
			continue
		}
		n := handSize - 3
		rest := append([]engine.PlayerIdx(nil), victims[i+1:]...)
		return engine.RequestDecision(gs, v, "militia", 0,
			engine.DiscardFromHandPrompt{Min: n, Max: n},
			map[engine.ContextKey]any{
				engine.CtxKeyAttacker:         attacker,
				engine.CtxKeyRemainingVictims: rest,
			})
	}
	return nil
}

func init() {
	DefaultRegistry.Register(Militia)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/engine/cards/ -run TestMilitia -v`
Expected: PASS — all 6 subtests.

Run: `go test ./...`
Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/cards/kingdom_militia.go internal/engine/cards/kingdom_militia_test.go
git commit -m "$(cat <<'EOF'
feat(cards): add Militia — +2 coins, each other discards to 3

Introduces the per-victim decision-queue pattern: OnPlay creates the
first victim's decision and stashes remaining victims in Context;
OnResolve processes the answer and parks the next. Victims with
≤3-card hands are skipped inline; Moat blocks via OnReaction.
EOF
)"
```

---

### Task 9: Bureaucrat

**Files:**
- Create: `dominion-grpc/internal/engine/cards/kingdom_bureaucrat.go`
- Create: `dominion-grpc/internal/engine/cards/kingdom_bureaucrat_test.go`

- [ ] **Step 1: Write failing tests**

Create `dominion-grpc/internal/engine/cards/kingdom_bureaucrat_test.go`:

```go
package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestBureaucrat_Metadata(t *testing.T) {
	require.Equal(t, "Bureaucrat", Bureaucrat.Name)
	require.Equal(t, 4, Bureaucrat.Cost)
	require.True(t, Bureaucrat.HasType(engine.TypeAction))
	require.True(t, Bureaucrat.HasType(engine.TypeAttack))
}

func TestBureaucrat_OnPlay_GainsSilverToDeck(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[0].Deck, engine.CardID("silver"))
	require.Equal(t, 9, gs.Supply.Piles["silver"])
}

func TestBureaucrat_OnPlay_SilverPileEmpty_AttackStillRuns(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 0
	gs.Players[1].Hand = []engine.CardID{"estate", "copper"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)
	// 1 Victory (estate) → forced put-on-deck, no prompt
	require.Contains(t, gs.Players[1].Deck, engine.CardID("estate"))
}

func TestBureaucrat_OnPlay_OpponentZeroVictories_NoDecision(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10
	gs.Players[1].Hand = []engine.CardID{"copper", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.NotContains(t, gs.Players[1].Deck, engine.CardID("estate"))
}

func TestBureaucrat_OnPlay_OpponentOneVictory_ForcedNoDecision(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10
	gs.Players[1].Hand = []engine.CardID{"copper", "estate", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Contains(t, gs.Players[1].Deck, engine.CardID("estate"),
		"the sole Victory is auto-moved to top of deck")
	require.NotContains(t, gs.Players[1].Hand, engine.CardID("estate"))
}

func TestBureaucrat_OnPlay_OpponentMultipleVictories_Prompts(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10
	gs.Players[1].Hand = []engine.CardID{"estate", "duchy", "copper"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, engine.PlayerIdx(1), gs.PendingDecision.PlayerIdx)
	p, ok := gs.PendingDecision.Prompt.(engine.PutOnDeckPrompt)
	require.True(t, ok)
	require.Contains(t, p.TypeFilter, engine.TypeVictory)
}

func TestBureaucrat_Resolve_PutsChosenVictoryOnDeck(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10
	gs.Players[1].Hand = []engine.CardID{"estate", "duchy", "copper"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardListAnswer{Cards: []engine.CardID{"estate"}},
	}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Contains(t, gs.Players[1].Deck, engine.CardID("estate"))
	require.NotContains(t, gs.Players[1].Hand, engine.CardID("estate"))
	require.Contains(t, gs.Players[1].Hand, engine.CardID("duchy"),
		"the other Victory stays in hand")
}

func TestBureaucrat_Resolve_NonVictoryAnswer_Error(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10
	gs.Players[1].Hand = []engine.CardID{"estate", "duchy", "copper"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardListAnswer{Cards: []engine.CardID{"copper"}}, // not a Victory
	}, testLookup)
	require.Error(t, err)
}

func TestBureaucrat_OnPlay_MoatBlocksOpponent(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bureaucrat"}
	gs.Supply.Piles["silver"] = 10
	gs.Players[1].Hand = []engine.CardID{"moat", "estate", "estate"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bureaucrat"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.NotContains(t, gs.Players[1].Deck, engine.CardID("estate"))
	require.Contains(t, gs.Players[0].Deck, engine.CardID("silver"),
		"attacker's own Silver gain to deck still happens")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/engine/cards/ -run TestBureaucrat -v`
Expected: FAIL — `Bureaucrat` undefined.

- [ ] **Step 3: Implement Bureaucrat**

Create `dominion-grpc/internal/engine/cards/kingdom_bureaucrat.go`:

```go
package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Bureaucrat = &engine.Card{
	ID:    "bureaucrat",
	Name:  "Bureaucrat",
	Cost:  4,
	Types: []engine.CardType{engine.TypeAction, engine.TypeAttack},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.GainCard(gs, px, "silver", engine.GainToDeck)
		victims, attackEvents := engine.ResolveAttackVictims(gs, px, "bureaucrat", registryLookup)
		events = append(events, attackEvents...)
		events = append(events, bureaucratQueue(gs, px, victims)...)
		return events
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		chosen := answer.(engine.CardListAnswer).Cards
		if len(chosen) != 1 {
			return nil, fmt.Errorf("bureaucrat: must choose exactly 1 Victory, got %d", len(chosen))
		}
		card, ok := lookup(chosen[0])
		if !ok {
			return nil, fmt.Errorf("bureaucrat: unknown card %q", chosen[0])
		}
		if !card.HasType(engine.TypeVictory) {
			return nil, fmt.Errorf("bureaucrat: chosen card %q is not a Victory", chosen[0])
		}
		if engine.IndexOf(gs.Players[px].Hand, chosen[0]) < 0 {
			return nil, fmt.Errorf("bureaucrat: card %q not in hand", chosen[0])
		}
		events := engine.PutOnDeck(gs, px, chosen)
		attacker := d.Context[engine.CtxKeyAttacker].(engine.PlayerIdx)
		remaining := d.Context[engine.CtxKeyRemainingVictims].([]engine.PlayerIdx)
		events = append(events, bureaucratQueue(gs, attacker, remaining)...)
		return events, nil
	},
}

// bureaucratQueue advances through the victim list, handling 0-Victory
// (reveal only) and 1-Victory (forced put-on-deck) cases inline and
// parking a decision on the first victim with 2+ Victories.
func bureaucratQueue(gs *engine.GameState, attacker engine.PlayerIdx, victims []engine.PlayerIdx) []engine.Event {
	var events []engine.Event
	for i, v := range victims {
		victories := victoriesInHand(gs, v)
		switch len(victories) {
		case 0:
			events = append(events, engine.Event{
				Kind:      engine.EventCardRevealed,
				PlayerIdx: v,
			})
		case 1:
			events = append(events, engine.PutOnDeck(gs, v, victories)...)
		default:
			rest := append([]engine.PlayerIdx(nil), victims[i+1:]...)
			events = append(events, engine.RequestDecision(gs, v, "bureaucrat", 0,
				engine.PutOnDeckPrompt{TypeFilter: []engine.CardType{engine.TypeVictory}},
				map[engine.ContextKey]any{
					engine.CtxKeyAttacker:         attacker,
					engine.CtxKeyRemainingVictims: rest,
				})...)
			return events
		}
	}
	return events
}

func victoriesInHand(gs *engine.GameState, px engine.PlayerIdx) []engine.CardID {
	var out []engine.CardID
	for _, id := range gs.Players[px].Hand {
		card, ok := DefaultRegistry.Lookup(id)
		if !ok {
			continue
		}
		if card.HasType(engine.TypeVictory) {
			out = append(out, id)
		}
	}
	return out
}

func init() {
	DefaultRegistry.Register(Bureaucrat)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/engine/cards/ -run TestBureaucrat -v`
Expected: PASS — all 9 subtests.

Run: `go test ./...`
Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/cards/kingdom_bureaucrat.go internal/engine/cards/kingdom_bureaucrat_test.go
git commit -m "$(cat <<'EOF'
feat(cards): add Bureaucrat — gain Silver to deck, victims top-deck a Victory

0 Victories → reveal-only event. 1 Victory → auto-top-deck, no prompt.
2+ Victories → PutOnDeckPrompt with TypeFilter=[VICTORY]. Empty Silver
pile silently skips the gain while the attack still runs.
EOF
)"
```

---

### Task 10: Bandit

**Files:**
- Create: `dominion-grpc/internal/engine/cards/kingdom_bandit.go`
- Create: `dominion-grpc/internal/engine/cards/kingdom_bandit_test.go`

- [ ] **Step 1: Write failing tests**

Create `dominion-grpc/internal/engine/cards/kingdom_bandit_test.go`:

```go
package cards

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

func TestBandit_Metadata(t *testing.T) {
	require.Equal(t, "Bandit", Bandit.Name)
	require.Equal(t, 5, Bandit.Cost)
	require.True(t, Bandit.HasType(engine.TypeAction))
	require.True(t, Bandit.HasType(engine.TypeAttack))
}

func TestBandit_OnPlay_GainsGoldToDiscard(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)
	require.Contains(t, gs.Players[0].Discard, engine.CardID("gold"))
	require.Equal(t, 29, gs.Supply.Piles["gold"])
}

func TestBandit_OnPlay_OpponentNoNonCopperTreasures_AutoDiscardsBoth(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Deck = []engine.CardID{"estate", "copper"} // copper is top

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Empty(t, gs.Players[1].Deck)
	require.ElementsMatch(t, []engine.CardID{"estate", "copper"}, gs.Players[1].Discard)
	require.Empty(t, gs.Trash)
}

func TestBandit_OnPlay_OpponentOneNonCopperTreasure_AutoTrashes(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Deck = []engine.CardID{"estate", "silver"} // silver on top

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Contains(t, gs.Trash, engine.CardID("silver"))
	require.Contains(t, gs.Players[1].Discard, engine.CardID("estate"))
}

func TestBandit_OnPlay_OpponentTwoNonCopperTreasures_Prompts(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Deck = []engine.CardID{"silver", "gold"} // gold on top

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision)
	p, ok := gs.PendingDecision.Prompt.(engine.TrashFromRevealedPrompt)
	require.True(t, ok)
	require.ElementsMatch(t, []engine.CardID{"silver", "gold"}, p.Cards)
}

func TestBandit_Resolve_TrashesChosenAndDiscardsRest(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Deck = []engine.CardID{"silver", "gold"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardChoiceAnswer{Card: "gold"},
	}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Contains(t, gs.Trash, engine.CardID("gold"))
	require.Contains(t, gs.Players[1].Discard, engine.CardID("silver"))
	require.Empty(t, gs.Players[1].Deck)
}

func TestBandit_Resolve_CardNotInRevealed_Error(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Deck = []engine.CardID{"silver", "gold"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)

	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  1,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardChoiceAnswer{Card: "estate"}, // not in revealed
	}, testLookup)
	require.Error(t, err)
}

func TestBandit_OnPlay_OpponentDeckEmpty_ShufflesOrSkips(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Deck = nil
	gs.Players[1].Discard = nil

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision, "no cards to reveal; no prompt")
}

func TestBandit_OnPlay_MoatBlocksOpponent(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"bandit"}
	gs.Supply.Piles["gold"] = 30
	gs.Players[1].Hand = []engine.CardID{"moat"}
	gs.Players[1].Deck = []engine.CardID{"silver", "gold"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "bandit"}, testLookup)
	require.NoError(t, err)
	require.Nil(t, gs.PendingDecision)
	require.Equal(t, []engine.CardID{"silver", "gold"}, gs.Players[1].Deck,
		"blocked opponent's deck must be untouched")
	require.Empty(t, gs.Trash)
	require.Contains(t, gs.Players[0].Discard, engine.CardID("gold"),
		"attacker still gains the Gold")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/engine/cards/ -run TestBandit -v`
Expected: FAIL — `Bandit` undefined.

- [ ] **Step 3: Implement Bandit**

Create `dominion-grpc/internal/engine/cards/kingdom_bandit.go`:

```go
package cards

import (
	"fmt"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
)

var Bandit = &engine.Card{
	ID:    "bandit",
	Name:  "Bandit",
	Cost:  5,
	Types: []engine.CardType{engine.TypeAction, engine.TypeAttack},
	OnPlay: func(gs *engine.GameState, px engine.PlayerIdx) []engine.Event {
		events := engine.GainCard(gs, px, "gold", engine.GainToDiscard)
		victims, attackEvents := engine.ResolveAttackVictims(gs, px, "bandit", registryLookup)
		events = append(events, attackEvents...)
		events = append(events, banditQueue(gs, px, victims)...)
		return events
	},
	OnResolve: func(gs *engine.GameState, px engine.PlayerIdx, d *engine.Decision, answer engine.Answer, lookup engine.CardLookup) ([]engine.Event, error) {
		choice := answer.(engine.CardChoiceAnswer)
		revealed := d.Context[engine.CtxKeyRevealedCards].([]engine.CardID)
		if !containsCard(revealed, choice.Card) {
			return nil, fmt.Errorf("bandit: card %q not in revealed list", choice.Card)
		}
		card, ok := lookup(choice.Card)
		if !ok {
			return nil, fmt.Errorf("bandit: unknown card %q", choice.Card)
		}
		if !card.HasType(engine.TypeTreasure) || choice.Card == "copper" {
			return nil, fmt.Errorf("bandit: chosen card %q is not a non-Copper Treasure", choice.Card)
		}
		events := engine.TrashFromDeck(gs, px, choice.Card)
		for _, c := range revealed {
			if c == choice.Card {
				continue
			}
			events = append(events, engine.DiscardFromDeck(gs, px, c)...)
		}
		attacker := d.Context[engine.CtxKeyAttacker].(engine.PlayerIdx)
		remaining := d.Context[engine.CtxKeyRemainingVictims].([]engine.PlayerIdx)
		events = append(events, banditQueue(gs, attacker, remaining)...)
		return events, nil
	},
}

// banditQueue advances through the victim list. For each victim:
// reveal top 2; zero non-Copper Treasures revealed → discard both;
// one → auto-trash, discard rest; two → prompt.
func banditQueue(gs *engine.GameState, attacker engine.PlayerIdx, victims []engine.PlayerIdx) []engine.Event {
	var events []engine.Event
	for i, v := range victims {
		revealed, revealEvents := engine.RevealFromDeck(gs, v, 2)
		events = append(events, revealEvents...)
		events = append(events, engine.Event{Kind: engine.EventCardRevealed, PlayerIdx: v})

		treasures := nonCopperTreasures(revealed)
		switch len(treasures) {
		case 0:
			for _, c := range revealed {
				events = append(events, engine.DiscardFromDeck(gs, v, c)...)
			}
		case 1:
			events = append(events, engine.TrashFromDeck(gs, v, treasures[0])...)
			for _, c := range revealed {
				if c == treasures[0] {
					continue
				}
				events = append(events, engine.DiscardFromDeck(gs, v, c)...)
			}
		default:
			rest := append([]engine.PlayerIdx(nil), victims[i+1:]...)
			events = append(events, engine.RequestDecision(gs, v, "bandit", 0,
				engine.TrashFromRevealedPrompt{Cards: treasures},
				map[engine.ContextKey]any{
					engine.CtxKeyAttacker:         attacker,
					engine.CtxKeyRemainingVictims: rest,
					engine.CtxKeyRevealedCards:    revealed,
				})...)
			return events
		}
	}
	return events
}

func nonCopperTreasures(cards []engine.CardID) []engine.CardID {
	var out []engine.CardID
	for _, id := range cards {
		if id == "copper" {
			continue
		}
		card, ok := DefaultRegistry.Lookup(id)
		if !ok {
			continue
		}
		if card.HasType(engine.TypeTreasure) {
			out = append(out, id)
		}
	}
	return out
}

func containsCard(cards []engine.CardID, target engine.CardID) bool {
	for _, c := range cards {
		if c == target {
			return true
		}
	}
	return false
}

func init() {
	DefaultRegistry.Register(Bandit)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/engine/cards/ -run TestBandit -v`
Expected: PASS — all 9 subtests.

Run: `go test ./...`
Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add internal/engine/cards/kingdom_bandit.go internal/engine/cards/kingdom_bandit_test.go
git commit -m "$(cat <<'EOF'
feat(cards): add Bandit — gain Gold, each other trashes non-Copper Treasure

Reveal-top-2 flow: 0 non-Copper Treasures revealed → auto-discard both;
1 → auto-trash, discard other; 2 → TrashFromRevealedPrompt. Empty Gold
pile silently skips the gain. Moat blocks via OnReaction.
EOF
)"
```

---

## Stage C: Bot strategies + integration sweeps

### Task 11: WitchBM strategy + done-criterion sweep

**Files:**
- Modify: `dominion-grpc/internal/bot/strategy.go` (extend `safeRefusal` + `cardCost` for Tier 3 prompts and costs)
- Create: `dominion-grpc/internal/bot/witch_bm.go`
- Create: `dominion-grpc/internal/bot/witch_bm_test.go`
- Modify: `dominion-grpc/internal/bot/integration_test.go`
- Modify: `dominion-grpc/cmd/bot/main.go`

- [ ] **Step 1: Update `cardCost` and `safeRefusal` in `strategy.go`**

In `dominion-grpc/internal/bot/strategy.go`, extend `cardCost`'s switch with Tier 3 costs:

```go
func cardCost(id string) int {
	switch id {
	case "copper", "curse":
		return 0
	case "estate", "moat":
		return 2
	case "silver", "cellar", "chapel":
		return 3
	case "harbinger", "vassal", "workshop":
		return 3
	case "militia", "bureaucrat", "moneylender", "poacher", "remodel", "smithy":
		return 4
	case "mine", "witch", "bandit", "laboratory", "market", "festival":
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

Extend `safeRefusal`'s switch with the new prompt types:

```go
case *pb.Decision_TrashFromRevealed:
	// Pick the cheapest listed card. The list only ever contains
	// non-Copper Treasures, so "cheapest" gives up the less valuable
	// treasure.
	tfr := d.GetTrashFromRevealed()
	if len(tfr.Cards) == 0 {
		r.Answer = &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{None: true}}
		break
	}
	best := tfr.Cards[0]
	bestCost := cardCost(best)
	for _, c := range tfr.Cards[1:] {
		if cost := cardCost(c); cost < bestCost {
			best, bestCost = c, cost
		}
	}
	r.Answer = &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{Card: best}}
```

Also update the existing `case *pb.Decision_PutOnDeck:` branch — when a `TypeFilter` is present, pick the first hand card matching the filter. Replace that branch with:

```go
case *pb.Decision_PutOnDeck:
	put := d.GetPutOnDeck()
	card := firstInHandMatching(cs, put.TypeFilter)
	r.Answer = &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{Cards: []string{card}}}
```

**Note the answer-type change.** Bureaucrat answers `PutOnDeckPrompt` with a `CardListAnswer` containing exactly one card (see Bureaucrat's `OnResolve` in Task 9). Prior Artisan usage used `CardChoiceAnswer`. The engine must accept both answer types for `PutOnDeckPrompt` — verify this by checking Artisan's OnResolve after editing `strategy.go`. If Artisan still expects `CardChoiceAnswer`, update safe-refusal to emit whichever type that card's OnResolve reads. Concretely: Artisan's OnResolve is in `internal/engine/cards/kingdom_artisan.go` — check it.

If Artisan reads `CardChoiceAnswer`, the simplest resolution is: Bureaucrat's `OnResolve` already reads `CardListAnswer`, so keep safe-refusal emitting `CardListAnswer` specifically when the prompt has a `TypeFilter`, else `CardChoiceAnswer`. Update `safeRefusal` branch:

```go
case *pb.Decision_PutOnDeck:
	put := d.GetPutOnDeck()
	if len(put.TypeFilter) > 0 {
		card := firstInHandMatching(cs, put.TypeFilter)
		r.Answer = &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{Cards: []string{card}}}
	} else {
		card := firstInHand(cs)
		r.Answer = &pb.ResolveDecision_CardChoice{CardChoice: &pb.CardChoiceAnswer{Card: card}}
	}
```

Add a helper alongside `firstInHand`:

```go
func firstInHandMatching(cs *ClientState, filter []pb.CardType) string {
	me := cs.MyPlayer()
	if me == nil || len(me.Hand) == 0 {
		return ""
	}
	if len(filter) == 0 {
		return me.Hand[0]
	}
	for _, c := range me.Hand {
		if cardHasAnyType(c, filter) {
			return c
		}
	}
	return me.Hand[0]
}

func cardHasAnyType(id string, filter []pb.CardType) bool {
	for _, t := range filter {
		if t == pb.CardType_CARD_TYPE_VICTORY && isVictory(id) {
			return true
		}
		if t == pb.CardType_CARD_TYPE_TREASURE && isTreasure(id) {
			return true
		}
	}
	return false
}

func isVictory(id string) bool {
	switch id {
	case "estate", "duchy", "province":
		return true
	}
	return false
}
```

`isTreasure` already exists in `bigmoney.go` — reuse it.

- [ ] **Step 2: Write failing WitchBM tests**

Create `dominion-grpc/internal/bot/witch_bm_test.go`:

```go
package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

func TestWitchBM_Name(t *testing.T) {
	require.Equal(t, "witch_bm", NewWitchBM().Name())
}

func TestWitchBM_BuysWitchOnTurn3_WhenCoinsAtLeast5(t *testing.T) {
	s := NewWitchBM()
	cs := &ClientState{
		Me:           0,
		Phase:        pb.Phase_PHASE_BUY,
		MyTurnsTaken: 3,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Coins: 5, Buys: 1},
				{PlayerIdx: 1},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "witch", Count: 10},
				{CardId: "province", Count: 8},
				{CardId: "gold", Count: 30},
				{CardId: "silver", Count: 40},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.Equal(t, "witch", action.GetBuyCard().CardId)
}

func TestWitchBM_DoesNotBuySecondWitch(t *testing.T) {
	s := NewWitchBM()
	s.witchOwned = true
	cs := &ClientState{
		Me:           0,
		Phase:        pb.Phase_PHASE_BUY,
		MyTurnsTaken: 3,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Coins: 5, Buys: 1},
				{PlayerIdx: 1},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "witch", Count: 10},
				{CardId: "province", Count: 8},
				{CardId: "gold", Count: 30},
				{CardId: "silver", Count: 40},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.NotEqual(t, "witch", action.GetBuyCard().CardId,
		"must not buy a second witch")
}

func TestWitchBM_DoesNotBuyWitchAfterTurn4(t *testing.T) {
	s := NewWitchBM()
	cs := &ClientState{
		Me:           0,
		Phase:        pb.Phase_PHASE_BUY,
		MyTurnsTaken: 5,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Coins: 5, Buys: 1},
				{PlayerIdx: 1},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "witch", Count: 10},
				{CardId: "province", Count: 8},
				{CardId: "gold", Count: 30},
				{CardId: "silver", Count: 40},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.NotEqual(t, "witch", action.GetBuyCard().CardId)
}

func TestWitchBM_PlaysWitchWhenActionsAvailable(t *testing.T) {
	s := NewWitchBM()
	cs := &ClientState{
		Me:    0,
		Phase: pb.Phase_PHASE_ACTION,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Actions: 1, Hand: []string{"witch", "copper"}},
				{PlayerIdx: 1},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.Equal(t, "witch", action.GetPlayCard().CardId)
}

func TestWitchBM_Resolve_UsesSafeRefusal(t *testing.T) {
	s := NewWitchBM()
	cs := &ClientState{Me: 0, Snapshot: &pb.GameStateSnapshot{
		Players: []*pb.PlayerView{{PlayerIdx: 0, Hand: []string{"copper"}}, {PlayerIdx: 1}},
	}}
	d := &pb.Decision{
		Id: "d1", PlayerIdx: 0,
		Prompt: &pb.Decision_DiscardFromHand{DiscardFromHand: &pb.DiscardFromHandPrompt{Min: 0, Max: 0}},
	}
	r := s.Resolve(cs, d)
	require.Equal(t, "d1", r.DecisionId)
}
```

- [ ] **Step 3: Run tests to verify failure**

Run: `go test ./internal/bot/ -run TestWitchBM -v`
Expected: FAIL — `NewWitchBM` undefined.

- [ ] **Step 4: Implement WitchBM**

Create `dominion-grpc/internal/bot/witch_bm.go`:

```go
package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// WitchBM is Big Money plus Witch. Buys exactly one Witch on turns 3-4
// when $5+ is available, then reverts to Big Money economy. The attack
// is the strategy's value; a second Witch displaces treasure buys without
// increasing Curse output meaningfully.
type WitchBM struct {
	witchOwned bool
}

func NewWitchBM() *WitchBM { return &WitchBM{} }

func (w *WitchBM) Name() string { return "witch_bm" }

func (w *WitchBM) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}

	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		if me.Actions >= 1 && handContains(me.Hand, "witch") {
			return playCard(cs.Me, "witch")
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
		earlyTurn := cs.MyTurnsTaken >= 3 && cs.MyTurnsTaken <= 4
		switch {
		case me.Coins >= 8 && supplyCount(cs.Snapshot, "province") > 0:
			return buyCard(cs.Me, "province")
		case me.Coins >= 6 && supplyCount(cs.Snapshot, "gold") > 0:
			return buyCard(cs.Me, "gold")
		case me.Coins >= 5 && earlyTurn && !w.witchOwned && supplyCount(cs.Snapshot, "witch") > 0:
			w.witchOwned = true
			return buyCard(cs.Me, "witch")
		case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
			return buyCard(cs.Me, "silver")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

func (w *WitchBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	return safeRefusal(cs, d)
}
```

- [ ] **Step 5: Run WitchBM unit tests**

Run: `go test ./internal/bot/ -run TestWitchBM -v`
Expected: PASS.

- [ ] **Step 6: Register `witch_bm` in `cmd/bot/main.go`**

Read `dominion-grpc/cmd/bot/main.go` to find the strategy dispatch (look for `bigmoney`, `smithy_bm`, `chapel_bm`, `remodel_bm`). Add a case:

```go
case "witch_bm":
	strat = bot.NewWitchBM()
```

If the existing code uses a `StrategyByName`-style helper in `internal/bot/strategy.go`, add the case there instead. Run `grep -rn "smithy_bm" dominion-grpc/cmd/ dominion-grpc/internal/bot/` first to locate the exact pattern.

- [ ] **Step 7: Add the done-criterion sweep**

Append to `dominion-grpc/internal/bot/integration_test.go`:

```go
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
```

- [ ] **Step 8: Run the full suite**

Run: `make test`
Expected: all green, including the new 200-game sweep. Budget ~10s for the WitchBM sweep; total additional time well under a minute.

If the sweep fails because the rate is below 0.55: confirm determinism (run with the same seeds again — it should be bit-identical), then inspect a specific failing seed using the per-seed `t.Run` pattern from `TestBotVsBot_SmithyBM_Outperforms_BigMoney` to debug. The spec flags 0.55 as safe against an expected ~0.70–0.80 rate; a sustained failure almost certainly means an engine bug.

- [ ] **Step 9: Commit**

```bash
git add internal/bot/strategy.go internal/bot/witch_bm.go internal/bot/witch_bm_test.go internal/bot/integration_test.go cmd/bot/main.go
git commit -m "$(cat <<'EOF'
feat(bot): add WitchBM + Tier 3 done-criterion sweep

WitchBM buys exactly one Witch on turns 3-4 at $5+, then reverts to
Big Money. 200-game sweep vs BigMoney with kingdom=[witch] and seats
alternated by seed parity; threshold ≥0.55 (expected rate ~0.70-0.80).
Extends safeRefusal for TrashFromRevealedPrompt and filter-aware
PutOnDeckPrompt; updates cardCost for Tier 3 cards.
EOF
)"
```

---

### Task 12: MilitiaBM + smoke sweep + random-kingdom pool update

**Files:**
- Create: `dominion-grpc/internal/bot/militia_bm.go`
- Create: `dominion-grpc/internal/bot/militia_bm_test.go`
- Modify: `dominion-grpc/internal/bot/integration_test.go`
- Modify: `dominion-grpc/cmd/bot/main.go`

- [ ] **Step 1: Write failing MilitiaBM tests**

Create `dominion-grpc/internal/bot/militia_bm_test.go`:

```go
package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

func TestMilitiaBM_Name(t *testing.T) {
	require.Equal(t, "militia_bm", NewMilitiaBM().Name())
}

func TestMilitiaBM_BuysMilitiaOnTurn3(t *testing.T) {
	s := NewMilitiaBM()
	cs := &ClientState{
		Me:           0,
		Phase:        pb.Phase_PHASE_BUY,
		MyTurnsTaken: 3,
		Snapshot: &pb.GameStateSnapshot{
			Players: []*pb.PlayerView{
				{PlayerIdx: 0, Coins: 4, Buys: 1},
				{PlayerIdx: 1},
			},
			Supply: []*pb.SupplyPile{
				{CardId: "militia", Count: 10},
				{CardId: "province", Count: 8},
				{CardId: "gold", Count: 30},
				{CardId: "silver", Count: 40},
			},
		},
	}

	action := s.PickAction(cs)
	require.NotNil(t, action)
	require.Equal(t, "militia", action.GetBuyCard().CardId)
}

func TestMilitiaBM_Resolve_DiscardsToThree_KeepsBestTreasures(t *testing.T) {
	s := NewMilitiaBM()
	cs := &ClientState{Me: 1, Snapshot: &pb.GameStateSnapshot{
		Players: []*pb.PlayerView{
			{PlayerIdx: 0},
			{PlayerIdx: 1, Hand: []string{"gold", "silver", "copper", "estate", "curse"}},
		},
	}}
	d := &pb.Decision{
		Id: "d1", PlayerIdx: 1, CardId: "militia",
		Prompt: &pb.Decision_DiscardFromHand{DiscardFromHand: &pb.DiscardFromHandPrompt{Min: 2, Max: 2}},
	}

	r := s.Resolve(cs, d)

	require.NotNil(t, r)
	cards := r.GetCardList().Cards
	require.Len(t, cards, 2)
	// Must keep gold, silver, copper (the three best). Must discard estate+curse.
	require.Contains(t, cards, "estate")
	require.Contains(t, cards, "curse")
}

func TestMilitiaBM_Resolve_OtherPromptTypes_UsesSafeRefusal(t *testing.T) {
	s := NewMilitiaBM()
	cs := &ClientState{Me: 1, Snapshot: &pb.GameStateSnapshot{
		Players: []*pb.PlayerView{{PlayerIdx: 0}, {PlayerIdx: 1, Hand: []string{"copper"}}},
	}}
	d := &pb.Decision{
		Id: "d1", PlayerIdx: 1,
		Prompt: &pb.Decision_TrashFromHand{TrashFromHand: &pb.TrashFromHandPrompt{Min: 0, Max: 1}},
	}
	r := s.Resolve(cs, d)
	require.Equal(t, "d1", r.DecisionId)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/bot/ -run TestMilitiaBM -v`
Expected: FAIL — `NewMilitiaBM` undefined.

- [ ] **Step 3: Implement MilitiaBM**

Create `dominion-grpc/internal/bot/militia_bm.go`:

```go
package bot

import (
	"sort"

	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// MilitiaBM is Big Money plus Militia. Buys one Militia on turns 3-4
// at $4+, then reverts to Big Money. When attacked by Militia (as the
// victim), keeps the highest-value 3 cards; discards the rest.
type MilitiaBM struct {
	militiaOwned bool
}

func NewMilitiaBM() *MilitiaBM { return &MilitiaBM{} }

func (m *MilitiaBM) Name() string { return "militia_bm" }

func (m *MilitiaBM) PickAction(cs *ClientState) *pb.Action {
	me := cs.MyPlayer()
	if me == nil {
		return endPhase(cs.Me)
	}

	switch cs.Phase {
	case pb.Phase_PHASE_ACTION:
		if me.Actions >= 1 && handContains(me.Hand, "militia") {
			return playCard(cs.Me, "militia")
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
		earlyTurn := cs.MyTurnsTaken >= 3 && cs.MyTurnsTaken <= 4
		switch {
		case me.Coins >= 8 && supplyCount(cs.Snapshot, "province") > 0:
			return buyCard(cs.Me, "province")
		case me.Coins >= 6 && supplyCount(cs.Snapshot, "gold") > 0:
			return buyCard(cs.Me, "gold")
		case me.Coins >= 4 && earlyTurn && !m.militiaOwned && supplyCount(cs.Snapshot, "militia") > 0:
			m.militiaOwned = true
			return buyCard(cs.Me, "militia")
		case me.Coins >= 3 && supplyCount(cs.Snapshot, "silver") > 0:
			return buyCard(cs.Me, "silver")
		}
		return endPhase(cs.Me)
	}
	return endPhase(cs.Me)
}

func (m *MilitiaBM) Resolve(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	if d.GetDiscardFromHand() != nil && d.CardId == "militia" {
		return m.resolveMilitiaDiscard(cs, d)
	}
	return safeRefusal(cs, d)
}

func (m *MilitiaBM) resolveMilitiaDiscard(cs *ClientState, d *pb.Decision) *pb.ResolveDecision {
	me := cs.MyPlayer()
	prompt := d.GetDiscardFromHand()
	n := int(prompt.Min)

	// Rank each hand card by keep-value descending; sort a copy so we
	// don't mutate the snapshot's slice.
	hand := append([]string(nil), me.Hand...)
	sort.SliceStable(hand, func(i, j int) bool {
		return militiaKeepValue(hand[i]) > militiaKeepValue(hand[j])
	})

	// Discard the worst n.
	toDiscard := hand[len(hand)-n:]
	return &pb.ResolveDecision{
		DecisionId: d.Id, PlayerIdx: d.PlayerIdx,
		Answer: &pb.ResolveDecision_CardList{CardList: &pb.CardListAnswer{Cards: toDiscard}},
	}
}

// militiaKeepValue ranks cards for Militia's discard-to-3 attack.
// Higher = keep. Gold > Silver > Copper > (dead VP/Curse).
func militiaKeepValue(id string) int {
	switch id {
	case "gold":
		return 100
	case "silver":
		return 90
	case "copper":
		return 80
	case "curse":
		return -10
	case "estate", "duchy", "province":
		return 0
	}
	// Unknown cards (Tier 1/2/3 kingdom cards the bot never buys) sit
	// between treasures and dead VP.
	return 50
}
```

- [ ] **Step 4: Run MilitiaBM tests**

Run: `go test ./internal/bot/ -run TestMilitiaBM -v`
Expected: PASS.

- [ ] **Step 5: Register `militia_bm` in `cmd/bot/main.go`** (same location as Task 11 Step 6)

```go
case "militia_bm":
	strat = bot.NewMilitiaBM()
```

- [ ] **Step 6: Add the smoke sweep and extend the random-kingdom pool**

Append to `dominion-grpc/internal/bot/integration_test.go`:

```go
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
```

- [ ] **Step 7: Extend the random-kingdom sweep**

The Tier 2 plan added (or preserved) a test that picks random kingdom subsets for bot-vs-bot coverage. Locate it — search for `TestBotVsBot_RandomKingdoms` or the `randomKingdomSubset` helper:

```bash
grep -rn "RandomKingdoms\|randomKingdom" dominion-grpc/internal/bot/
```

If the test exists and defines a kingdom pool (likely a `[]string` of card IDs), append the Tier 3 IDs to that pool: `"moat", "militia", "bureaucrat", "witch", "bandit"`.

If no such test exists yet (Tier 2 may have deferred it), skip this step and note in the commit that the random-kingdom pool extension is deferred to whenever that test lands.

- [ ] **Step 8: Run the full suite**

Run: `make test`
Expected: all green, including the new 50-game MilitiaBM sweep.

- [ ] **Step 9: Verify the linter and proto-compat gates**

Run: `make lint`
Expected: `buf lint` clean, `buf breaking` clean, `golangci-lint` clean.

Run: `buf breaking --against '.git#branch=main,subdir=proto' proto`
Expected: no breaking changes reported (all proto edits in this plan are additive).

- [ ] **Step 10: Commit**

```bash
git add internal/bot/militia_bm.go internal/bot/militia_bm_test.go internal/bot/integration_test.go cmd/bot/main.go
git commit -m "$(cat <<'EOF'
feat(bot): add MilitiaBM + Tier 3 smoke sweep

MilitiaBM buys one Militia on turns 3-4 at $4+. Resolve keeps the
highest-value 3 cards (Gold > Silver > Copper > unknown > VP > Curse).
50-game smoke sweep vs BigMoney with kingdom=[militia] asserts termination
and at least one win (engine-bug canary, not a strength claim). If the
random-kingdom sweep exists, its pool is extended to include Tier 3 IDs.
EOF
)"
```

---

## Milestone exit

- All 12 tasks above green.
- `make test` passes: unit tests + property sweep + WitchBM 200-game sweep + MilitiaBM 50-game sweep + existing Tier 0/1/2 sweeps.
- `make lint` clean.
- `make generate` produces no uncommitted diff.
- `buf breaking` reports no breaking changes.
- `make bot ARGS="-strategy witch_bm ..."` and `make bot ARGS="-strategy militia_bm ..."` work against a running `make server`.

---

## Plan dependencies summary

```
Task 1 OnReaction+Trigger ─┐
                           ├── Task 3 ResolveAttackVictims ──┐
Task 2 Reveal/Trash/       │                                 │
  DiscardFromDeck ─────────┤                                 ├── Task 6 Moat
                           │                                 ├── Task 7 Witch ── Task 11 WitchBM + sweep
Task 4 PutOnDeckPrompt     │                                 ├── Task 8 Militia ── Task 12 MilitiaBM + smoke sweep
  TypeFilter ──────────────┤                                 ├── Task 9 Bureaucrat
                           │                                 └── Task 10 Bandit
Task 5 TrashFromRevealed ──┘
  Prompt
```

Stage A (1–5) can be reviewed and merged in any order as long as all land before Stage B. Stage B cards depend on Stage A but are otherwise independent — Moat must land before Witch/Militia/Bureaucrat/Bandit can be exercised end-to-end against a Moat-holding opponent, but the test fixtures don't depend on another card's code. Task 11 depends on Task 7 (Witch); Task 12 depends on Task 8 (Militia).
