# Phase 1b — Frontend Design

**Stack:** React 18 / Vite / TypeScript / Zustand / PixiJS / Playwright

**Done-criterion:** A human can play a full game against the BigMoney bot in a browser — action phase, buy phase, decisions (discard, trash, gain, etc.) — until the game ends.

**Branch:** `feat/phase1b-frontend`

---

## Phase definitions recap

| Phase | Scope |
|---|---|
| 1a | Go engine + Connect server + bot-vs-bot (complete) |
| **1b** | React frontend — human vs bot in browser |
| 2 | Auth, lobby, multiplayer, persistence |

---

## Section 1 — Repo layout & tooling

The frontend lives at `clients/web-react/` inside the existing `dominion-grpc` repo. The buf-generated TypeScript client goes to `gen/ts/`.

```
dominion-grpc/
├── clients/
│   └── web-react/          ← Vite/React app (new)
│       ├── src/
│       ├── index.html
│       ├── vite.config.ts
│       ├── tsconfig.json
│       └── package.json
├── gen/
│   ├── go/                 ← existing, unchanged
│   └── ts/                 ← buf-generated TS client (new)
├── proto/                  ← unchanged
└── buf.gen.yaml            ← extended to also emit TS
```

**Tooling additions:**

- `buf.gen.yaml` gains `@connectrpc/protoc-gen-connect-es` + `@bufbuild/protoc-gen-es` plugin blocks to emit `gen/ts/`
- `make generate` regenerates both Go and TS stubs
- `clients/web-react/` has its own `package.json`; no root-level Node introduced
- Card images sourced from `docs/mockups/cards/` — copied into `clients/web-react/public/cards/`

---

## Section 2 — Zustand store

One flat store. No slices.

```typescript
interface GameStore {
  // Connection
  gameId: string | null
  myIdx: number | null        // which seat the human is in (0 or 1)

  // Server state
  snapshot: GameStateSnapshot | null
  log: LogEntry[]             // accumulated from stream events, pre-rendered to strings
  streamSeq: bigint           // last seen sequence number

  // UI state
  selectedCards: string[]     // card_ids currently selected in hand or decision modal
  decisionModalOpen: boolean

  // Actions
  setGame: (gameId: string, myIdx: number) => void
  applyEvent: (evt: StreamGameEventsResponse) => void
  toggleCardSelection: (cardId: string) => void
  confirmDecision: () => void
  clearGame: () => void
}

interface LogEntry {
  seq: bigint
  at: Date
  text: string   // pre-rendered human-readable description of the event
}
```

`applyEvent` is the single write path from the stream — it updates `snapshot`, appends to `log`, and bumps `streamSeq`. The store function itself is the reducer; no separate reducer layer.

**`myIdx` is used for three things:**

1. **Rendering** — `snapshot.players[myIdx]` is the human (rendered at the bottom with a full hand); all other entries are opponents.
2. **Action submission** — every `Action` proto field `player_idx` is filled from `myIdx`.
3. **Decision gating** — the decision modal only opens when `pending_decision.player_idx === myIdx`.

**4-player readiness:** Player count is always derived from `snapshot.players.length`. Nothing hardcodes 2. Scaling to 4 players is a UI layout change only.

---

## Section 3 — Screens & routing

Three screens via React Router:

```
/               → Lobby
/new            → Kingdom Picker
/game/:gameId   → Game Table
```

**Lobby (`/`)**

- Shows a "Create Game" button.
- Phase 1b does not list existing games (no `ListGames` RPC).
- "Create Game" navigates to `/new`.

**Kingdom Picker (`/new`)**

- Displays all 25 Base kingdom cards as tiles.
- "Randomize" button picks 10 at random client-side and selects them.
- Manual mode: user toggles tiles; confirm button is enabled only when exactly 10 are selected.
- "Start Game" calls `CreateGame` with `players: ["human", "bigmoney"]`, `seed: Date.now()`, `kingdom: [selected 10 card_ids]`.
- On success, navigates to `/game/:gameId` and sets `myIdx = 0` in the store.

**Game Table (`/game/:gameId`)**

- Mounts the stream on load.
- Contains: opponent info bar (top), supply grid (middle), card area PixiJS canvas — hand fan + in-play (center/bottom), game log sidebar (right), action controls (bottom bar).
- Unmounts the stream and calls `clearGame` on navigation away.

---

## Section 4 — PixiJS integration

PixiJS is used in exactly two places. Both follow the same pattern: `useRef` holds the canvas element, `useEffect` creates and destroys the `PIXI.Application`, a store subscription drives imperative sprite updates.

**1. Card play area (Game Table)**

Renders the human's hand fan and the in-play area. Each card is a `PIXI.Sprite` with a texture loaded from `/cards/{card_id}.jpg`. Clicking a card calls `toggleCardSelection`.

```tsx
const canvasRef = useRef<HTMLCanvasElement>(null)

useEffect(() => {
  let app: PIXI.Application

  async function init() {
    app = new PIXI.Application()
    await app.init({ canvas: canvasRef.current! })
    // subscribe to store, update sprites on snapshot change
  }

  init()
  return () => app?.destroy()
}, [])
```

**2. Decision modal canvas**

A React `<dialog>` opens when `decisionModalOpen` is true. Inside it, a second `PIXI.Application` renders only the eligible cards for the current decision prompt (filtered from hand or supply per the prompt type). Selecting a card calls `toggleCardSelection`. "Confirm" calls `confirmDecision`.

```tsx
<dialog open={decisionModalOpen}>
  <p>{promptDescription}</p>
  <canvas ref={decisionCanvasRef} />
  <button onClick={confirmDecision}>Confirm</button>
</dialog>
```

**Texture cache:** All card textures are loaded once at app startup via `PIXI.Assets.load` into a shared cache keyed by `card_id`. Both PixiJS instances share the cache — no duplicate loads.

---

## Section 5 — Stream connection & event flow

The Game Table mounts one `useEffect` that opens `StreamGameEvents` and pipes events into the store:

```typescript
useEffect(() => {
  const ac = new AbortController()

  async function run() {
    for await (const evt of client.streamGameEvents(
      { gameId, playerIdx: myIdx },
      { signal: ac.signal }
    )) {
      store.applyEvent(evt)
    }
  }

  run().catch(err => { if (!ac.signal.aborted) console.error(err) })
  return () => ac.abort()
}, [gameId, myIdx])
```

**`applyEvent` event handling:**

| Event kind | Store update |
|---|---|
| `snapshot` | replace `snapshot` |
| `action_applied` | replace `snapshot` from `state_after`, append log entry |
| `phase_changed` | append log entry |
| `turn_started` | append log entry |
| `decision` | set `snapshot.pending_decision`; open modal if `player_idx === myIdx` |
| `ended` | replace `snapshot`, append "Game over — winner: X" log entry |

**Sequence gap recovery:** if `evt.sequence > streamSeq + 1n`, abort and resubscribe from scratch (same recovery strategy as the Go bot).

**Action submission:** `SubmitAction` is called directly from UI event handlers (play card, buy card, end phase, confirm decision). Fire-and-forget — the result arrives on the stream.

---

## Section 6 — Testing (Playwright)

Playwright e2e tests run against the real Go server.

```typescript
// playwright.config.ts
webServer: {
  command: 'make -C ../.. server',
  url: 'http://localhost:8080',
  reuseExistingServer: true,
}
```

**Test suite:**

| File | Coverage |
|---|---|
| `lobby.spec.ts` | Create Game button navigates to kingdom picker |
| `kingdom-picker.spec.ts` | Randomize selects 10; manual picker enforces exactly 10; Start Game creates game and navigates to `/game/:id` |
| `game-basic.spec.ts` | Human plays a full game vs BigMoney to completion; `ended` event received |
| `decision.spec.ts` | Decision modal opens when `pending_decision.player_idx === myIdx`; submitting answer closes modal |

PixiJS canvas internals are verified via Playwright's `toHaveScreenshot` visual snapshots rather than DOM assertions.

---

## Section 7 — Visual design

Follows the existing HTML mockups in `docs/mockups/`:

- **Color palette:** Solarized Light (base3 `#fdf6e3` background, base02 `#073642` text, yellow `#b58900` accents)
- **Typography:** Georgia serif for headers/labels, SFMono monospace for counts/ids
- **Card states:** default, playable (gold border glow), selected (blue border + lift), hover (shadow + translate-y)
- **Layout:** opponent bar top, supply grid center-left, PixiJS canvas center, log sidebar right, hand + controls bottom

Tailwind CSS for all React DOM layout; PixiJS handles card rendering inside the canvas regions.
