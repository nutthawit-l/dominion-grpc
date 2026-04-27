import { client } from '../client'
import { getCard } from '../cards'
import { Phase } from '@@gen/dominion/v1/common_pb'
import type { SupplyPile } from '@@gen/dominion/v1/card_pb'
import type { GameStateSnapshot } from '@@gen/dominion/v1/game_pb'

interface Props {
  supply: SupplyPile[]
  myIdx: number
  snapshot: GameStateSnapshot
}

export default function SupplyGrid({ supply, myIdx, snapshot }: Props) {
  const isMyTurn = snapshot.currentPlayer === myIdx
  const isBuyPhase = snapshot.phase === Phase.BUY
  const myPlayer = snapshot.players.find((_, i) => i === myIdx)
  const coins = myPlayer?.coins ?? 0
  const buys = myPlayer?.buys ?? 0

  async function buy(cardId: string) {
    if (!isMyTurn || !isBuyPhase || buys === 0) return
    const card = getCard(cardId)
    if (card.cost > coins) return
    await client.submitAction({
      gameId: snapshot.gameId,
      action: { kind: { case: 'buyCard', value: { playerIdx: myIdx, cardId } } },
    })
  }

  const basics = supply.filter((p) => !getCard(p.cardId).isKingdom)
  const kingdom = supply.filter((p) => getCard(p.cardId).isKingdom)

  function PileCard({ pile }: { pile: SupplyPile }) {
    const card = getCard(pile.cardId)
    const affordable = isMyTurn && isBuyPhase && buys > 0 && card.cost <= coins && pile.count > 0
    return (
      <button
        data-testid={`supply-${pile.cardId}`}
        onClick={() => buy(pile.cardId)}
        disabled={!affordable}
        title={`${card.name} — $${card.cost}`}
        className={[
          'relative rounded overflow-hidden border-2 transition-all w-14',
          affordable
            ? 'border-sol-yellow/70 shadow-md hover:scale-105 cursor-pointer'
            : 'border-sol-base1/20 opacity-70 cursor-default',
        ].join(' ')}
      >
        <img src={`/cards/${pile.cardId}.jpg`} alt={card.name} className="w-full h-auto block" />
        <div className="absolute bottom-0 right-0 bg-sol-base3/85 text-sol-base02 text-[9px] font-mono font-black px-1 rounded-tl">
          {pile.count}
        </div>
      </button>
    )
  }

  return (
    <div data-testid="supply-grid" className="space-y-3">
      <div>
        <p className="text-xs uppercase tracking-widest text-sol-base1 mb-1">Basics</p>
        <div className="flex flex-wrap gap-2">
          {basics.map((p) => <PileCard key={p.cardId} pile={p} />)}
        </div>
      </div>
      <div>
        <p className="text-xs uppercase tracking-widest text-sol-base1 mb-1">Kingdom</p>
        <div className="flex flex-wrap gap-2">
          {kingdom.map((p) => <PileCard key={p.cardId} pile={p} />)}
        </div>
      </div>
    </div>
  )
}
