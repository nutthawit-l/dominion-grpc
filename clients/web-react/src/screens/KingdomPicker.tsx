import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { client } from '../client'
import { gameStore } from '../store'
import { KINGDOM_CARDS } from '../cards'

function shuffle<T>(arr: T[]): T[] {
  const a = [...arr]
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[a[i], a[j]] = [a[j], a[i]]
  }
  return a
}

export default function KingdomPicker() {
  const navigate = useNavigate()
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else if (next.size < 10) {
        next.add(id)
      }
      return next
    })
  }

  function randomize() {
    const ids = shuffle(KINGDOM_CARDS.map((c) => c.id)).slice(0, 10)
    setSelected(new Set(ids))
  }

  async function startGame() {
    if (selected.size !== 10) return
    setLoading(true)
    setError(null)
    try {
      const res = await client.createGame({
        players: ['human', 'bigmoney'],
        seed: BigInt(Date.now()),
        kingdom: Array.from(selected),
      })
      gameStore.getState().setGame(res.gameId, 0)
      navigate(`/game/${res.gameId}`)
    } catch (e) {
      setError(String(e))
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen font-serif bg-sol-base3">
      <header className="bg-sol-base2 border-b border-sol-base1/30 sticky top-0 z-10">
        <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-full bg-sol-yellow/15 border border-sol-yellow/40 flex items-center justify-center">
              <span className="text-sol-yellow text-base">♛</span>
            </div>
            <h1 className="text-xl tracking-widest text-sol-base02 uppercase">Dominion</h1>
          </div>
          <span className="text-xs tracking-widest uppercase text-sol-base1">
            {selected.size}/10 selected
          </span>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-8">
        <div className="mb-6 flex flex-col sm:flex-row sm:items-end gap-4 justify-between">
          <div>
            <h2 className="text-2xl text-sol-base02 mb-1">Kingdom Picker</h2>
            <p className="text-sol-base0 text-sm">Select exactly 10 kingdom cards.</p>
          </div>
          <div className="flex gap-3">
            <button
              data-testid="randomize-btn"
              onClick={randomize}
              className="px-4 py-2 border border-sol-base1 text-sol-base01 rounded-lg hover:bg-sol-base2 transition-colors text-sm font-serif"
            >
              Randomize
            </button>
            <button
              data-testid="start-game-btn"
              onClick={startGame}
              disabled={selected.size !== 10 || loading}
              className="px-6 py-2 bg-sol-blue hover:bg-sol-blue/90 disabled:opacity-40 text-white font-bold rounded-lg transition-colors text-sm uppercase tracking-widest"
            >
              {loading ? 'Creating…' : 'Start Game'}
            </button>
          </div>
        </div>

        {error && (
          <p className="mb-4 text-sol-red text-sm" data-testid="error-msg">
            {error}
          </p>
        )}

        <div className="grid grid-cols-5 sm:grid-cols-7 gap-3" data-testid="kingdom-grid">
          {KINGDOM_CARDS.map((card) => {
            const isSel = selected.has(card.id)
            return (
              <button
                key={card.id}
                data-testid={`card-tile-${card.id}`}
                onClick={() => toggle(card.id)}
                title={`${card.name} — $${card.cost} — ${card.description}`}
                className={[
                  'relative rounded-lg overflow-hidden border-2 transition-all duration-150 cursor-pointer',
                  isSel
                    ? 'border-sol-blue shadow-lg scale-105'
                    : 'border-sol-base1/30 hover:scale-105 hover:shadow-md',
                ].join(' ')}
              >
                <img
                  src={`/cards/${card.id}.jpg`}
                  alt={card.name}
                  className={['w-full h-auto block', isSel ? 'brightness-75' : ''].join(' ')}
                />
                {isSel && (
                  <div className="absolute inset-0 flex items-center justify-center">
                    <span className="text-white text-2xl font-bold drop-shadow">✓</span>
                  </div>
                )}
                <div className="absolute bottom-0 left-0 right-0 bg-sol-base02/70 text-white text-[9px] font-mono text-center py-0.5 truncate px-1">
                  {card.name}
                </div>
              </button>
            )
          })}
        </div>
      </main>
    </div>
  )
}
