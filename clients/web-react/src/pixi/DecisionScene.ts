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
