// chart_controller — renders an inline-SVG chart into its element using the
// charts.js toolkit (no chart library). Data is either provided inline
// (points-value, handy for sparklines/previews) or fetched from dcrdata's chart
// API (url-value), pulling the y-series out by key (ykey-value).
import { Controller } from '@hotwired/stimulus'
import { areaChart, sparkline, donut } from '../charts.js'

export default class extends Controller {
  static values = {
    url: String,
    ykey: String,
    color: { type: String, default: '#2ED6A1' },
    kind: { type: String, default: 'area' }, // area | sparkline | donut
    points: String, // inline JSON array, overrides url
    max: Number, // for donut
    accumulate: Boolean, // running-sum the series before drawing (e.g. per-period flows -> balance)
    ratio: String // optional: divide ykey by this key elementwise (e.g. poolval / circulation)
  }

  connect () {
    if (this.pointsValue) {
      this.draw(JSON.parse(this.pointsValue))
    } else if (this.hasUrlValue) {
      // Lazy-load: only fetch once the chart nears the viewport. The charts
      // page renders many large series; fetching them all at once is slow, so
      // off-screen charts wait until scrolled toward.
      this.lazyObserver = new IntersectionObserver((entries) => {
        if (entries.some(e => e.isIntersecting)) {
          this.lazyObserver.disconnect()
          this.lazyObserver = null
          this.fetchAndDraw()
        }
      }, { rootMargin: '300px' })
      this.lazyObserver.observe(this.element)
    }
  }

  disconnect () {
    // An in-flight fetch that resolves 503 after disconnect must not re-arm
    // the retry timer cleared below; fetchAndDraw checks this flag.
    this.disconnected = true
    clearTimeout(this.retryTimer)
    if (this.lazyObserver) {
      this.lazyObserver.disconnect()
      this.lazyObserver = null
    }
  }

  async fetchAndDraw (attempt = 0) {
    if (attempt === 0) this.element.innerHTML = '<div class="chart-empty">loading…</div>'
    try {
      // Request a size-matched downsample so these preview charts don't pull
      // full history. Quantized to 100s for cache friendliness; the chart API
      // ignores ?max when it can't downsample. The interactive detail view uses
      // a different controller and keeps full resolution.
      const px = this.element.clientWidth || 300
      const maxPoints = Math.min(1000, Math.max(200, Math.round((px * 1.5) / 100) * 100))
      const url = new URL(this.urlValue, window.location.origin)
      url.searchParams.set('max', String(maxPoints))
      const resp = await fetch(url, { headers: { Accept: 'application/json' } })
      if (this.disconnected) return
      // Some series (e.g. the combined treasury chart) are built lazily and
      // return 503 until ready; poll briefly so the chart fills in on its own.
      if (resp.status === 503 && attempt < 12) {
        this.element.innerHTML = '<div class="chart-empty">preparing…</div>'
        this.retryTimer = setTimeout(() => this.fetchAndDraw(attempt + 1), 5000)
        return
      }
      if (!resp.ok) throw new Error(resp.status)
      const data = await resp.json()
      let ys = ((this.ykeyValue && data[this.ykeyValue]) || []).map(Number)
      if (this.ratioValue) {
        const den = (data[this.ratioValue] || []).map(Number)
        ys = ys.map((v, i) => (den[i] ? v / den[i] : 0))
      }
      if (this.accumulateValue) {
        // Running total seeded with a 0 baseline, so the series starts from
        // zero (e.g. an address balance before its first tx) and a single-period
        // history still has two points to draw instead of rendering "no data".
        let sum = 0
        ys = [0, ...ys.map(v => (sum += v))]
      }
      this.draw(ys)
    } catch (e) {
      this.element.innerHTML = '<div class="chart-empty">chart unavailable</div>'
    }
  }

  draw (ys) {
    const opts = { color: this.colorValue }
    if (this.kindValue === 'donut') {
      this.element.innerHTML = donut(ys[0] || 0, this.maxValue || 1, opts)
    } else if (this.kindValue === 'sparkline') {
      this.element.innerHTML = sparkline(ys, opts)
    } else {
      this.element.innerHTML = areaChart(ys, opts)
    }
  }
}
