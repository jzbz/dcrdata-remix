// charts.js — minimal inline-SVG chart toolkit (no chart library), matching the
// redesign's type and color system. Builds SVG markup strings: smooth area/line
// charts (Catmull-Rom spline -> cubic bezier, vertical gradient fill, a draw-in
// stroke animation), sparklines, and donuts. After inserting the markup, call
// drawIn() to activate the line animation.

let uid = 0

const clamp = (v, lo, hi) => (v < lo ? lo : v > hi ? hi : v)

// smoothPath builds a cubic-bezier "d" through points using Catmull-Rom control
// points, giving organic curves without a charting dependency. The control
// points are clamped to each segment's bounding box so the spline never
// overshoots its endpoints; without this, spiky series sprout phantom peaks and
// dip below the baseline, which makes the area fill self-intersect into visible
// gaps.
function smoothPath (pts) {
  if (!pts.length) return ''
  if (pts.length === 1) return `M${pts[0].x},${pts[0].y}`
  let d = `M${pts[0].x.toFixed(2)},${pts[0].y.toFixed(2)}`
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[i - 1] || pts[i]
    const p1 = pts[i]
    const p2 = pts[i + 1]
    const p3 = pts[i + 2] || p2
    const loY = Math.min(p1.y, p2.y)
    const hiY = Math.max(p1.y, p2.y)
    const c1x = clamp(p1.x + (p2.x - p0.x) / 6, p1.x, p2.x)
    const c1y = clamp(p1.y + (p2.y - p0.y) / 6, loY, hiY)
    const c2x = clamp(p2.x - (p3.x - p1.x) / 6, p1.x, p2.x)
    const c2y = clamp(p2.y - (p3.y - p1.y) / 6, loY, hiY)
    d += `C${c1x.toFixed(2)},${c1y.toFixed(2)} ${c2x.toFixed(2)},${c2y.toFixed(2)} ${p2.x.toFixed(2)},${p2.y.toFixed(2)}`
  }
  return d
}

// downsample reduces a dense series to roughly `target` points by averaging
// equal-width buckets. The history charts return thousands of points for a few
// hundred pixels of width; drawing them all turns real volatility into
// unreadable spline noise, so we average down to about the pixel resolution for
// a clean trend line. Returns the input unchanged when it already fits.
function downsample (values, target) {
  const n = values.length
  if (!target || n <= target) return values
  const out = new Array(target)
  for (let i = 0; i < target; i++) {
    const start = Math.floor((i * n) / target)
    const end = Math.max(start + 1, Math.floor(((i + 1) * n) / target))
    let sum = 0
    for (let j = start; j < end; j++) sum += values[j]
    out[i] = sum / (end - start)
  }
  return out
}

function scale (values, w, h, pad) {
  const n = values.length
  let min = Infinity
  let max = -Infinity
  for (const v of values) { if (v < min) min = v; if (v > max) max = v }
  const range = (max - min) || 1
  return values.map((v, i) => ({
    x: n === 1 ? w / 2 : (i / (n - 1)) * w,
    y: h - pad - ((v - min) / range) * (h - pad * 2)
  }))
}

// areaChart returns an SVG string: a gradient-filled smooth area under a line.
// Call drawIn() on the container afterward to animate the line (see main.css).
export function areaChart (values, opts = {}) {
  const { color = '#2ED6A1', height = 90, width = 320, pad = 6 } = opts
  if (!values || values.length < 2) return '<div class="chart-empty">no data</div>'
  const pts = scale(downsample(values, width), width, height, pad)
  const line = smoothPath(pts)
  const area = `${line} L${width.toFixed(2)},${height} L0,${height} Z`
  const id = `cg${++uid}`
  return `<svg class="chart" viewBox="0 0 ${width} ${height}" preserveAspectRatio="none" role="img" aria-hidden="true">
  <defs><linearGradient id="${id}" x1="0" x2="0" y1="0" y2="1">
    <stop offset="0" stop-color="${color}" stop-opacity="0.32"/>
    <stop offset="1" stop-color="${color}" stop-opacity="0"/>
  </linearGradient></defs>
  <path class="chart-fill" d="${area}" fill="url(#${id})"/>
  <path class="chart-line" d="${line}" fill="none" stroke="${color}" stroke-width="2" vector-effect="non-scaling-stroke"/>
</svg>`
}

// drawIn activates the line draw-in: it measures each chart line's length and
// exposes it as the --len custom property that the CSS animation keys off (see
// .chart-line in main.css). The line is stroked with vector-effect:
// non-scaling-stroke inside a viewBox stretched by preserveAspectRatio="none",
// so its dash pattern is measured in *screen pixels*, not SVG user units.
// getTotalLength() returns user units, which is wrong whenever the container's
// aspect ratio differs from the viewBox — the dash comes out too short and the
// line is only drawn partway ("broken"). We instead measure the real on-screen
// length by sampling the path and mapping each sample through its screen CTM.
// Call this after inserting the markup into a connected, laid-out node.
export function drawIn (root) {
  if (!root) return
  for (const line of root.querySelectorAll('.chart-line')) {
    line.style.setProperty('--len', screenLength(line).toFixed(2))
  }
}

// screenLength returns a path's rendered length in CSS pixels, accounting for
// any non-uniform viewBox scaling. Falls back to the user-space length when the
// element is not laid out (no screen CTM available).
function screenLength (path) {
  const total = path.getTotalLength()
  const ctm = path.getScreenCTM()
  if (!total || !ctm) return total || 0
  const steps = 160
  let len = 0
  let px = null
  let py = null
  for (let i = 0; i <= steps; i++) {
    const p = path.getPointAtLength((i / steps) * total)
    const sx = ctm.a * p.x + ctm.c * p.y + ctm.e
    const sy = ctm.b * p.x + ctm.d * p.y + ctm.f
    if (px !== null) len += Math.hypot(sx - px, sy - py)
    px = sx
    py = sy
  }
  return len
}

// sparkline is a compact area chart for stat tiles.
export function sparkline (values, opts = {}) {
  return areaChart(values, { height: 44, width: 160, pad: 3, ...opts })
}

// donut returns an SVG ring that sweeps from 0 to value/max (animated in CSS).
export function donut (value, max, opts = {}) {
  const { color = '#2ED6A1', track = 'rgba(255,255,255,0.08)', size = 132, stroke = 13 } = opts
  const r = (size - stroke) / 2
  const c = 2 * Math.PI * r
  const frac = Math.max(0, Math.min(1, max ? value / max : 0))
  const off = c * (1 - frac)
  const cx = size / 2
  return `<svg class="donut" viewBox="0 0 ${size} ${size}" role="img" aria-hidden="true">
  <circle cx="${cx}" cy="${cx}" r="${r.toFixed(2)}" fill="none" stroke="${track}" stroke-width="${stroke}"/>
  <circle class="donut-ring" cx="${cx}" cy="${cx}" r="${r.toFixed(2)}" fill="none" stroke="${color}" stroke-width="${stroke}"
    stroke-linecap="round" transform="rotate(-90 ${cx} ${cx})"
    stroke-dasharray="${c.toFixed(2)}" style="--dash:${c.toFixed(2)}; --off:${off.toFixed(2)}"/>
</svg>`
}
