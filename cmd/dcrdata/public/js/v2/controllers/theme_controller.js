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
    // Dark is the default theme (the CSS :root is dark; light needs an explicit
    // data-theme="light"). The head boot script normally sets the attribute, but
    // default to dark if it is ever missing so the first toggle isn't a no-op.
    return document.documentElement.dataset.theme === 'light' ? 'light' : 'dark'
  }

  sync () {
    if (this.hasGlyphTarget) {
      this.glyphTarget.textContent = this.current() === 'dark' ? '☀' : '☾'
    }
  }
}
