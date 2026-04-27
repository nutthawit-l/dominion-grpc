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
