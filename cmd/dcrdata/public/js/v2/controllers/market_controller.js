// market_controller — pragmatic v2 market chart. Renders a close-price line
// from dcrdata's candlestick API using the inline-SVG charts toolkit (no chart
// library). The exchange comparison table, aggregate price and fiat indices are
// all server-rendered; this controller only drives the one history chart.
import { Controller } from '@hotwired/stimulus'
import { areaChart } from '../charts.js'

export default class extends Controller {
  static targets = ['exchange', 'bin', 'chart', 'meta']

  connect () {
    if (!this.hasExchangeTarget || this.exchangeTarget.options.length === 0) {
      this.empty('No candlestick data available.')
      return
    }
    this.syncBins()
    this.fetchChart()
  }

  currentOption () { return this.exchangeTarget.selectedOptions[0] }

  syncBins () {
    const opt = this.currentOption()
    // data-bins is rendered from a Go map, whose template iteration order is
    // lexicographic ("1d;1h;1mo"): sort by real duration and default to the
    // hourly view like the legacy market page, not whatever sorts first.
    const order = { '5m': 1, '30m': 2, '1h': 3, '1d': 4, '1mo': 5 }
    const bins = ((opt && opt.dataset.bins) || '').split(';').filter(Boolean)
      .sort((a, b) => (order[a] || 99) - (order[b] || 99))
    const active = bins.includes('1h') ? '1h' : bins[0]
    this.binTarget.innerHTML = bins.map(b =>
      `<button type="button" class="${b === active ? 'active' : ''}" data-bin="${b}" data-action="market#chooseBin">${b}</button>`
    ).join('')
  }

  currentBin () {
    const active = this.binTarget.querySelector('button.active') || this.binTarget.querySelector('button')
    return active ? active.dataset.bin : null
  }

  changeExchange () { this.syncBins(); this.fetchChart() }

  chooseBin (e) {
    this.binTarget.querySelectorAll('button').forEach(b => b.classList.toggle('active', b === e.currentTarget))
    this.fetchChart()
  }

  async fetchChart () {
    const opt = this.currentOption()
    const bin = this.currentBin()
    if (!opt || !bin) { this.empty('No candlestick data available.'); return }
    // Guard against out-of-order responses: switching exchange/bin while a
    // slow fetch is in flight must not let the stale response overwrite the
    // newer chart (the meta label is set synchronously, so they would disagree).
    const seq = (this.seq = (this.seq || 0) + 1)
    const token = opt.value
    const pair = opt.dataset.pair
    const url = `/api/chart/market/${token}/candlestick/${bin}?currencyPair=${encodeURIComponent(pair)}`
    this.chartTarget.innerHTML = '<div class="chart-empty">loading…</div>'
    if (this.hasMetaTarget) this.metaTarget.textContent = `${opt.textContent.trim()} · ${bin}`
    try {
      const resp = await fetch(url, { headers: { Accept: 'application/json' } })
      if (!resp.ok) throw new Error(resp.status)
      const data = await resp.json()
      if (seq !== this.seq) return // superseded by a newer selection
      const closes = (data.sticks || []).map(s => Number(s.close)).filter(n => !isNaN(n))
      if (closes.length < 2) { this.empty('Not enough data for this market.'); return }
      this.chartTarget.innerHTML = areaChart(closes, { color: '#5BA8FF', width: 820, height: 300 })
    } catch (e) {
      if (seq === this.seq) this.empty('Chart unavailable.')
    }
  }

  empty (msg) {
    // No chart target exists when exchange monitoring is disabled (the
    // template renders only an alert); don't throw on the missing target.
    if (this.hasChartTarget) this.chartTarget.innerHTML = `<div class="chart-empty">${msg}</div>`
  }
}
