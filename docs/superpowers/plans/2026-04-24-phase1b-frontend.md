# Phase 1b Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a React/Vite/TypeScript frontend at `clients/web-react/` that lets a human play Dominion against the BigMoney bot in a browser.

**Architecture:** One flat Zustand store holds all state; a single `useEffect` in GameTable opens the Connect-ES server-streaming RPC and pipes every `snapshot`/`ended` event into the store via `applyEvent`. PixiJS renders the card hand+play area as a canvas island inside the React DOM, with a second PixiJS canvas inside a React `<dialog>` for decision prompts.

**Tech Stack:** React 18, Vite 5, TypeScript 5, Zustand 5, PixiJS 8, Playwright, connect-es v2 (`@connectrpc/connect`, `@connectrpc/connect-web`, `@bufbuild/protobuf`), React Router 6, Tailwind CSS 3, Vitest.

**Branch:** `feat/phase1b-frontend`

**Working directory for all commands:** `dominion-grpc/` unless noted.

**Important — server event model:** The service's `fanOut` sends only two event kinds: `snapshot` (one scrubbed snapshot per action, per subscriber) and `ended` (on game over). The other proto event variants (`action_applied`, `phase_changed`, `turn_started`, `decision`) are proto scaffolding for future use and are NOT currently emitted. The frontend only handles `snapshot` and `ended`.

**Design doc:** `docs/superpowers/specs/2026-04-24-phase1b-frontend-design.md`

---

## File Structure

| Stage | File | Purpose |
|---|---|---|
| A | `buf.gen.yaml` | Add TS plugins |
| A | `gen/ts/` | Generated TS stubs (committed, never hand-edited) |
| B | `clients/web-react/package.json` | Dependencies |
| B | `clients/web-react/vite.config.ts` | Vite config + dev proxy |
| B | `clients/web-react/tsconfig.json` | TS config |
| B | `clients/web-react/index.html` | HTML entry |
| B | `clients/web-react/tailwind.config.ts` | Tailwind config |
| B | `clients/web-react/postcss.config.js` | PostCSS config |
| B | `clients/web-react/src/index.css` | Tailwind base styles |
| B | `clients/web-react/src/main.tsx` | React entry |
| C | `clients/web-react/src/client.ts` | Connect-ES client singleton |
| D | `clients/web-react/src/store.ts` | Zustand store |
| D | `clients/web-react/src/__tests__/store.test.ts` | Vitest unit tests for applyEvent |
| E | `clients/web-react/src/cards.ts` | Static card metadata lookup |
| F | `clients/web-react/src/App.tsx` | React Router setup |
| F | `clients/web-react/src/screens/Lobby.tsx` | Lobby screen |
| G | `clients/web-react/src/screens/KingdomPicker.tsx` | Kingdom picker screen |
| H | `clients/web-react/src/screens/GameTable.tsx` | Game table screen + stream hook |
| H | `clients/web-react/src/components/SupplyGrid.tsx` | Supply pile grid |
| H | `clients/web-react/src/components/PlayerInfo.tsx` | Opponent info bar |
| H | `clients/web-react/src/components/GameLog.tsx` | Scrollable event log |
| H | `clients/web-react/src/components/ActionBar.tsx` | Phase/coins/actions + End Phase button |
| I | `clients/web-react/src/pixi/textures.ts` | Shared PIXI.Assets texture cache |
| I | `clients/web-react/src/pixi/CardAreaScene.ts` | PixiJS scene: hand fan + in-play |
| I | `clients/web-react/src/components/CardAreaCanvas.tsx` | React wrapper for CardAreaScene |
| J | `clients/web-react/src/pixi/DecisionScene.ts` | PixiJS scene: decision eligible cards |
| J | `clients/web-react/src/components/DecisionModal.tsx` | React dialog + DecisionScene |
| K | `clients/web-react/playwright.config.ts` | Playwright config |
| K | `clients/web-react/e2e/lobby.spec.ts` | Playwright: lobby |
| K | `clients/web-react/e2e/kingdom-picker.spec.ts` | Playwright: kingdom picker |
| K | `clients/web-react/e2e/game-basic.spec.ts` | Playwright: full game |
| K | `clients/web-react/e2e/decision.spec.ts` | Playwright: decision modal |

Card images: copy `docs/mockups/cards/*.jpg` → `clients/web-react/public/cards/` in Task 2.

---

## Stage A: Proto TS generation

### Task 1: Create branch and extend buf.gen.yaml

**Files:**
- Create branch: `feat/phase1b-frontend`
- Modify: `buf.gen.yaml`

- [ ] **Step 1: Create the branch**

```bash
git checkout -b feat/phase1b-frontend
```

Expected: branch created, clean working tree.

- [ ] **Step 2: Add TS plugins to buf.gen.yaml**

Replace the full contents of `buf.gen.yaml` with:

```yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/nutthawit-l/dominion-grpc/gen/go
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/connectrpc/go
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/bufbuild/es
    out: gen/ts
    opt: target=ts
  - remote: buf.build/connectrpc/es
    out: gen/ts
    opt: target=ts
```

- [ ] **Step 3: Run generate**

```bash
make generate
```

Expected: new files appear under `gen/ts/dominion/v1/`:
- `common_pb.ts`
- `card_pb.ts`
- `game_pb.ts`
- `game_connect.ts`

Verify with: `ls gen/ts/dominion/v1/`

- [ ] **Step 4: Commit**

```bash
git add buf.gen.yaml gen/ts/
git commit -m "feat(proto): add buf TS generation for connect-es client"
```

---

## Stage B: Vite app scaffold

### Task 2: Bootstrap Vite app

**Files:**
- Create: `clients/web-react/package.json`
- Create: `clients/web-react/vite.config.ts`
- Create: `clients/web-react/tsconfig.json`
- Create: `clients/web-react/index.html`
- Create: `clients/web-react/src/main.tsx`
- Create: `clients/web-react/tailwind.config.ts`
- Create: `clients/web-react/postcss.config.js`
- Create: `clients/web-react/src/index.css`

- [ ] **Step 1: Create clients/web-react directory and copy card images**

```bash
mkdir -p clients/web-react/public/cards clients/web-react/src/screens clients/web-react/src/components clients/web-react/src/pixi clients/web-react/src/__tests__ clients/web-react/e2e
cp docs/mockups/cards/*.jpg clients/web-react/public/cards/
```

- [ ] **Step 2: Create package.json**

Create `clients/web-react/package.json`:

```json
{
  "name": "dominion-web",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "test:watch": "vitest",
    "playwright": "playwright test"
  },
  "dependencies": {
    "@bufbuild/protobuf": "^2.3.0",
    "@connectrpc/connect": "^2.0.1",
    "@connectrpc/connect-web": "^2.0.1",
    "pixi.js": "^8.5.0",
    "react": "^18.3.0",
    "react-dom": "^18.3.0",
    "react-router-dom": "^6.27.0",
    "zustand": "^5.0.3"
  },
  "devDependencies": {
    "@playwright/test": "^1.48.0",
    "@types/react": "^18.3.0",
    "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react": "^4.3.0",
    "autoprefixer": "^10.4.0",
    "jsdom": "^25.0.0",
    "postcss": "^8.4.0",
    "tailwindcss": "^3.4.0",
    "typescript": "^5.5.0",
    "vite": "^5.4.0",
    "vitest": "^2.1.0"
  }
}
```

- [ ] **Step 3: Create vite.config.ts**

Create `clients/web-react/vite.config.ts`:

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/dominion.v1.GameService': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
  },
})
```

- [ ] **Step 4: Create tsconfig.json**

Create `clients/web-react/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "paths": {
      "@@gen/*": ["../../gen/ts/*"]
    }
  },
  "include": ["src", "e2e"]
}
```

- [ ] **Step 5: Create index.html**

Create `clients/web-react/index.html`:

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/favicon.ico" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Dominion</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 6: Create Tailwind config**

Create `clients/web-react/tailwind.config.ts`:

```typescript
import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        sol: {
          base03: '#002b36', base02: '#073642', base01: '#586e75',
          base00: '#657b83', base0:  '#839496', base1:  '#93a1a1',
          base2:  '#eee8d5', base3:  '#fdf6e3',
          yellow: '#b58900', orange: '#cb4b16', red:    '#dc322f',
          magenta:'#d33682', violet: '#6c71c4', blue:   '#268bd2',
          cyan:   '#2aa198', green:  '#859900',
        },
      },
      fontFamily: {
        serif: ['Georgia', 'Cambria', 'Times New Roman', 'serif'],
        mono:  ['SFMono-Regular', 'Menlo', 'Consolas', 'monospace'],
      },
    },
  },
  plugins: [],
} satisfies Config
```

- [ ] **Step 7: Create postcss.config.js**

Create `clients/web-react/postcss.config.js`:

```js
export default {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
}
```

- [ ] **Step 8: Create src/index.css**

Create `clients/web-react/src/index.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

* { box-sizing: border-box; }
body { background-color: #fdf6e3; color: #657b83; margin: 0; }
```

- [ ] **Step 9: Create src/main.tsx**

Create `clients/web-react/src/main.tsx`:

```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
)
```

- [ ] **Step 10: Create placeholder App.tsx**

Create `clients/web-react/src/App.tsx`:

```tsx
export default function App() {
  return <div className="p-8 font-serif text-sol-base02">Loading…</div>
}
```

- [ ] **Step 11: Install dependencies**

```bash
cd clients/web-react && npm install
```

Expected: `node_modules/` created, no errors.

- [ ] **Step 12: Verify dev server starts**

```bash
cd clients/web-react && npm run dev
```

Expected output contains: `Local: http://localhost:5173/`

Stop with Ctrl+C.

- [ ] **Step 13: Commit**

```bash
git add clients/web-react/
git commit -m "feat(web): scaffold Vite/React/TS app with Tailwind"
```

---

## Stage C: Connect-ES client

### Task 3: Connect client singleton

**Files:**
- Create: `clients/web-react/src/client.ts`

The Vite proxy forwards all `/dominion.v1.GameService/*` requests to the Go server at `:8080`, so `baseUrl: '/'` avoids cross-origin issues in development.

- [ ] **Step 1: Create client.ts**

Create `clients/web-react/src/client.ts`:

```typescript
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { GameService } from '@@gen/dominion/v1/game_connect'

const transport = createConnectTransport({ baseUrl: '/' })

export const client = createClient(GameService, transport)
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
cd clients/web-react && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add clients/web-react/src/client.ts
git commit -m "feat(web): add Connect-ES client singleton"
```

---

## Stage D: Zustand store

### Task 4: Zustand store + unit tests

**Files:**
- Create: `clients/web-react/src/store.ts`
- Create: `clients/web-react/src/__tests__/store.test.ts`

The store handles only `snapshot` and `ended` events (what the server actually emits). Decision modal state is derived from `snapshot.pendingDecision`.

- [ ] **Step 1: Write failing unit tests**

Create `clients/web-react/src/__tests__/store.test.ts`:

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { gameStore } from '../store'
import type { StreamGameEventsResponse, GameStateSnapshot } from '@@gen/dominion/v1/game_pb'
import { Phase } from '@@gen/dominion/v1/common_pb'

function makeSnapshot(overrides: Partial<GameStateSnapshot> = {}): GameStateSnapshot {
  return {
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
  return { sequence: seq, kind: { case: 'snapshot', value: snap } } as StreamGameEventsResponse
}

function endedEvt(seq = 5n): StreamGameEventsResponse {
  return { sequence: seq, kind: { case: 'ended', value: { winners: [1], finalScores: {} } } } as StreamGameEventsResponse
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
        id: 'd1', playerIdx: 0, cardId: 'militia', step: 0,
        prompt: { case: 'discardFromHand', value: { min: 0, max: 3 } },
      } as any,
    })
    gameStore.getState().applyEvent(snapshotEvt(snap, 1n))
    expect(gameStore.getState().decisionModalOpen).toBe(true)
  })

  it('does not open modal when pendingDecision.playerIdx !== myIdx', () => {
    gameStore.getState().setGame('g1', 0)
    const snap = makeSnapshot({
      pendingDecision: {
        id: 'd1', playerIdx: 1, cardId: 'militia', step: 0,
        prompt: { case: 'discardFromHand', value: { min: 0, max: 3 } },
      } as any,
    })
    gameStore.getState().applyEvent(snapshotEvt(snap, 1n))
    expect(gameStore.getState().decisionModalOpen).toBe(false)
  })

  it('closes modal when pendingDecision is cleared', () => {
    gameStore.getState().setGame('g1', 0)
    const snapWith = makeSnapshot({
      pendingDecision: {
        id: 'd1', playerIdx: 0, cardId: 'chapel', step: 0,
        prompt: { case: 'trashFromHand', value: { min: 0, max: 4 } },
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
```

- [ ] **Step 2: Run tests — expect failure**

```bash
cd clients/web-react && npm test
```

Expected: errors like "Cannot find module '../store'".

- [ ] **Step 3: Create store.ts**

Create `clients/web-react/src/store.ts`:

```typescript
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
        set({
          snapshot: snap,
          streamSeq: seq,
          log: [...log, ...entries],
          decisionModalOpen: isMyDecision,
          selectedCards: isMyDecision && !get().decisionModalOpen ? [] : get().selectedCards,
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
```

- [ ] **Step 4: Run tests — expect pass**

```bash
cd clients/web-react && npm test
```

Expected: all tests pass, no TypeScript errors.

- [ ] **Step 5: Commit**

```bash
git add clients/web-react/src/store.ts clients/web-react/src/__tests__/store.test.ts
git commit -m "feat(web): add Zustand store with applyEvent and unit tests"
```

---

## Stage E: Card metadata

### Task 5: Static card data

**Files:**
- Create: `clients/web-react/src/cards.ts`

- [ ] **Step 1: Create cards.ts**

Create `clients/web-react/src/cards.ts`:

```typescript
export type CardInfo = {
  id: string
  name: string
  cost: number
  types: ('treasure' | 'victory' | 'curse' | 'action' | 'attack' | 'reaction')[]
  description: string
  isKingdom: boolean
}

export const CARD_INFO: Record<string, CardInfo> = {
  copper:       { id: 'copper',       name: 'Copper',       cost: 0, types: ['treasure'],             description: '+$1',                                                    isKingdom: false },
  silver:       { id: 'silver',       name: 'Silver',       cost: 3, types: ['treasure'],             description: '+$2',                                                    isKingdom: false },
  gold:         { id: 'gold',         name: 'Gold',         cost: 6, types: ['treasure'],             description: '+$3',                                                    isKingdom: false },
  estate:       { id: 'estate',       name: 'Estate',       cost: 2, types: ['victory'],              description: '1 VP',                                                   isKingdom: false },
  duchy:        { id: 'duchy',        name: 'Duchy',        cost: 5, types: ['victory'],              description: '3 VP',                                                   isKingdom: false },
  province:     { id: 'province',     name: 'Province',     cost: 8, types: ['victory'],              description: '6 VP',                                                   isKingdom: false },
  curse:        { id: 'curse',        name: 'Curse',        cost: 0, types: ['curse'],                description: '-1 VP',                                                  isKingdom: false },
  artisan:      { id: 'artisan',      name: 'Artisan',      cost: 6, types: ['action'],               description: '+1 Card to hand. Gain a card to your hand costing up to $5. Put a card from your hand onto your deck.', isKingdom: true },
  bandit:       { id: 'bandit',       name: 'Bandit',       cost: 5, types: ['action', 'attack'],     description: 'Gain a Gold. Each other player reveals the top 2 cards of their deck, trashes a revealed Treasure other than Copper, and discards the rest.', isKingdom: true },
  bureaucrat:   { id: 'bureaucrat',   name: 'Bureaucrat',   cost: 4, types: ['action', 'attack'],     description: 'Gain a Silver onto your deck. Each other player with a Victory card reveals one from their hand and puts it onto their deck.', isKingdom: true },
  cellar:       { id: 'cellar',       name: 'Cellar',       cost: 2, types: ['action'],               description: '+1 Action. Discard any number of cards, then draw that many.', isKingdom: true },
  chapel:       { id: 'chapel',       name: 'Chapel',       cost: 2, types: ['action'],               description: 'Trash up to 4 cards from your hand.',                    isKingdom: true },
  council_room: { id: 'council_room', name: 'Council Room', cost: 5, types: ['action'],               description: '+4 Cards. +1 Buy. Each other player draws a card.',      isKingdom: true },
  festival:     { id: 'festival',     name: 'Festival',     cost: 5, types: ['action'],               description: '+2 Actions. +1 Buy. +$2',                                isKingdom: true },
  harbinger:    { id: 'harbinger',    name: 'Harbinger',    cost: 3, types: ['action'],               description: '+1 Card. +1 Action. You may put a card from your discard onto your deck.', isKingdom: true },
  laboratory:   { id: 'laboratory',   name: 'Laboratory',   cost: 5, types: ['action'],               description: '+2 Cards. +1 Action.',                                   isKingdom: true },
  market:       { id: 'market',       name: 'Market',       cost: 5, types: ['action'],               description: '+1 Card. +1 Action. +1 Buy. +$1',                        isKingdom: true },
  militia:      { id: 'militia',      name: 'Militia',      cost: 4, types: ['action', 'attack'],     description: '+$2. Each other player discards down to 3 cards in hand.', isKingdom: true },
  mine:         { id: 'mine',         name: 'Mine',         cost: 5, types: ['action'],               description: 'Trash a Treasure from your hand. Gain a Treasure to your hand costing up to $3 more than it.', isKingdom: true },
  moat:         { id: 'moat',         name: 'Moat',         cost: 2, types: ['action', 'reaction'],   description: '+2 Cards. When another player plays an Attack, you may reveal this to be unaffected.', isKingdom: true },
  moneylender:  { id: 'moneylender',  name: 'Moneylender',  cost: 4, types: ['action'],               description: 'You may trash a Copper from your hand for +$3.',          isKingdom: true },
  poacher:      { id: 'poacher',      name: 'Poacher',      cost: 4, types: ['action'],               description: '+1 Card. +1 Action. +$1. Discard a card per empty Supply pile.', isKingdom: true },
  remodel:      { id: 'remodel',      name: 'Remodel',      cost: 4, types: ['action'],               description: 'Trash a card from your hand. Gain a card costing up to $2 more than it.', isKingdom: true },
  smithy:       { id: 'smithy',       name: 'Smithy',       cost: 4, types: ['action'],               description: '+3 Cards.',                                              isKingdom: true },
  vassal:       { id: 'vassal',       name: 'Vassal',       cost: 3, types: ['action'],               description: '+$2. Discard the top card of your deck. If it\'s an Action card, you may play it.', isKingdom: true },
  village:      { id: 'village',      name: 'Village',      cost: 3, types: ['action'],               description: '+1 Card. +2 Actions.',                                   isKingdom: true },
  witch:        { id: 'witch',        name: 'Witch',        cost: 5, types: ['action', 'attack'],     description: '+2 Cards. Each other player gains a Curse.',             isKingdom: true },
  workshop:     { id: 'workshop',     name: 'Workshop',     cost: 3, types: ['action'],               description: 'Gain a card costing up to $4.',                          isKingdom: true },
}

export const KINGDOM_CARDS: CardInfo[] = Object.values(CARD_INFO).filter((c) => c.isKingdom)

export function getCard(id: string): CardInfo {
  return CARD_INFO[id] ?? { id, name: id, cost: 0, types: [], description: '', isKingdom: false }
}
```

- [ ] **Step 2: Verify compile**

```bash
cd clients/web-react && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add clients/web-react/src/cards.ts
git commit -m "feat(web): add static card metadata lookup"
```

---

## Stage F: Routing + Lobby

### Task 6: App routing and Lobby screen

**Files:**
- Modify: `clients/web-react/src/App.tsx`
- Create: `clients/web-react/src/screens/Lobby.tsx`

- [ ] **Step 1: Update App.tsx with routes**

Replace `clients/web-react/src/App.tsx`:

```tsx
import { Routes, Route, Navigate } from 'react-router-dom'
import Lobby from './screens/Lobby'
import KingdomPicker from './screens/KingdomPicker'
import GameTable from './screens/GameTable'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Lobby />} />
      <Route path="/new" element={<KingdomPicker />} />
      <Route path="/game/:gameId" element={<GameTable />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
```

- [ ] **Step 2: Create placeholder screens so App compiles**

Create `clients/web-react/src/screens/KingdomPicker.tsx`:

```tsx
export default function KingdomPicker() {
  return <div>Kingdom Picker (coming soon)</div>
}
```

Create `clients/web-react/src/screens/GameTable.tsx`:

```tsx
export default function GameTable() {
  return <div>Game Table (coming soon)</div>
}
```

- [ ] **Step 3: Create Lobby.tsx**

Create `clients/web-react/src/screens/Lobby.tsx`:

```tsx
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
```

- [ ] **Step 4: Verify dev server shows Lobby**

```bash
cd clients/web-react && npm run dev
```

Open `http://localhost:5173/`. Expected: Lobby page with "Create Game" button visible.

Stop with Ctrl+C.

- [ ] **Step 5: Commit**

```bash
git add clients/web-react/src/App.tsx clients/web-react/src/screens/
git commit -m "feat(web): add routing and Lobby screen"
```

---

## Stage G: Kingdom Picker

### Task 7: Kingdom Picker screen

**Files:**
- Modify: `clients/web-react/src/screens/KingdomPicker.tsx`

- [ ] **Step 1: Implement KingdomPicker.tsx**

Replace `clients/web-react/src/screens/KingdomPicker.tsx`:

```tsx
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { client } from '../client'
import { gameStore } from '../store'
import { KINGDOM_CARDS, getCard } from '../cards'

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
```

- [ ] **Step 2: Verify visually**

```bash
cd clients/web-react && npm run dev
```

Open `http://localhost:5173/new`. Expected:
- Grid of 21 card tiles
- Click tiles to select (blue border + checkmark), up to 10
- "Randomize" picks 10 at random
- "Start Game" disabled until exactly 10 selected

Stop with Ctrl+C.

- [ ] **Step 3: Commit**

```bash
git add clients/web-react/src/screens/KingdomPicker.tsx
git commit -m "feat(web): add Kingdom Picker screen with random and manual selection"
```

---

## Stage H: Game Table

### Task 8: Game Table shell + stream hook

**Files:**
- Modify: `clients/web-react/src/screens/GameTable.tsx`

- [ ] **Step 1: Implement GameTable.tsx**

Replace `clients/web-react/src/screens/GameTable.tsx`:

```tsx
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
      try {
        const stream = client.streamGameEvents(
          { gameId: gameId!, playerIdx: myIdx! },
          { signal: ac.signal },
        )
        for await (const evt of stream) {
          const seq = evt.sequence
          if (seq > streamSeqRef.current + 1n) {
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
      gameStore.getState().clearGame()
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
```

- [ ] **Step 2: Create placeholder components so GameTable compiles**

Create `clients/web-react/src/components/SupplyGrid.tsx`:

```tsx
import type { SupplyPile, GameStateSnapshot } from '@@gen/dominion/v1/game_pb'
export default function SupplyGrid(_: { supply: SupplyPile[]; myIdx: number; snapshot: GameStateSnapshot }) {
  return <div data-testid="supply-grid">Supply (coming soon)</div>
}
```

Create `clients/web-react/src/components/PlayerInfo.tsx`:

```tsx
import type { PlayerView } from '@@gen/dominion/v1/game_pb'
export default function PlayerInfo(_: { player: PlayerView; isCurrentPlayer: boolean }) {
  return <div data-testid="player-info">Player info (coming soon)</div>
}
```

Create `clients/web-react/src/components/GameLog.tsx`:

```tsx
import type { LogEntry } from '../store'
export default function GameLog(_: { entries: LogEntry[] }) {
  return <div data-testid="game-log">Log (coming soon)</div>
}
```

Create `clients/web-react/src/components/ActionBar.tsx`:

```tsx
import type { PlayerView } from '@@gen/dominion/v1/game_pb'
import { Phase } from '@@gen/dominion/v1/common_pb'
export default function ActionBar(_: { player: PlayerView; phase: Phase; currentPlayer: number; myIdx: number; gameId: string; ended: boolean }) {
  return <div data-testid="action-bar">Action bar (coming soon)</div>
}
```

Create `clients/web-react/src/components/CardAreaCanvas.tsx`:

```tsx
export default function CardAreaCanvas() {
  return <canvas data-testid="card-area" style={{ width: '100%', height: '100%' }} />
}
```

Create `clients/web-react/src/components/DecisionModal.tsx`:

```tsx
export default function DecisionModal(_: { gameId: string; myIdx: number }) {
  return null
}
```

- [ ] **Step 3: Verify compile**

```bash
cd clients/web-react && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add clients/web-react/src/screens/GameTable.tsx clients/web-react/src/components/
git commit -m "feat(web): add GameTable shell with stream hook and placeholder components"
```

### Task 9: Supply, PlayerInfo, GameLog, ActionBar

**Files:**
- Modify: `clients/web-react/src/components/SupplyGrid.tsx`
- Modify: `clients/web-react/src/components/PlayerInfo.tsx`
- Modify: `clients/web-react/src/components/GameLog.tsx`
- Modify: `clients/web-react/src/components/ActionBar.tsx`

- [ ] **Step 1: Implement SupplyGrid.tsx**

Replace `clients/web-react/src/components/SupplyGrid.tsx`:

```tsx
import { client } from '../client'
import { gameStore } from '../store'
import { getCard } from '../cards'
import { Phase } from '@@gen/dominion/v1/common_pb'
import type { SupplyPile, GameStateSnapshot } from '@@gen/dominion/v1/game_pb'

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
```

- [ ] **Step 2: Implement PlayerInfo.tsx**

Replace `clients/web-react/src/components/PlayerInfo.tsx`:

```tsx
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
```

- [ ] **Step 3: Implement GameLog.tsx**

Replace `clients/web-react/src/components/GameLog.tsx`:

```tsx
import { useEffect, useRef } from 'react'
import type { LogEntry } from '../store'

interface Props {
  entries: LogEntry[]
}

export default function GameLog({ entries }: Props) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [entries.length])

  return (
    <div data-testid="game-log" className="flex flex-col gap-0.5 text-xs font-mono text-sol-base00">
      <p className="text-sol-base1 uppercase tracking-widest text-[10px] mb-1">Log</p>
      {entries.map((e) => (
        <div key={String(e.seq)} className="leading-relaxed">
          {e.text}
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  )
}
```

- [ ] **Step 4: Implement ActionBar.tsx**

Replace `clients/web-react/src/components/ActionBar.tsx`:

```tsx
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
```

- [ ] **Step 5: Verify compile**

```bash
cd clients/web-react && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add clients/web-react/src/components/
git commit -m "feat(web): implement SupplyGrid, PlayerInfo, GameLog, ActionBar"
```

---

## Stage I: PixiJS card area

### Task 10: Texture cache

**Files:**
- Create: `clients/web-react/src/pixi/textures.ts`

- [ ] **Step 1: Create textures.ts**

Create `clients/web-react/src/pixi/textures.ts`:

```typescript
import { Assets, Texture } from 'pixi.js'

const ALL_CARD_IDS = [
  'copper', 'silver', 'gold',
  'estate', 'duchy', 'province', 'curse',
  'artisan', 'bandit', 'bureaucrat', 'cellar', 'chapel',
  'council_room', 'festival', 'harbinger', 'laboratory', 'market',
  'militia', 'mine', 'moat', 'moneylender', 'poacher',
  'remodel', 'smithy', 'vassal', 'village', 'witch', 'workshop',
]

let loaded = false

export async function loadTextures(): Promise<void> {
  if (loaded) return
  await Assets.load(
    ALL_CARD_IDS.map((id) => ({ alias: id, src: `/cards/${id}.jpg` })),
  )
  loaded = true
}

export function getTexture(cardId: string): Texture {
  return Assets.get(cardId) ?? Texture.EMPTY
}
```

- [ ] **Step 2: Commit**

```bash
git add clients/web-react/src/pixi/textures.ts
git commit -m "feat(web): add PixiJS texture cache loader"
```

### Task 11: CardAreaScene + CardAreaCanvas

**Files:**
- Create: `clients/web-react/src/pixi/CardAreaScene.ts`
- Modify: `clients/web-react/src/components/CardAreaCanvas.tsx`

The scene renders two rows: the human's hand fan (bottom) and in-play cards (top). Clicking a card in hand during action phase plays it; otherwise no-op. The GameTable decides what click means based on phase.

- [ ] **Step 1: Create CardAreaScene.ts**

Create `clients/web-react/src/pixi/CardAreaScene.ts`:

```typescript
import { Application, Container, Sprite, Graphics } from 'pixi.js'
import { getTexture } from './textures'

const CARD_W = 72
const CARD_H = 108
const HAND_OVERLAP = 18
const IN_PLAY_OVERLAP = 10

export class CardAreaScene {
  private app: Application
  private handContainer: Container
  private inPlayContainer: Container
  private onClickCb: ((cardId: string) => void) | null = null

  constructor(app: Application) {
    this.app = app
    this.handContainer = new Container()
    this.inPlayContainer = new Container()
    app.stage.addChild(this.inPlayContainer)
    app.stage.addChild(this.handContainer)
  }

  onCardClick(cb: (cardId: string) => void) {
    this.onClickCb = cb
  }

  update(hand: string[], inPlay: string[], selected: string[]) {
    this.renderHand(hand, selected)
    this.renderInPlay(inPlay)
  }

  private renderHand(hand: string[], selected: string[]) {
    this.handContainer.removeChildren()
    const w = this.app.renderer.width
    const h = this.app.renderer.height
    const totalW = hand.length * (CARD_W - HAND_OVERLAP) + HAND_OVERLAP
    const startX = (w - totalW) / 2
    const baseY = h - CARD_H - 8

    hand.forEach((cardId, i) => {
      const sprite = new Sprite(getTexture(cardId))
      sprite.width = CARD_W
      sprite.height = CARD_H
      sprite.x = startX + i * (CARD_W - HAND_OVERLAP)
      const isSel = selected.includes(cardId)
      sprite.y = isSel ? baseY - 16 : baseY
      sprite.eventMode = 'static'
      sprite.cursor = 'pointer'
      sprite.on('pointerdown', () => this.onClickCb?.(cardId))

      // selected border
      if (isSel) {
        const border = new Graphics()
        border.rect(0, 0, CARD_W, CARD_H).stroke({ color: 0x268bd2, width: 3 })
        sprite.addChild(border)
      }

      this.handContainer.addChild(sprite)
    })
  }

  private renderInPlay(inPlay: string[]) {
    this.inPlayContainer.removeChildren()
    const w = this.app.renderer.width
    const totalW = inPlay.length * (CARD_W - IN_PLAY_OVERLAP) + IN_PLAY_OVERLAP
    const startX = (w - totalW) / 2

    inPlay.forEach((cardId, i) => {
      const sprite = new Sprite(getTexture(cardId))
      sprite.width = CARD_W
      sprite.height = CARD_H
      sprite.x = startX + i * (CARD_W - IN_PLAY_OVERLAP)
      sprite.y = 8
      this.inPlayContainer.addChild(sprite)
    })
  }
}
```

- [ ] **Step 2: Implement CardAreaCanvas.tsx**

Replace `clients/web-react/src/components/CardAreaCanvas.tsx`:

```tsx
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
        if (!snapshot || myIdx === null || !gameId) return
        const isMyTurn = snapshot.currentPlayer === myIdx
        const isAction = snapshot.phase === Phase.ACTION
        const cardInfo = CARD_INFO[cardId]
        if (isMyTurn && isAction && cardInfo?.types.includes('action')) {
          client.submitAction({
            gameId,
            action: { kind: { case: 'playCard', value: { playerIdx: myIdx, cardId } } },
          })
        } else {
          toggleCardSelection(cardId)
        }
      })
    }
    init()
    return () => {
      appRef.current?.destroy(true)
    }
  }, [])

  useEffect(() => {
    if (!sceneRef.current || !snapshot || myIdx === null) return
    const me = snapshot.players.find((_, i) => i === myIdx)
    const hand = decisionModalOpen ? [] : (me?.hand ?? [])
    const inPlay = me?.inPlay ?? []
    sceneRef.current.update(hand, inPlay, selectedCards)
  }, [snapshot, myIdx, selectedCards, decisionModalOpen])

  return <canvas ref={canvasRef} style={{ width: '100%', height: '100%', display: 'block' }} data-testid="card-area" />
}
```

- [ ] **Step 3: Verify compile**

```bash
cd clients/web-react && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add clients/web-react/src/pixi/ clients/web-react/src/components/CardAreaCanvas.tsx
git commit -m "feat(web): add PixiJS CardAreaScene with hand fan and in-play rendering"
```

---

## Stage J: Decision modal

### Task 12: DecisionScene + DecisionModal

**Files:**
- Create: `clients/web-react/src/pixi/DecisionScene.ts`
- Modify: `clients/web-react/src/components/DecisionModal.tsx`

- [ ] **Step 1: Create DecisionScene.ts**

Create `clients/web-react/src/pixi/DecisionScene.ts`:

```typescript
import { Application, Container, Sprite, Graphics } from 'pixi.js'
import { getTexture } from './textures'

const CARD_W = 80
const CARD_H = 120
const GAP = 8

export class DecisionScene {
  private app: Application
  private container: Container
  private onClickCb: ((cardId: string) => void) | null = null

  constructor(app: Application) {
    this.app = app
    this.container = new Container()
    app.stage.addChild(this.container)
  }

  onCardClick(cb: (cardId: string) => void) {
    this.onClickCb = cb
  }

  update(eligible: string[], selected: string[]) {
    this.container.removeChildren()
    const totalW = eligible.length * (CARD_W + GAP) - GAP
    const startX = Math.max(0, (this.app.renderer.width - totalW) / 2)

    eligible.forEach((cardId, i) => {
      const sprite = new Sprite(getTexture(cardId))
      sprite.width = CARD_W
      sprite.height = CARD_H
      sprite.x = startX + i * (CARD_W + GAP)
      sprite.y = 8
      sprite.eventMode = 'static'
      sprite.cursor = 'pointer'
      sprite.on('pointerdown', () => this.onClickCb?.(cardId))

      if (selected.includes(cardId)) {
        const border = new Graphics()
        border.rect(0, 0, CARD_W, CARD_H).stroke({ color: 0x268bd2, width: 3 })
        sprite.addChild(border)
        sprite.y = 0
      }

      this.container.addChild(sprite)
    })
  }
}
```

- [ ] **Step 2: Implement DecisionModal.tsx**

Replace `clients/web-react/src/components/DecisionModal.tsx`:

```tsx
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
  }, [decisionModalOpen])

  useEffect(() => {
    if (sceneRef.current) {
      sceneRef.current.update(eligible, selectedCards)
    }
  }, [eligible.join(','), selectedCards.join(',')])

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
```

- [ ] **Step 3: Verify compile**

```bash
cd clients/web-react && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add clients/web-react/src/pixi/DecisionScene.ts clients/web-react/src/components/DecisionModal.tsx
git commit -m "feat(web): add DecisionScene and DecisionModal with PixiJS canvas"
```

---

## Stage K: Playwright e2e

### Task 13: Playwright config

**Files:**
- Create: `clients/web-react/playwright.config.ts`

- [ ] **Step 1: Install Playwright browsers**

```bash
cd clients/web-react && npx playwright install chromium
```

Expected: Chromium downloaded.

- [ ] **Step 2: Create playwright.config.ts**

Create `clients/web-react/playwright.config.ts`:

```typescript
import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  use: {
    baseURL: 'http://localhost:5173',
    headless: true,
  },
  webServer: [
    {
      command: 'make -C ../.. server',
      url: 'http://localhost:8080',
      reuseExistingServer: true,
      timeout: 10_000,
    },
    {
      command: 'npm run dev',
      url: 'http://localhost:5173',
      reuseExistingServer: true,
      timeout: 15_000,
    },
  ],
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
})
```

- [ ] **Step 3: Commit**

```bash
git add clients/web-react/playwright.config.ts
git commit -m "feat(web): add Playwright config"
```

### Task 14: Lobby and Kingdom Picker Playwright specs

**Files:**
- Create: `clients/web-react/e2e/lobby.spec.ts`
- Create: `clients/web-react/e2e/kingdom-picker.spec.ts`

- [ ] **Step 1: Create lobby.spec.ts**

Create `clients/web-react/e2e/lobby.spec.ts`:

```typescript
import { test, expect } from '@playwright/test'

test('lobby shows Create Game button', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByTestId('create-game-btn')).toBeVisible()
})

test('Create Game button navigates to /new', async ({ page }) => {
  await page.goto('/')
  await page.getByTestId('create-game-btn').click()
  await expect(page).toHaveURL('/new')
  await expect(page.getByTestId('kingdom-grid')).toBeVisible()
})
```

- [ ] **Step 2: Create kingdom-picker.spec.ts**

Create `clients/web-react/e2e/kingdom-picker.spec.ts`:

```typescript
import { test, expect } from '@playwright/test'

test('Randomize selects exactly 10 cards', async ({ page }) => {
  await page.goto('/new')
  await page.getByTestId('randomize-btn').click()
  const startBtn = page.getByTestId('start-game-btn')
  await expect(startBtn).toBeEnabled()
  // count visually-selected tiles (have brightness-75 class)
  const selected = page.locator('[data-testid^="card-tile-"] img.brightness-75')
  await expect(selected).toHaveCount(10)
})

test('manual picker enforces exactly 10', async ({ page }) => {
  await page.goto('/new')
  const startBtn = page.getByTestId('start-game-btn')
  await expect(startBtn).toBeDisabled()

  // select 10 kingdom cards
  const tiles = page.locator('[data-testid^="card-tile-"]')
  for (let i = 0; i < 10; i++) {
    await tiles.nth(i).click()
  }
  await expect(startBtn).toBeEnabled()

  // deselect one — button disables again
  await tiles.nth(0).click()
  await expect(startBtn).toBeDisabled()
})

test('Start Game creates game and navigates to /game/:id', async ({ page }) => {
  await page.goto('/new')
  await page.getByTestId('randomize-btn').click()
  await page.getByTestId('start-game-btn').click()
  await expect(page).toHaveURL(/\/game\/.+/)
  await expect(page.getByTestId('supply-grid')).toBeVisible()
})
```

- [ ] **Step 3: Run specs**

```bash
cd clients/web-react && npm run playwright -- e2e/lobby.spec.ts e2e/kingdom-picker.spec.ts
```

Expected: all 4 tests pass.

- [ ] **Step 4: Commit**

```bash
git add clients/web-react/e2e/lobby.spec.ts clients/web-react/e2e/kingdom-picker.spec.ts
git commit -m "test(web): add Playwright specs for Lobby and Kingdom Picker"
```

### Task 15: Game flow Playwright specs

**Files:**
- Create: `clients/web-react/e2e/game-basic.spec.ts`
- Create: `clients/web-react/e2e/decision.spec.ts`

- [ ] **Step 1: Create game-basic.spec.ts**

The test plays through a full game: the human just ends phase every turn. BigMoney buys provinces; eventually the province pile empties and the game ends.

Create `clients/web-react/e2e/game-basic.spec.ts`:

```typescript
import { test, expect } from '@playwright/test'

async function createAndJoin(page: import('@playwright/test').Page) {
  await page.goto('/new')
  await page.getByTestId('randomize-btn').click()
  await page.getByTestId('start-game-btn').click()
  await expect(page).toHaveURL(/\/game\/.+/)
}

test('game table shows supply grid after creation', async ({ page }) => {
  await createAndJoin(page)
  await expect(page.getByTestId('supply-grid')).toBeVisible()
  await expect(page.getByTestId('game-log')).toBeVisible()
  await expect(page.getByTestId('card-area')).toBeVisible()
})

test('End Phase button is visible on my turn', async ({ page }) => {
  await createAndJoin(page)
  // player 0 goes first — End Phase should be available
  await expect(page.getByTestId('end-phase-btn')).toBeVisible({ timeout: 5000 })
})

test('human can end phases until game over', async ({ page }) => {
  test.setTimeout(120_000)
  await createAndJoin(page)

  // Keep clicking End Phase whenever it appears, until game ends
  const maxClicks = 200
  let clicks = 0
  while (clicks < maxClicks) {
    const endBtn = page.getByTestId('end-phase-btn')
    const gameOver = page.getByText(/game over/i)

    const which = await Promise.race([
      endBtn.waitFor({ state: 'visible', timeout: 8000 }).then(() => 'btn' as const),
      gameOver.waitFor({ state: 'visible', timeout: 8000 }).then(() => 'over' as const),
    ]).catch(() => 'timeout' as const)

    if (which === 'over') break
    if (which === 'timeout') break
    await endBtn.click()
    clicks++
  }

  await expect(page.getByText(/game over/i)).toBeVisible({ timeout: 5000 })
})
```

- [ ] **Step 2: Create decision.spec.ts**

The decision modal test uses a kingdom containing Chapel (trash up to 4 cards) to reliably trigger a decision on turn 1. We pick a fixed kingdom that includes chapel.

Create `clients/web-react/e2e/decision.spec.ts`:

```typescript
import { test, expect } from '@playwright/test'

test('decision modal opens and can be confirmed', async ({ page }) => {
  test.setTimeout(60_000)

  // Use a kingdom with chapel so a decision is likely to appear
  await page.goto('/new')
  // manually select chapel + 9 others
  await page.getByTestId('card-tile-chapel').click()
  await page.getByTestId('card-tile-smithy').click()
  await page.getByTestId('card-tile-village').click()
  await page.getByTestId('card-tile-market').click()
  await page.getByTestId('card-tile-laboratory').click()
  await page.getByTestId('card-tile-festival').click()
  await page.getByTestId('card-tile-cellar').click()
  await page.getByTestId('card-tile-mine').click()
  await page.getByTestId('card-tile-witch').click()
  await page.getByTestId('card-tile-moat').click()
  await page.getByTestId('start-game-btn').click()
  await expect(page).toHaveURL(/\/game\/.+/)

  // Play until a decision modal appears or 30 end-phase clicks
  let clicks = 0
  while (clicks < 30) {
    const modal = page.getByTestId('decision-modal')
    const endBtn = page.getByTestId('end-phase-btn')

    const which = await Promise.race([
      modal.waitFor({ state: 'visible', timeout: 3000 }).then(() => 'modal' as const),
      endBtn.waitFor({ state: 'visible', timeout: 3000 }).then(() => 'btn' as const),
    ]).catch(() => 'timeout' as const)

    if (which === 'modal') {
      // Modal is open — confirm it
      const confirmBtn = page.getByTestId('decision-confirm-btn')
      const yesBtn = page.getByTestId('decision-yes-btn')
      if (await confirmBtn.isVisible()) {
        await confirmBtn.click()
      } else if (await yesBtn.isVisible()) {
        await yesBtn.click()
      }
      await expect(modal).not.toBeVisible({ timeout: 5000 })
      break
    }
    if (which === 'btn') {
      await endBtn.click()
      clicks++
    } else {
      break
    }
  }
})
```

- [ ] **Step 3: Run all e2e specs**

```bash
cd clients/web-react && npm run playwright
```

Expected: all specs pass. The `game-basic` full-game test may take up to 2 minutes.

- [ ] **Step 4: Commit**

```bash
git add clients/web-react/e2e/game-basic.spec.ts clients/web-react/e2e/decision.spec.ts
git commit -m "test(web): add Playwright e2e specs for game flow and decision modal"
```

---

## Final: Push branch

- [ ] **Step 1: Run full unit test suite**

```bash
cd clients/web-react && npm test
```

Expected: all Vitest tests pass.

- [ ] **Step 2: TypeScript clean build**

```bash
cd clients/web-react && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Run all Playwright specs**

```bash
cd clients/web-react && npm run playwright
```

Expected: all specs pass.

- [ ] **Step 4: Push branch**

```bash
git push -u origin feat/phase1b-frontend
```
