// chart_detail_controller — a large, interactive single-chart view built on the
// same dependency-free SVG approach as charts.js. It fetches the full time
// series from /api/chart/{type}?axis=time, renders it at real pixel size (so
// axis labels stay crisp — no preserveAspectRatio stretch), draws gridlines and
// date/value axes, and tracks the pointer to show a crosshair + tooltip with
// the exact date and value. Range presets (All / 1Y / 90D / 30D) act as a
// lightweight, drag-free zoom by slicing the series.
import { Controller } from '@hotwired/stimulus'
import { smoothPath, downsample } from '../charts.js'

const DAY = 86400
const RANGES = [
  { label: 'All', days: 0 },
  { label: '1Y', days: 365 },
  { label: '90D', days: 90 },
  { label: '30D', days: 30 }
]
const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

export default class extends Controller {
  static targets = ['plot', 'tip', 'range']
  static values = {
    type: String,
    ykey: String,
    ratio: String, // optional: divide ykey by this key (e.g. poolval / circulation)
    bin: { type: String, default: 'day' },
    color: { type: String, default: '#2ED6A1' },
    unit: { type: String, default: '' },
    scale: { type: Number, default: 1 }
  }

  connect () {
    this.rangeIdx = 0
    this.onMove = this.onMove.bind(this)
    this.onLeave = this.onLeave.bind(this)
    this.plotTarget.addEventListener('pointermove', this.onMove)
    this.plotTarget.addEventListener('pointerleave', this.onLeave)
    this.fetchData()
    this.ro = new ResizeObserver(() => {
      cancelAnimationFrame(this.raf)
      this.raf = requestAnimationFrame(() => this.render())
    })
    this.ro.observe(this.plotTarget)
  }

  disconnect () {
    cancelAnimationFrame(this.raf)
    if (this.ro) this.ro.disconnect()
    this.plotTarget.removeEventListener('pointermove', this.onMove)
    this.plotTarget.removeEventListener('pointerleave', this.onLeave)
  }

  async fetchData () {
    this.plotTarget.innerHTML = '<div class="chart-empty">loading…</div>'
    try {
      const url = `/api/chart/${this.typeValue}?bin=${this.binValue}&axis=time`
      const resp = await fetch(url, { headers: { Accept: 'application/json' } })
      if (!resp.ok) throw new Error(resp.status)
      const data = await resp.json()
      const raw = (data[this.ykeyValue] || []).map(Number)
      const ts = (data.t || []).map(Number)
      let vals
      if (this.ratioValue) {
        const den = (data[this.ratioValue] || []).map(Number)
        vals = raw.map((v, i) => (den[i] ? (v / den[i]) * this.scaleValue : 0))
      } else {
        vals = raw.map(v => v * this.scaleValue)
      }
      const n = Math.min(vals.length, ts.length)
      this.ts = ts.slice(0, n)
      this.ys = vals.slice(0, n)
      this.render()
    } catch (e) {
      this.plotTarget.innerHTML = '<div class="chart-empty">chart unavailable</div>'
    }
  }

  setRange (e) {
    this.rangeIdx = Number(e.currentTarget.dataset.range)
    this.rangeTargets.forEach((b, i) => b.classList.toggle('active', i === this.rangeIdx))
    this.render()
  }

  // sliceStart returns the first index of the active range's visible window.
  sliceStart () {
    const days = RANGES[this.rangeIdx].days
    if (!days || !this.ts || !this.ts.length) return 0
    const cutoff = this.ts[this.ts.length - 1] - days * DAY
    let i = this.ts.length - 1
    while (i > 0 && this.ts[i - 1] >= cutoff) i--
    return i
  }

  render () {
    // Show an explicit empty state rather than silently leaving the previous
    // plot (or the "loading…" placeholder) behind the newly-active button.
    if (!this.ts || this.ts.length < 2) {
      this.plotTarget.innerHTML = '<div class="chart-empty">no data</div>'
      return
    }
    const i0 = this.sliceStart()
    const ts = this.ts.slice(i0)
    const ys = this.ys.slice(i0)
    if (ts.length < 2) {
      this.plotTarget.innerHTML = '<div class="chart-empty">no data in this range</div>'
      return
    }

    const W = Math.max(320, Math.round(this.plotTarget.clientWidth || 800))
    const H = Math.max(220, Math.round(this.plotTarget.clientHeight || 360))
    const m = { l: 60, r: 14, t: 14, b: 26 }
    const iw = W - m.l - m.r
    const ih = H - m.t - m.b

    // y-range from the raw visible values (so hover dots can't escape the plot).
    let mn = Infinity
    let mx = -Infinity
    for (const v of ys) { if (v < mn) mn = v; if (v > mx) mx = v }
    if (!(mx > mn)) { mx = mn + 1 }
    const padY = (mx - mn) * 0.08
    const y0 = mn - padY
    const y1 = mx + padY
    const sx = i => m.l + (ts.length === 1 ? iw / 2 : (i / (ts.length - 1)) * iw)
    const sy = v => m.t + ih - ((v - y0) / (y1 - y0)) * ih

    // The drawn line is downsampled to ~pixel resolution for a clean curve;
    // hover still reads the raw series for exact values.
    const dys = downsample(ys, Math.max(2, Math.min(ys.length, iw)))
    const dpts = dys.map((v, i) => ({
      x: m.l + (dys.length === 1 ? iw / 2 : (i / (dys.length - 1)) * iw),
      y: sy(v)
    }))
    const line = smoothPath(dpts)
    const baseY = (m.t + ih).toFixed(2)
    const area = `${line} L${(m.l + iw).toFixed(2)},${baseY} L${m.l.toFixed(2)},${baseY} Z`

    let grid = ''
    let ylabels = ''
    for (const v of niceTicks(y0, y1, 5)) {
      const yy = sy(v).toFixed(1)
      grid += `<line class="cd-grid" x1="${m.l}" y1="${yy}" x2="${m.l + iw}" y2="${yy}"/>`
      ylabels += `<text class="cd-ylabel" x="${m.l - 8}" y="${yy}" text-anchor="end" dominant-baseline="middle">${humanize(v)}</text>`
    }
    let xlabels = ''
    const yearly = (ts[ts.length - 1] - ts[0]) > 2 * 365 * DAY
    for (let k = 0; k < 6; k++) {
      const i = Math.round((k / 5) * (ts.length - 1))
      xlabels += `<text class="cd-xlabel" x="${sx(i).toFixed(1)}" y="${(m.t + ih + 16).toFixed(1)}" text-anchor="middle">${fmtTick(ts[i], yearly)}</text>`
    }

    const id = `cd-${this.typeValue}`
    this.plotTarget.innerHTML = `<svg class="cd-svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}">
      <defs><linearGradient id="${id}" x1="0" x2="0" y1="0" y2="1">
        <stop offset="0" stop-color="${this.colorValue}" stop-opacity="0.26"/>
        <stop offset="1" stop-color="${this.colorValue}" stop-opacity="0"/>
      </linearGradient></defs>
      ${grid}
      <path d="${area}" fill="url(#${id})"/>
      <path d="${line}" fill="none" stroke="${this.colorValue}" stroke-width="1.6"/>
      ${ylabels}${xlabels}
      <g class="cd-cursor" style="display:none">
        <line class="cd-crosshair" x1="0" y1="${m.t}" x2="0" y2="${m.t + ih}"/>
        <circle class="cd-dot" r="3.5" cx="0" cy="0" fill="${this.colorValue}"/>
      </g>
    </svg>`

    this.view = { ts, ys, m, iw, sx, sy, W }
    this.cursor = this.plotTarget.querySelector('.cd-cursor')
    this.crosshair = this.plotTarget.querySelector('.cd-crosshair')
    this.dot = this.plotTarget.querySelector('.cd-dot')
  }

  onMove (e) {
    const v = this.view
    if (!v) return
    const rect = this.plotTarget.getBoundingClientRect()
    const mx = e.clientX - rect.left
    if (mx < v.m.l || mx > v.m.l + v.iw) { this.onLeave(); return }
    const n = v.ts.length
    let i = Math.round(((mx - v.m.l) / v.iw) * (n - 1))
    i = Math.max(0, Math.min(n - 1, i))
    const px = v.sx(i)
    const py = v.sy(v.ys[i])
    this.cursor.style.display = ''
    this.crosshair.setAttribute('x1', px.toFixed(1))
    this.crosshair.setAttribute('x2', px.toFixed(1))
    this.dot.setAttribute('cx', px.toFixed(1))
    this.dot.setAttribute('cy', py.toFixed(1))

    const tip = this.tipTarget
    const unit = this.unitValue ? ' ' + this.unitValue : ''
    tip.innerHTML = `<span class="cd-tip-date">${fmtDate(v.ts[i])}</span><span class="cd-tip-val">${humanize(v.ys[i])}${unit}</span>`
    tip.style.display = ''
    const tw = tip.offsetWidth
    let tx = px + 14
    if (tx + tw > v.W) tx = px - 14 - tw
    tx = Math.max(0, Math.min(tx, v.W - tw))
    tip.style.left = `${Math.round(this.plotTarget.offsetLeft + tx)}px`
    tip.style.top = `${Math.round(this.plotTarget.offsetTop + v.m.t + 6)}px`
  }

  onLeave () {
    if (this.cursor) this.cursor.style.display = 'none'
    if (this.hasTipTarget) this.tipTarget.style.display = 'none'
  }
}

// humanize formats a number compactly with a K/M/B/T suffix for axis labels and
// tooltips.
function humanize (v) {
  const a = Math.abs(v)
  for (const [n, s] of [[1e12, 'T'], [1e9, 'B'], [1e6, 'M'], [1e3, 'K']]) {
    if (a >= n) { const x = v / n; return trimZeros(Math.abs(x) >= 100 ? x.toFixed(0) : x.toFixed(1)) + s }
  }
  if (a >= 1) return trimZeros(v.toFixed(2))
  return v === 0 ? '0' : trimZeros(v.toFixed(4))
}

// trimZeros drops trailing fractional zeros: "34.00" -> "34", "1.50" -> "1.5".
function trimZeros (s) {
  return s.indexOf('.') < 0 ? s : s.replace(/\.?0+$/, '')
}

function fmtDate (s) {
  const d = new Date(s * 1000)
  return `${MONTHS[d.getUTCMonth()]} ${d.getUTCDate()}, ${d.getUTCFullYear()}`
}

function fmtTick (s, yearly) {
  const d = new Date(s * 1000)
  return yearly ? `${MONTHS[d.getUTCMonth()]} ${d.getUTCFullYear()}` : `${MONTHS[d.getUTCMonth()]} ${d.getUTCDate()}`
}

// niceTicks returns ~count axis ticks at human-friendly 1/2/5 * 10^n steps.
function niceTicks (lo, hi, count) {
  const range = hi - lo
  if (!(range > 0)) return [lo]
  const raw = range / count
  const mag = Math.pow(10, Math.floor(Math.log10(raw)))
  const norm = raw / mag
  const step = (norm <= 1 ? 1 : norm <= 2 ? 2 : norm <= 5 ? 5 : 10) * mag
  // Generate by index, not by accumulating v += step: if step ever falls
  // below the float ULP of v the accumulator stops advancing and the loop
  // never terminates. The cap bounds pathological ranges outright.
  const first = Math.ceil(lo / step) * step
  const n = Math.min(Math.floor((hi - first) / step + 1e-6) + 1, 20)
  const ticks = []
  for (let k = 0; k < n; k++) ticks.push(first + k * step)
  return ticks
}
