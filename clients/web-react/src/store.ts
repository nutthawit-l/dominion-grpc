import { createStore as createZustandStore } from 'zustand/vanilla'
import { useStore } from 'zustand'
import { client } from './client'
import type {
  GameStateSnapshot,
  StreamGameEventsResponse,
  Decision,
} from '@@gen/dominion/v1/game_pb'

export interface LogEntry {
  seq: bigint
  at: Date
  text: string
}

export interface GameStoreState {
  gameId: string | null
  myIdx: number | null
  snapshot: GameStateSnapshot | null
  log: LogEntry[]
  streamSeq: bigint
  selectedCards: string[]
  decisionModalOpen: boolean

  setGame: (gameId: string, myIdx: number) => void
  applyEvent: (evt: StreamGameEventsResponse) => void
  toggleCardSelection: (cardId: string) => void
  confirmDecision: () => Promise<void>
  answerYesNo: (yes: boolean) => Promise<void>
  clearGame: () => void
}

function deriveLogEntries(
  prev: GameStateSnapshot | null,
  next: GameStateSnapshot,
  seq: bigint,
): LogEntry[] {
  const entries: LogEntry[] = []
  const at = new Date()

  if (!prev || prev.turn !== next.turn) {
    entries.push({ seq, at, text: `Turn ${next.turn} — Player ${next.currentPlayer}'s turn` })
  }

  return entries
}

function resolveAnswer(decision: Decision, selectedCards: string[]) {
  switch (decision.prompt.case) {
    case 'discardFromHand':
    case 'trashFromHand':
    case 'trashFromRevealed':
      return { case: 'cardList' as const, value: { cards: selectedCards } }
    case 'gainFromSupply':
    case 'chooseFromDiscard':
    case 'putOnDeck':
      return {
        case: 'cardChoice' as const,
        value: { card: selectedCards[0] ?? '', none: selectedCards.length === 0 },
      }
    default:
      return undefined
  }
}

export const gameStore = createZustandStore<GameStoreState>((set, get) => ({
  gameId: null,
  myIdx: null,
  snapshot: null,
  log: [],
  streamSeq: 0n,
  selectedCards: [],
  decisionModalOpen: false,

  setGame: (gameId, myIdx) => set({ gameId, myIdx }),

  applyEvent: (evt) => {
    const seq = evt.sequence
    const { streamSeq, log, snapshot: prev, myIdx } = get()

    if (seq > streamSeq + 1n) {
      console.warn(`[store] sequence gap: expected ${streamSeq + 1n}, got ${seq}`)
    }

    switch (evt.kind.case) {
      case 'snapshot': {
        const snap = evt.kind.value
        const entries = deriveLogEntries(prev, snap, seq)
        const hasPending = snap.pendingDecision !== undefined
        const isMyDecision = hasPending && snap.pendingDecision!.playerIdx === myIdx
        const wasOpen = get().decisionModalOpen
        set({
          snapshot: snap,
          streamSeq: seq,
          log: [...log, ...entries],
          decisionModalOpen: isMyDecision,
          selectedCards: isMyDecision && !wasOpen ? [] : get().selectedCards,
        })
        break
      }
      case 'ended': {
        const winners = evt.kind.value.winners
        const text =
          winners.length === 0
            ? 'Game over — draw'
            : `Game over — Player ${winners[0]} wins`
        set({
          streamSeq: seq,
          log: [...log, { seq, at: new Date(), text }],
        })
        break
      }
    }
  },

  toggleCardSelection: (cardId) => {
    const { selectedCards } = get()
    set({
      selectedCards: selectedCards.includes(cardId)
        ? selectedCards.filter((c) => c !== cardId)
        : [...selectedCards, cardId],
    })
  },

  confirmDecision: async () => {
    const { snapshot, myIdx, gameId, selectedCards } = get()
    if (!snapshot?.pendingDecision || gameId === null || myIdx === null) return
    const decision = snapshot.pendingDecision
    const answer = resolveAnswer(decision, selectedCards)
    if (!answer) return
    set({ decisionModalOpen: false, selectedCards: [] })
    await client.submitAction({
      gameId,
      action: {
        kind: {
          case: 'resolve',
          value: { decisionId: decision.id, playerIdx: myIdx, answer },
        },
      },
    })
  },

  answerYesNo: async (yes: boolean) => {
    const { snapshot, myIdx, gameId } = get()
    if (!snapshot?.pendingDecision || gameId === null || myIdx === null) return
    const decision = snapshot.pendingDecision
    set({ decisionModalOpen: false, selectedCards: [] })
    await client.submitAction({
      gameId,
      action: {
        kind: {
          case: 'resolve',
          value: {
            decisionId: decision.id,
            playerIdx: myIdx,
            answer: { case: 'yesNo', value: { yes } },
          },
        },
      },
    })
  },

  clearGame: () =>
    set({
      gameId: null,
      myIdx: null,
      snapshot: null,
      log: [],
      streamSeq: 0n,
      selectedCards: [],
      decisionModalOpen: false,
    }),
}))

export function useGameStore<T>(selector: (s: GameStoreState) => T): T {
  return useStore(gameStore, selector)
}
