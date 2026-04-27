import { useEffect, useRef } from 'react'
import { Application } from 'pixi.js'
import { useGameStore } from '../store'
import { CardAreaScene } from '../pixi/CardAreaScene'
import { loadTextures } from '../pixi/textures'
import { client } from '../client'
import { Phase } from '@@gen/dominion/v1/common_pb'
import { CARD_INFO } from '../cards'

export default function CardAreaCanvas() {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const sceneRef = useRef<CardAreaScene | null>(null)
  const appRef = useRef<Application | null>(null)

  const snapshot = useGameStore((s) => s.snapshot)
  const myIdx = useGameStore((s) => s.myIdx)
  const selectedCards = useGameStore((s) => s.selectedCards)
  const gameId = useGameStore((s) => s.gameId)
  const toggleCardSelection = useGameStore((s) => s.toggleCardSelection)
  const decisionModalOpen = useGameStore((s) => s.decisionModalOpen)

  useEffect(() => {
    let app: Application
    async function init() {
      await loadTextures()
      app = new Application()
      await app.init({
        canvas: canvasRef.current!,
        resizeTo: canvasRef.current!.parentElement ?? undefined,
        backgroundAlpha: 0,
        antialias: true,
      })
      appRef.current = app
      const scene = new CardAreaScene(app)
      sceneRef.current = scene
      scene.onCardClick((cardId) => {
        const snap = snapshot
        const idx = myIdx
        const gid = gameId
        if (!snap || idx === null || !gid) return
        const isMyTurn = snap.currentPlayer === idx
        const isAction = snap.phase === Phase.ACTION
        const cardInfo = CARD_INFO[cardId]
        if (isMyTurn && isAction && cardInfo?.types.includes('action')) {
          client.submitAction({
            gameId: gid,
            action: { kind: { case: 'playCard', value: { playerIdx: idx, cardId } } },
          })
        } else {
          toggleCardSelection(cardId)
        }
      })
    }
    init()
    return () => {
      appRef.current?.destroy(true)
      appRef.current = null
      sceneRef.current = null
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!sceneRef.current || !snapshot || myIdx === null) return
    const me = snapshot.players.find((_, i) => i === myIdx)
    const hand = decisionModalOpen ? [] : (me?.hand ?? [])
    const inPlay = me?.inPlay ?? []
    sceneRef.current.update(hand, inPlay, selectedCards)
  }, [snapshot, myIdx, selectedCards, decisionModalOpen])

  return <canvas ref={canvasRef} style={{ width: '100%', height: '100%', display: 'block' }} data-testid="card-area" />
}
