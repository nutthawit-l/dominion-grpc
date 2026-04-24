import { describe, it, expect, beforeEach } from 'vitest'
import { gameStore } from '../store'
import type { StreamGameEventsResponse, GameStateSnapshot } from '@@gen/dominion/v1/game_pb'
import { Phase } from '@@gen/dominion/v1/common_pb'

function makeSnapshot(overrides: Partial<GameStateSnapshot> = {}): GameStateSnapshot {
  return {
    $typeName: 'dominion.v1.GameStateSnapshot',
    gameId: 'g1',
    seed: 42n,
    turn: 1,
    currentPlayer: 0,
    phase: Phase.ACTION,
    players: [],
    supply: [],
    trashSize: 0,
    pendingDecision: undefined,
    ended: false,
    winners: [],
    ...overrides,
  } as GameStateSnapshot
}

function snapshotEvt(snap: GameStateSnapshot, seq = 1n): StreamGameEventsResponse {
  return {
    $typeName: 'dominion.v1.StreamGameEventsResponse',
    sequence: seq,
    at: undefined,
    kind: { case: 'snapshot', value: snap },
  } as unknown as StreamGameEventsResponse
}

function endedEvt(seq = 5n): StreamGameEventsResponse {
  return {
    $typeName: 'dominion.v1.StreamGameEventsResponse',
    sequence: seq,
    at: undefined,
    kind: { case: 'ended', value: { $typeName: 'dominion.v1.GameEnded', winners: [1], finalScores: {} } },
  } as unknown as StreamGameEventsResponse
}

beforeEach(() => {
  gameStore.getState().clearGame()
})

describe('applyEvent – snapshot', () => {
  it('stores snapshot and bumps streamSeq', () => {
    gameStore.getState().setGame('g1', 0)
    gameStore.getState().applyEvent(snapshotEvt(makeSnapshot(), 3n))
    const s = gameStore.getState()
    expect(s.snapshot?.gameId).toBe('g1')
    expect(s.streamSeq).toBe(3n)
  })

  it('appends a log entry when turn changes', () => {
    gameStore.getState().setGame('g1', 0)
    gameStore.getState().applyEvent(snapshotEvt(makeSnapshot({ turn: 1 }), 1n))
    gameStore.getState().applyEvent(snapshotEvt(makeSnapshot({ turn: 2 }), 2n))
    expect(gameStore.getState().log.length).toBeGreaterThanOrEqual(2)
  })

  it('opens decision modal when pendingDecision.playerIdx === myIdx', () => {
    gameStore.getState().setGame('g1', 0)
    const snap = makeSnapshot({
      pendingDecision: {
        $typeName: 'dominion.v1.Decision',
        id: 'd1', playerIdx: 0, cardId: 'militia', step: 0,
        prompt: { case: 'discardFromHand', value: { $typeName: 'dominion.v1.DiscardFromHandPrompt', min: 0, max: 3 } },
      } as any,
    })
    gameStore.getState().applyEvent(snapshotEvt(snap, 1n))
    expect(gameStore.getState().decisionModalOpen).toBe(true)
  })

  it('does not open modal when pendingDecision.playerIdx !== myIdx', () => {
    gameStore.getState().setGame('g1', 0)
    const snap = makeSnapshot({
      pendingDecision: {
        $typeName: 'dominion.v1.Decision',
        id: 'd1', playerIdx: 1, cardId: 'militia', step: 0,
        prompt: { case: 'discardFromHand', value: { $typeName: 'dominion.v1.DiscardFromHandPrompt', min: 0, max: 3 } },
      } as any,
    })
    gameStore.getState().applyEvent(snapshotEvt(snap, 1n))
    expect(gameStore.getState().decisionModalOpen).toBe(false)
  })

  it('closes modal when pendingDecision is cleared', () => {
    gameStore.getState().setGame('g1', 0)
    const snapWith = makeSnapshot({
      pendingDecision: {
        $typeName: 'dominion.v1.Decision',
        id: 'd1', playerIdx: 0, cardId: 'chapel', step: 0,
        prompt: { case: 'trashFromHand', value: { $typeName: 'dominion.v1.TrashFromHandPrompt', min: 0, max: 4, typeFilter: [], cardFilter: [] } },
      } as any,
    })
    gameStore.getState().applyEvent(snapshotEvt(snapWith, 1n))
    expect(gameStore.getState().decisionModalOpen).toBe(true)

    const snapWithout = makeSnapshot({ pendingDecision: undefined })
    gameStore.getState().applyEvent(snapshotEvt(snapWithout, 2n))
    expect(gameStore.getState().decisionModalOpen).toBe(false)
  })
})

describe('applyEvent – ended', () => {
  it('appends game-over log entry', () => {
    gameStore.getState().setGame('g1', 0)
    gameStore.getState().applyEvent(endedEvt(5n))
    const last = gameStore.getState().log.at(-1)
    expect(last?.text).toMatch(/game over/i)
    expect(gameStore.getState().streamSeq).toBe(5n)
  })
})

describe('toggleCardSelection', () => {
  it('toggles card in and out of selectedCards', () => {
    gameStore.getState().toggleCardSelection('copper')
    expect(gameStore.getState().selectedCards).toContain('copper')
    gameStore.getState().toggleCardSelection('copper')
    expect(gameStore.getState().selectedCards).not.toContain('copper')
  })
})

describe('clearGame', () => {
  it('resets all state', () => {
    gameStore.getState().setGame('g1', 0)
    gameStore.getState().applyEvent(snapshotEvt(makeSnapshot(), 1n))
    gameStore.getState().clearGame()
    const s = gameStore.getState()
    expect(s.gameId).toBeNull()
    expect(s.snapshot).toBeNull()
    expect(s.log).toHaveLength(0)
  })
})
