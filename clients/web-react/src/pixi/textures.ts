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
