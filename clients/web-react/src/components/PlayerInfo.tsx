import type { PlayerView } from '@@gen/dominion/v1/game_pb'

interface Props {
  player: PlayerView
  isCurrentPlayer: boolean
}

export default function PlayerInfo({ player, isCurrentPlayer }: Props) {
  return (
    <div
      data-testid={`player-info-${player.playerIdx}`}
      className={[
        'flex items-center gap-3 px-3 py-1.5 rounded-lg text-sm font-serif',
        isCurrentPlayer ? 'bg-sol-yellow/10 border border-sol-yellow/30' : '',
      ].join(' ')}
    >
      <span className="font-bold text-sol-base02">{player.name || `Player ${player.playerIdx}`}</span>
      <span className="text-sol-base1 text-xs font-mono">
        deck:{player.deckSize} discard:{player.discardSize} hand:{player.handSize}
      </span>
      {isCurrentPlayer && (
        <span className="text-sol-yellow text-xs font-mono tracking-wide">▶ their turn</span>
      )}
    </div>
  )
}
