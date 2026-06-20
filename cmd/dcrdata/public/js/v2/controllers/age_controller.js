// Renders relative "age" strings (e.g. "3m 12s ago") from a UNIX timestamp in
// each element's data-age attribute, and refreshes them on an interval. This is
// the v2 equivalent of the legacy time_controller's age handling.
import { Controller } from '@hotwired/stimulus'

export default class extends Controller {
  static targets = ['age']

  connect () {
    this.render()
    this.timer = setInterval(() => this.render(), 1000)
  }

  disconnect () {
    clearInterval(this.timer)
  }

  render () {
    const now = Date.now() / 1000
    this.ageTargets.forEach((el) => {
      const ts = Number(el.dataset.age)
      if (!ts) return
      el.textContent = humanize(Math.max(0, Math.floor(now - ts)))
    })
  }
}

function humanize (secs) {
  if (secs < 60) return secs + 's ago'
  const m = Math.floor(secs / 60)
  if (m < 60) return m + 'm ' + (secs % 60) + 's ago'
  const h = Math.floor(m / 60)
  if (h < 24) return h + 'h ' + (m % 60) + 'm ago'
  const d = Math.floor(h / 24)
  return d + 'd ' + (h % 24) + 'h ago'
}
