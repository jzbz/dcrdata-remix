// chart_controller — renders an inline-SVG chart into its element using the
// charts.js toolkit (no chart library). Data is either provided inline
// (points-value, handy for sparklines/previews) or fetched from dcrdata's chart
// API (url-value), pulling the y-series out by key (ykey-value).
import { Controller } from '@hotwired/stimulus'
import { areaChart, sparkline, donut, drawIn } from '../charts.js'

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
    // The draw-in dash length is measured in rendered pixels, so it must be
    // re-measured when the container resizes or the line re-breaks (see
    // drawIn/screenLength in charts.js). Donuts don't use the dash draw-in.
    if (this.kindValue !== 'donut') {
      this.resizeObserver = new ResizeObserver(() => {
        cancelAnimationFrame(this.redrawFrame)
        this.redrawFrame = requestAnimationFrame(() => drawIn(this.element))
      })
      this.resizeObserver.observe(this.element)
    }
  }

  disconnect () {
    cancelAnimationFrame(this.redrawFrame)
    clearTimeout(this.retryTimer)
    if (this.lazyObserver) {
      this.lazyObserver.disconnect()
      this.lazyObserver = null
    }
    if (this.resizeObserver) {
      this.resizeObserver.disconnect()
      this.resizeObserver = null
    }
  }

  async fetchAndDraw (attempt = 0) {
    if (attempt === 0) this.element.innerHTML = '<div class="chart-empty">loading…</div>'
    try {
      const resp = await fetch(this.urlValue, { headers: { Accept: 'application/json' } })
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
        let sum = 0
        ys = ys.map(v => (sum += v))
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
      drawIn(this.element)
    } else {
      this.element.innerHTML = areaChart(ys, opts)
      drawIn(this.element)
    }
  }
}
