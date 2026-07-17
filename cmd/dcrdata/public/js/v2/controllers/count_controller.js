// count_controller — animates a number from 0 up to its target on load (cubic
// ease-out), for the dashboard's "alive" counters. The element's server-rendered
// text is the final value, so without JS (or with reduced motion) it still shows
// the right number.
import { Controller } from '@hotwired/stimulus'

export default class extends Controller {
  static values = {
    to: Number,
    decimals: { type: Number, default: 0 },
    dur: { type: Number, default: 1400 }
  }

  connect () {
    if (matchMedia('(prefers-reduced-motion: reduce)').matches) return
    this.start = null
    this.rafId = requestAnimationFrame(this.step)
  }

  disconnect () {
    // Stop the animation chain: orphaned frames would keep writing into a
    // detached element, and a quick reconnect would run two competing chains.
    cancelAnimationFrame(this.rafId)
  }

  step = (now) => {
    if (this.start === null) this.start = now
    const t = Math.min(1, (now - this.start) / this.durValue)
    const eased = 1 - Math.pow(1 - t, 3)
    this.element.textContent = (this.toValue * eased).toLocaleString('en-US', {
      minimumFractionDigits: this.decimalsValue,
      maximumFractionDigits: this.decimalsValue
    })
    if (t < 1) this.rafId = requestAnimationFrame(this.step)
  }
}
