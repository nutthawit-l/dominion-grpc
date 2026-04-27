import { client } from '../client'
import { Phase } from '@@gen/dominion/v1/common_pb'
import type { PlayerView } from '@@gen/dominion/v1/game_pb'

interface Props {
  player: PlayerView
  phase: Phase
  currentPlayer: number
  myIdx: number
  gameId: string
  ended: boolean
}

export default function ActionBar({ player, phase, currentPlayer, myIdx, gameId, ended }: Props) {
  const isMyTurn = currentPlayer === myIdx

  const phaseLabel =
    phase === Phase.ACTION ? 'Action' : phase === Phase.BUY ? 'Buy' : 'Cleanup'

  async function endPhase() {
    if (!isMyTurn || ended) return
    await client.submitAction({
      gameId,
      action: { kind: { case: 'endPhase', value: { playerIdx: myIdx } } },
    })
  }

  return (
    <div
      data-testid="action-bar"
      className="flex items-center gap-4 px-4 py-2 text-sm font-mono text-sol-base00"
    >
      <span className="text-sol-base1 uppercase tracking-widest text-xs">
        Phase: <span className="text-sol-base01 font-bold">{phaseLabel}</span>
      </span>
      <span>Actions: <span className="text-sol-yellow font-bold">{player.actions}</span></span>
      <span>Buys: <span className="text-sol-cyan font-bold">{player.buys}</span></span>
      <span>Coins: <span className="text-sol-green font-bold">${player.coins}</span></span>

      {ended ? (
        <span className="ml-auto text-sol-base1 italic">Game over</span>
      ) : isMyTurn ? (
        <button
          data-testid="end-phase-btn"
          onClick={endPhase}
          className="ml-auto px-4 py-1.5 bg-sol-base01 hover:bg-sol-base02 text-sol-base3 rounded-lg text-xs uppercase tracking-widest font-sans font-bold transition-colors"
        >
          End Phase
        </button>
      ) : (
        <span className="ml-auto text-sol-base1 italic">Waiting for opponent…</span>
      )}
    </div>
  )
}
