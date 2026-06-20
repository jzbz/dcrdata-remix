// Dark/light theme toggle. Persists the choice to localStorage and reflects it
// as data-theme on <html>. The initial theme is applied by an inline script in
// the <head> (before paint) to avoid a flash; this controller only handles the
// toggle button and keeps the glyph in sync.
import { Controller } from '@hotwired/stimulus'

export default class extends Controller {
  static targets = ['glyph']

  connect () {
    this.sync()
  }

  toggle () {
    const next = this.current() === 'dark' ? 'light' : 'dark'
    document.documentElement.dataset.theme = next
    try { localStorage.setItem('theme', next) } catch (_) { /* ignore */ }
    this.sync()
  }

  current () {
    return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light'
  }

  sync () {
    if (this.hasGlyphTarget) {
      this.glyphTarget.textContent = this.current() === 'dark' ? '☀' : '☾'
    }
  }
}
