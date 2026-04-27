import { useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { client } from '../client'
import { gameStore, useGameStore } from '../store'
import SupplyGrid from '../components/SupplyGrid'
import PlayerInfo from '../components/PlayerInfo'
import GameLog from '../components/GameLog'
import ActionBar from '../components/ActionBar'
import CardAreaCanvas from '../components/CardAreaCanvas'
import DecisionModal from '../components/DecisionModal'

export default function GameTable() {
  const { gameId } = useParams<{ gameId: string }>()
  const navigate = useNavigate()
  const myIdx = useGameStore((s) => s.myIdx)
  const snapshot = useGameStore((s) => s.snapshot)
  const log = useGameStore((s) => s.log)
  const streamSeqRef = useRef(0n)
  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    if (!gameId || myIdx === null) return

    async function subscribe() {
      const ac = new AbortController()
      abortRef.current = ac
      streamSeqRef.current = 0n
      try {
        const stream = client.streamGameEvents(
          { gameId: gameId!, playerIdx: myIdx! },
          { signal: ac.signal },
        )
        for await (const evt of stream) {
          const seq = evt.sequence
          // Only gap-check after the first non-initial event establishes a
          // baseline. The initial snapshot always carries seq=0; the first
          // subsequent event may be at any seq if the bot acted first.
          if (streamSeqRef.current > 0n && seq > streamSeqRef.current + 1n) {
            console.warn('[stream] sequence gap — resubscribing')
            ac.abort()
            subscribe()
            return
          }
          streamSeqRef.current = seq
          gameStore.getState().applyEvent(evt)
        }
      } catch (e: unknown) {
        if (!ac.signal.aborted) {
          console.error('[stream] error', e)
        }
      }
    }

    subscribe()

    return () => {
      abortRef.current?.abort()
    }
  }, [gameId, myIdx])

  if (!gameId || myIdx === null) {
    return (
      <div className="min-h-screen bg-sol-base3 flex items-center justify-center font-serif text-sol-base00">
        <p>
          No active game.{' '}
          <button className="text-sol-blue underline" onClick={() => navigate('/')}>
            Back to lobby
          </button>
        </p>
      </div>
    )
  }

  const opponents = snapshot?.players.filter((_, i) => i !== myIdx) ?? []
  const me = snapshot?.players.find((_, i) => i === myIdx)

  return (
    <div className="h-screen bg-sol-base3 flex flex-col overflow-hidden font-serif">
      {/* Opponent info bar */}
      <div className="flex gap-2 px-3 py-2 bg-sol-base2 border-b border-sol-base1/20">
        {opponents.map((p) => (
          <PlayerInfo key={p.playerIdx} player={p} isCurrentPlayer={snapshot?.currentPlayer === p.playerIdx} />
        ))}
      </div>

      {/* Middle: supply + log */}
      <div className="flex flex-1 min-h-0 gap-0">
        <div className="flex-1 overflow-y-auto p-3">
          {snapshot && <SupplyGrid supply={snapshot.supply} myIdx={myIdx} snapshot={snapshot} />}
        </div>
        <div className="w-56 border-l border-sol-base1/20 overflow-y-auto p-2 flex flex-col">
          <GameLog entries={log} />
        </div>
      </div>

      {/* Card area (PixiJS canvas) */}
      <div className="bg-sol-base2 border-t border-sol-base1/20 h-44">
        <CardAreaCanvas />
      </div>

      {/* Action bar */}
      <div className="bg-sol-base2 border-t border-sol-base1/20">
        {me && snapshot && (
          <ActionBar
            player={me}
            phase={snapshot.phase}
            currentPlayer={snapshot.currentPlayer}
            myIdx={myIdx}
            gameId={gameId}
            ended={snapshot.ended}
          />
        )}
      </div>

      {/* Decision modal */}
      <DecisionModal gameId={gameId} myIdx={myIdx} />
    </div>
  )
}
