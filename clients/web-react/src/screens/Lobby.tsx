import { useNavigate } from 'react-router-dom'

export default function Lobby() {
  const navigate = useNavigate()

  return (
    <div className="min-h-screen font-serif bg-sol-base3">
      <header className="bg-sol-base2 border-b border-sol-base1/30 sticky top-0 z-10">
        <div className="max-w-5xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-full bg-sol-yellow/15 border border-sol-yellow/40 flex items-center justify-center">
              <span className="text-sol-yellow text-base leading-none">♛</span>
            </div>
            <h1 className="text-xl font-serif tracking-widest text-sol-base02 uppercase">Dominion</h1>
          </div>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-10">
        <div className="mb-8">
          <h2 className="text-2xl text-sol-base02 font-serif mb-1">Lobby</h2>
          <p className="text-sol-base0 text-sm">Start a new game against the BigMoney bot.</p>
        </div>

        <button
          data-testid="create-game-btn"
          onClick={() => navigate('/new')}
          className="inline-flex items-center gap-2 bg-sol-blue hover:bg-sol-blue/90 text-white font-serif font-bold px-6 py-2.5 rounded-lg transition-colors shadow text-sm uppercase tracking-widest"
        >
          <span>+</span> Create Game
        </button>
      </main>
    </div>
  )
}
