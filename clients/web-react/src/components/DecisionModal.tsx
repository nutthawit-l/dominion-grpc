import { useEffect, useRef } from 'react'
import { Application } from 'pixi.js'
import { useGameStore } from '../store'
import { DecisionScene } from '../pixi/DecisionScene'
import { loadTextures } from '../pixi/textures'
import type { Decision } from '@@gen/dominion/v1/game_pb'
import { getCard } from '../cards'

interface Props {
  gameId: string
  myIdx: number
}

function promptLabel(decision: Decision): string {
  switch (decision.prompt.case) {
    case 'discardFromHand': {
      const p = decision.prompt.value
      return `Discard down to ${p.min} cards (discard ${p.max} max)`
    }
    case 'trashFromHand': {
      const p = decision.prompt.value
      return `Trash ${p.min === 0 ? 'up to' : ''} ${p.max} card${p.max !== 1 ? 's' : ''} from your hand`
    }
    case 'gainFromSupply': {
      const p = decision.prompt.value
      return `Gain a card costing up to $${p.maxCost}`
    }
    case 'chooseFromDiscard':
      return 'Choose a card from your discard pile'
    case 'putOnDeck':
      return 'Put a card from your hand onto your deck'
    case 'mayPlayAction':
      return `May play ${getCard(decision.prompt.value.cardId ?? '').name}?`
    case 'trashFromRevealed':
      return 'Trash one of the revealed cards'
    default:
      return 'Make a decision'
  }
}

function eligibleCards(decision: Decision, hand: string[], supply: { cardId: string; count: number }[]): string[] {
  switch (decision.prompt.case) {
    case 'discardFromHand':
    case 'trashFromHand':
    case 'putOnDeck':
      return hand
    case 'gainFromSupply': {
      const p = decision.prompt.value
      return supply
        .filter((s) => s.count > 0 && getCard(s.cardId).cost <= p.maxCost)
        .map((s) => s.cardId)
    }
    case 'chooseFromDiscard':
      return decision.prompt.value.cards
    case 'trashFromRevealed':
      return decision.prompt.value.cards
    default:
      return []
  }
}

export default function DecisionModal({ myIdx }: Props) {
  const decisionModalOpen = useGameStore((s) => s.decisionModalOpen)
  const snapshot = useGameStore((s) => s.snapshot)
  const selectedCards = useGameStore((s) => s.selectedCards)
  const toggleCardSelection = useGameStore((s) => s.toggleCardSelection)
  const confirmDecision = useGameStore((s) => s.confirmDecision)
  const answerYesNo = useGameStore((s) => s.answerYesNo)

  const canvasRef = useRef<HTMLCanvasElement>(null)
  const sceneRef = useRef<DecisionScene | null>(null)
  const appRef = useRef<Application | null>(null)

  const decision = snapshot?.pendingDecision
  const me = snapshot?.players.find((_, i) => i === myIdx)
  const hand = me?.hand ?? []
  const supply = snapshot?.supply ?? []

  const eligible = decision ? eligibleCards(decision, hand, supply) : []
  const isYesNo = decision?.prompt.case === 'mayPlayAction'

  useEffect(() => {
    if (!decisionModalOpen) return
    let app: Application
    async function init() {
      await loadTextures()
      app = new Application()
      await app.init({
        canvas: canvasRef.current!,
        width: Math.min(window.innerWidth - 64, eligible.length * 88 + 16),
        height: 140,
        backgroundAlpha: 0,
        antialias: true,
      })
      appRef.current = app
      const scene = new DecisionScene(app)
      sceneRef.current = scene
      scene.onCardClick(toggleCardSelection)
      scene.update(eligible, selectedCards)
    }
    init()
    return () => {
      appRef.current?.destroy(true)
      appRef.current = null
      sceneRef.current = null
    }
  }, [decisionModalOpen]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (sceneRef.current) {
      sceneRef.current.update(eligible, selectedCards)
    }
  }, [eligible.join(','), selectedCards.join(',')]) // eslint-disable-line react-hooks/exhaustive-deps

  if (!decisionModalOpen || !decision) return null

  return (
    <div
      data-testid="decision-modal"
      className="fixed inset-0 z-50 flex items-center justify-center bg-sol-base03/60"
    >
      <div className="bg-sol-base3 border border-sol-base1/30 rounded-xl shadow-2xl px-6 py-5 max-w-2xl w-full mx-4 font-serif">
        <h3 className="text-sol-base02 font-bold text-lg mb-1">Decision Required</h3>
        <p className="text-sol-base0 text-sm mb-4">{promptLabel(decision)}</p>

        {!isYesNo && eligible.length > 0 && (
          <div className="mb-4 flex justify-center overflow-x-auto">
            <canvas ref={canvasRef} data-testid="decision-canvas" />
          </div>
        )}

        <div className="flex gap-3 justify-end">
          {isYesNo ? (
            <>
              <button
                data-testid="decision-no-btn"
                onClick={() => answerYesNo(false)}
                className="px-5 py-2 border border-sol-base1 text-sol-base01 rounded-lg hover:bg-sol-base2 transition-colors text-sm font-serif"
              >
                No
              </button>
              <button
                data-testid="decision-yes-btn"
                onClick={() => answerYesNo(true)}
                className="px-5 py-2 bg-sol-blue text-white rounded-lg hover:bg-sol-blue/90 transition-colors text-sm font-bold"
              >
                Yes
              </button>
            </>
          ) : (
            <button
              data-testid="decision-confirm-btn"
              onClick={confirmDecision}
              className="px-5 py-2 bg-sol-blue text-white rounded-lg hover:bg-sol-blue/90 transition-colors text-sm font-bold"
            >
              Confirm
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
