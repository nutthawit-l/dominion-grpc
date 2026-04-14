---
name: New card
about: Implement a single Dominion card
title: "card: <Name>"
labels: area:engine, kind:card
---

## Card: <Name>

**Tier:** <0-5>
**Cost:** <int>
**Types:** <Action/Treasure/Victory/Attack/Reaction>

### Rulebook text
> <verbatim from 2nd edition rulebook>

### Primitives used
- [ ] DrawCards
- [ ] AddActions
- [ ]

### Prompts used (if any)
- [ ] DiscardFromHandPrompt
- [ ]

### Unit tests required
- [ ] Happy path
- [ ] Empty deck / empty discard
- [ ] Decision-pending guard (if applicable)
- [ ] Resource guards (Actions/Buys/Coins)
- [ ] Game-ended guard

### Definition of done
- [ ] Card file in `internal/engine/cards/`
- [ ] Registered via `init()` → `cards.Register()`
- [ ] Unit tests pass
- [ ] Included in the Tier N integration sweep
- [ ] No new engine primitives needed (or: new primitive issue linked)
