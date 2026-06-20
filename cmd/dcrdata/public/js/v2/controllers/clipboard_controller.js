// Copy-to-clipboard, demonstrating an interactive Stimulus controller running
// straight from native ESM (no build). Reads the text from a data attribute on
// the clicked element and flashes a brief confirmation.
import { Controller } from '@hotwired/stimulus'

export default class extends Controller {
  copy (event) {
    const text = event.currentTarget.dataset.clipboardText
    if (!text) return
    navigator.clipboard?.writeText(text).then(() => {
      const el = event.currentTarget
      const prev = el.textContent
      el.textContent = 'copied ✓'
      setTimeout(() => { el.textContent = prev }, 1100)
    }).catch(() => {})
  }
}
