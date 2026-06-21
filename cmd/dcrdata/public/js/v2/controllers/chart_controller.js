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
    max: Number // for donut
  }

  connect () {
    if (this.pointsValue) {
      this.draw(JSON.parse(this.pointsValue))
    } else if (this.hasUrlValue) {
      this.fetchAndDraw()
    }
  }

  async fetchAndDraw () {
    this.element.innerHTML = '<div class="chart-empty">loading…</div>'
    try {
      const resp = await fetch(this.urlValue, { headers: { Accept: 'application/json' } })
      if (!resp.ok) throw new Error(resp.status)
      const data = await resp.json()
      const ys = (this.ykeyValue && data[this.ykeyValue]) || []
      this.draw(ys.map(Number))
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
