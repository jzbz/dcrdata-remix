// Copy-to-clipboard, demonstrating an interactive Stimulus controller running
// straight from native ESM (no build). Reads the text from a data attribute on
// the clicked element and flashes a brief confirmation.
import { Controller } from '@hotwired/stimulus'

export default class extends Controller {
  copy (event) {
    // Capture the element synchronously: currentTarget is nulled once event
    // dispatch completes, so reading it inside the promise callback throws
    // and the confirmation never shows.
    const el = event.currentTarget
    const text = el.dataset.clipboardText
    if (!text) return
    navigator.clipboard?.writeText(text).then(() => {
      const prev = el.textContent
      el.textContent = 'copied ✓'
      setTimeout(() => { el.textContent = prev }, 1100)
    }).catch(() => {})
  }
}
