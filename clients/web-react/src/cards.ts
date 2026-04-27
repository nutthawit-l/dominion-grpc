export type CardInfo = {
  id: string
  name: string
  cost: number
  types: ('treasure' | 'victory' | 'curse' | 'action' | 'attack' | 'reaction')[]
  description: string
  isKingdom: boolean
}

export const CARD_INFO: Record<string, CardInfo> = {
  copper:       { id: 'copper',       name: 'Copper',       cost: 0, types: ['treasure'],             description: '+$1',                                                                                                                                                                    isKingdom: false },
  silver:       { id: 'silver',       name: 'Silver',       cost: 3, types: ['treasure'],             description: '+$2',                                                                                                                                                                    isKingdom: false },
  gold:         { id: 'gold',         name: 'Gold',         cost: 6, types: ['treasure'],             description: '+$3',                                                                                                                                                                    isKingdom: false },
  estate:       { id: 'estate',       name: 'Estate',       cost: 2, types: ['victory'],              description: '1 VP',                                                                                                                                                                   isKingdom: false },
  duchy:        { id: 'duchy',        name: 'Duchy',        cost: 5, types: ['victory'],              description: '3 VP',                                                                                                                                                                   isKingdom: false },
  province:     { id: 'province',     name: 'Province',     cost: 8, types: ['victory'],              description: '6 VP',                                                                                                                                                                   isKingdom: false },
  curse:        { id: 'curse',        name: 'Curse',        cost: 0, types: ['curse'],                description: '-1 VP',                                                                                                                                                                  isKingdom: false },
  artisan:      { id: 'artisan',      name: 'Artisan',      cost: 6, types: ['action'],               description: '+1 Card to hand. Gain a card to your hand costing up to $5. Put a card from your hand onto your deck.',                                                                 isKingdom: true },
  bandit:       { id: 'bandit',       name: 'Bandit',       cost: 5, types: ['action', 'attack'],     description: 'Gain a Gold. Each other player reveals the top 2 cards of their deck, trashes a revealed Treasure other than Copper, and discards the rest.',                            isKingdom: true },
  bureaucrat:   { id: 'bureaucrat',   name: 'Bureaucrat',   cost: 4, types: ['action', 'attack'],     description: 'Gain a Silver onto your deck. Each other player with a Victory card reveals one from their hand and puts it onto their deck.',                                            isKingdom: true },
  cellar:       { id: 'cellar',       name: 'Cellar',       cost: 2, types: ['action'],               description: '+1 Action. Discard any number of cards, then draw that many.',                                                                                                           isKingdom: true },
  chapel:       { id: 'chapel',       name: 'Chapel',       cost: 2, types: ['action'],               description: 'Trash up to 4 cards from your hand.',                                                                                                                                   isKingdom: true },
  council_room: { id: 'council_room', name: 'Council Room', cost: 5, types: ['action'],               description: '+4 Cards. +1 Buy. Each other player draws a card.',                                                                                                                     isKingdom: true },
  festival:     { id: 'festival',     name: 'Festival',     cost: 5, types: ['action'],               description: '+2 Actions. +1 Buy. +$2',                                                                                                                                               isKingdom: true },
  harbinger:    { id: 'harbinger',    name: 'Harbinger',    cost: 3, types: ['action'],               description: '+1 Card. +1 Action. You may put a card from your discard onto your deck.',                                                                                               isKingdom: true },
  laboratory:   { id: 'laboratory',   name: 'Laboratory',   cost: 5, types: ['action'],               description: '+2 Cards. +1 Action.',                                                                                                                                                  isKingdom: true },
  market:       { id: 'market',       name: 'Market',       cost: 5, types: ['action'],               description: '+1 Card. +1 Action. +1 Buy. +$1',                                                                                                                                      isKingdom: true },
  militia:      { id: 'militia',      name: 'Militia',      cost: 4, types: ['action', 'attack'],     description: '+$2. Each other player discards down to 3 cards in hand.',                                                                                                               isKingdom: true },
  mine:         { id: 'mine',         name: 'Mine',         cost: 5, types: ['action'],               description: 'Trash a Treasure from your hand. Gain a Treasure to your hand costing up to $3 more than it.',                                                                          isKingdom: true },
  moat:         { id: 'moat',         name: 'Moat',         cost: 2, types: ['action', 'reaction'],   description: '+2 Cards. When another player plays an Attack, you may reveal this to be unaffected.',                                                                                   isKingdom: true },
  moneylender:  { id: 'moneylender',  name: 'Moneylender',  cost: 4, types: ['action'],               description: 'You may trash a Copper from your hand for +$3.',                                                                                                                        isKingdom: true },
  poacher:      { id: 'poacher',      name: 'Poacher',      cost: 4, types: ['action'],               description: '+1 Card. +1 Action. +$1. Discard a card per empty Supply pile.',                                                                                                        isKingdom: true },
  remodel:      { id: 'remodel',      name: 'Remodel',      cost: 4, types: ['action'],               description: 'Trash a card from your hand. Gain a card costing up to $2 more than it.',                                                                                               isKingdom: true },
  smithy:       { id: 'smithy',       name: 'Smithy',       cost: 4, types: ['action'],               description: '+3 Cards.',                                                                                                                                                             isKingdom: true },
  vassal:       { id: 'vassal',       name: 'Vassal',       cost: 3, types: ['action'],               description: '+$2. Discard the top card of your deck. If it\'s an Action card, you may play it.',                                                                                     isKingdom: true },
  village:      { id: 'village',      name: 'Village',      cost: 3, types: ['action'],               description: '+1 Card. +2 Actions.',                                                                                                                                                  isKingdom: true },
  witch:        { id: 'witch',        name: 'Witch',        cost: 5, types: ['action', 'attack'],     description: '+2 Cards. Each other player gains a Curse.',                                                                                                                             isKingdom: true },
  workshop:     { id: 'workshop',     name: 'Workshop',     cost: 3, types: ['action'],               description: 'Gain a card costing up to $4.',                                                                                                                                         isKingdom: true },
}

export const KINGDOM_CARDS: CardInfo[] = Object.values(CARD_INFO).filter((c) => c.isKingdom)

export function getCard(id: string): CardInfo {
  return CARD_INFO[id] ?? { id, name: id, cost: 0, types: [], description: '', isKingdom: false }
}
