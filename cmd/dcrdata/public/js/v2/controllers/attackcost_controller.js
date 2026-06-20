// attackcost_controller — majority-attack cost calculator. Ported from the
// legacy controller: the PoW/PoS cost math is preserved verbatim, but the
// heavy deps are gone — Dygraph is replaced by a small inline-SVG curve,
// TurboQuery by a minimal history.replaceState, and dompurify/event-bus are
// dropped (theming is pure CSS, and the only markup written is a constant
// miner link). Visibility toggles use the `hidden` attribute.
import { Controller } from '@hotwired/stimulus'

function digitformat (amount, decimalPlaces, noComma) {
  if (!amount) return 0
  if (noComma) return amount.toFixed(decimalPlaces)
  decimalPlaces = decimalPlaces || 0
  const result = parseFloat(amount).toLocaleString(undefined, { minimumFractionDigits: decimalPlaces, maximumFractionDigits: decimalPlaces }).replace(/\.0*$/, '')
  if (result.indexOf('.') > -1 && result.endsWith('0')) return removeTrailingZeros(result)
  return result
}

function removeTrailingZeros (value) {
  value = value.toString()
  if (value.indexOf('.') === -1) return value
  let cutFrom = value.length - 1
  do { if (value[cutFrom] === '0') cutFrom-- } while (value[cutFrom] === '0')
  if (value[cutFrom] === '.') cutFrom--
  return value.substr(0, cutFrom + 1)
}

// Hashpower multiplier required to match the honest network given stake
// fraction y under Decred's hybrid PoW/PoS. See:
// https://medium.com/decred/decreds-hybrid-protocol-a-superior-deterrent-to-majority-attacks-9421bf486292
function rateCalculation (y) {
  y = y || 0.99
  const x = 1 - y
  return ((6 * Math.pow(x, 5)) - (15 * Math.pow(x, 4)) + (10 * Math.pow(x, 3))) / ((6 * Math.pow(y, 5)) - (15 * Math.pow(y, 4)) + (10 * Math.pow(y, 3)))
}

const deviceList = {
  0: { hashrate: 34, units: 'Th/s', power: 1610, cost: 1282, name: 'DCR5', link: 'https://www.cryptocompare.com/mining/bitmain/antminer-dr5-blake256r14-34ths/' },
  1: { hashrate: 44, units: 'Th/s', power: 2200, cost: 4199, name: 'D1', link: 'https://www.cryptocompare.com/mining/crypto-drilling/microbt-whatsminer-d1-plus-psu-dcr-44ths/' }
}

const externalAttackType = 'external'
const internalAttackType = 'internal'

let height, dcrPrice, hashrate, tpSize, tpValue, tpPrice, coinSupply

export default class extends Controller {
  static get targets () {
    return [
      'actualHashRate', 'attackPercent', 'attackPeriod', 'blockHeight', 'countDevice', 'device',
      'deviceDesc', 'deviceName', 'devicePronoun', 'deviceSuffix', 'internalHash', 'kwhRate',
      'otherCosts', 'otherCostsValue', 'priceDCR', 'internalAttackText', 'targetHashRate',
      'externalAttackText', 'externalAttackPosText', 'additionalDcr', 'newTicketPoolValue', 'internalAttackPosText',
      'additionalHashRate', 'newHashRate', 'targetPos', 'targetPow', 'ticketPoolAttack',
      'ticketPoolSizeLabel', 'ticketPoolValue', 'ticketPrice', 'tickets', 'ticketSizeAttack', 'durationUnit', 'durationLongDesc',
      'total', 'totalDeviceCost', 'totalElectricity', 'totalKwh', 'totalPos', 'totalPow',
      'graph', 'projectedTicketPrice', 'projectedTicketPriceIncrease', 'projectedTicketPriceSign', 'attackType', 'attackPosPercentAmountLabel',
      'dcrPriceLabel', 'totalDCRPosLabel', 'projectedPriceDiv', 'attackNotPossibleWrapperDiv', 'coinSupply', 'totalAttackCostContainer'
    ]
  }

  connect () {
    this.settings = {}
    this.readQuery()

    height = parseInt(this.data.get('height'))
    hashrate = parseInt(this.data.get('hashrate'))
    dcrPrice = parseFloat(this.data.get('dcrprice'))
    tpPrice = parseFloat(this.data.get('ticketPrice'))
    tpValue = parseFloat(this.data.get('ticketPoolValue'))
    tpSize = parseInt(this.data.get('ticketPoolSize'))
    coinSupply = parseInt(this.data.get('coinSupply'))

    this.defaultSettings = {
      attack_time: 1, target_pow: 100, kwh_rate: 0.1, other_costs: 5,
      target_pos: 51, price: dcrPrice, device: 0, attack_type: externalAttackType
    }

    if (this.settings.attack_time) this.attackPeriodTarget.value = parseInt(this.settings.attack_time)
    if (this.settings.target_pow) this.targetPowTarget.value = parseFloat(this.settings.target_pow)
    if (this.settings.kwh_rate) this.kwhRateTarget.value = parseFloat(this.settings.kwh_rate)
    if (this.settings.other_costs) this.otherCostsTarget.value = parseFloat(this.settings.other_costs)
    if (this.settings.target_pos) this.setAllInputs(this.targetPosTargets, parseFloat(this.settings.target_pos))
    if (this.settings.price) this.priceDCRTarget.value = parseFloat(this.settings.price)
    if (this.settings.device) this.setDevice(this.settings.device)
    if (this.settings.attack_type) this.attackTypeTarget.value = this.settings.attack_type
    if (this.settings.target_pos) this.attackPercentTarget.value = parseFloat(this.targetPosTarget.value) / 100

    if (this.settings.attack_type !== internalAttackType) this.settings.attack_type = externalAttackType

    this.setDevicesDesc()
    this.plotGraph()
    this.updateSliderData()
  }

  // --- URL state (minimal replacement for TurboQuery) ---
  readQuery () {
    const p = new URLSearchParams(location.search)
    for (const k of ['attack_time', 'target_pow', 'kwh_rate', 'other_costs', 'target_pos', 'price', 'device', 'attack_type']) {
      if (p.has(k)) this.settings[k] = p.get(k)
    }
  }

  updateQueryString () {
    const params = new URLSearchParams()
    for (const k in this.settings) {
      const v = this.settings[k]
      if (!v || v.toString() === this.defaultSettings[k].toString()) continue
      params.set(k, v)
    }
    const qs = params.toString()
    history.replaceState(null, '', qs ? `${location.pathname}?${qs}` : location.pathname)
  }

  // --- inline-SVG attack curve (replaces Dygraph) ---
  plotGraph () {
    const W = 620, H = 200, padL = 44, padR = 12, padT = 14, padB = 26
    this.curve = []
    for (let i = 10; i <= 990; i += 5) {
      const y = i / 1000
      this.curve.push([y, Math.max(rateCalculation(y), 1e-4)])
    }
    const logs = this.curve.map(p => Math.log10(p[1]))
    const lo = Math.min(...logs), hi = Math.max(...logs)
    this.gx = (f) => padL + f * (W - padL - padR)
    this.gy = (mult) => {
      const t = (Math.log10(Math.max(mult, 1e-4)) - lo) / (hi - lo)
      return padT + (1 - t) * (H - padT - padB)
    }
    const line = this.curve.map((p, i) => `${i ? 'L' : 'M'}${this.gx(p[0]).toFixed(1)} ${this.gy(p[1]).toFixed(1)}`).join(' ')
    const x51 = this.gx(0.51)
    // y gridlines at multiplier = 0.1, 1, 10, 100
    const ticks = [100, 10, 1, 0.1].filter(m => m <= this.curve[0][1] && m >= this.curve[this.curve.length - 1][1])
    const grid = ticks.map(m => `<line x1="${padL}" y1="${this.gy(m).toFixed(1)}" x2="${W - padR}" y2="${this.gy(m).toFixed(1)}" class="ac-grid"/><text x="${padL - 6}" y="${(this.gy(m) + 4).toFixed(1)}" class="ac-axis" text-anchor="end">${m}x</text>`).join('')
    this.graphTarget.innerHTML = `<svg viewBox="0 0 ${W} ${H}" class="ac-svg" preserveAspectRatio="none" role="img" aria-label="Hashpower multiplier versus attacker stake">
      ${grid}
      <line x1="${x51.toFixed(1)}" y1="${padT}" x2="${x51.toFixed(1)}" y2="${H - padB}" class="ac-51"/>
      <text x="${(x51 + 4).toFixed(1)}" y="${padT + 10}" class="ac-axis ac-51-label">51%</text>
      <path d="${line}" class="ac-line"/>
      <circle r="5" class="ac-marker" data-attackcost-marker cx="${this.gx(0.51).toFixed(1)}" cy="${this.gy(rateCalculation(0.51)).toFixed(1)}"/>
      <text x="${padL}" y="${H - 8}" class="ac-axis" text-anchor="start">low stake</text>
      <text x="${W - padR}" y="${H - 8}" class="ac-axis" text-anchor="end">high stake →</text>
    </svg>`
    this.marker = this.graphTarget.querySelector('[data-attackcost-marker]')
  }

  setActivePoint () {
    if (!this.marker) return
    const f = Math.min(parseFloat(this.attackPercentTarget.value) || 0, 0.99)
    this.marker.setAttribute('cx', this.gx(f).toFixed(1))
    this.marker.setAttribute('cy', this.gy(rateCalculation(f)).toFixed(1))
  }

  // --- input handlers ---
  updateAttackTime () { this.settings.attack_time = this.attackPeriodTarget.value; this.updateSliderData() }

  updateTargetPow (e) {
    this.preserveTargetPow = true
    const targetPercentage = parseFloat(e.currentTarget.value) / 100
    // invert rateCalculation by scanning the precomputed curve for the nearest multiplier
    let best = 0.5
    let bestGap = Infinity
    for (const [y, mult] of this.curve) {
      const gap = Math.abs(mult - targetPercentage)
      if (gap < bestGap) { bestGap = gap; best = y }
    }
    this.attackPercentTarget.value = best
    this.updateSliderData()
  }

  chooseDevice () { this.settings.device = this.selectedDevice(); this.updateSliderData() }
  chooseAttackType () { this.settings.attack_type = this.selectedAttackType(); this.updateSliderData() }
  updateKwhRate () { this.settings.kwh_rate = this.kwhRateTarget.value; this.updateSliderData() }
  updateOtherCosts () { this.settings.other_costs = this.otherCostsTarget.value; this.updateSliderData() }

  updateTargetPos (e) {
    this.settings.target_pos = e.currentTarget.value
    this.preserveTargetPoS = true
    this.setAllInputs(this.targetPosTargets, e.currentTarget.value)
    this.attackPercentTarget.value = parseFloat(this.targetPosTarget.value) / 100
    this.updateSliderData()
  }

  updatePrice () { this.settings.price = this.priceDCRTarget.value; dcrPrice = this.priceDCRTarget.value; this.updateSliderData() }

  selectedDevice () { return this.deviceTarget.value }
  selectedAttackType () { return this.attackTypeTarget.value }

  setDevicesDesc () {
    this.deviceDescTargets.map((n) => {
      const info = deviceList[n.value]
      if (!info) return
      n.textContent = `${info.name} (${info.hashrate} ${info.units}, ${info.power} W, $${digitformat(info.cost)} ea.)`
    })
  }

  setDevice (selectedVal) { this.deviceTargets.map((n) => { n.selected = n.value === selectedVal }) }

  updateTargetHashRate () {
    const ticketPercentage = parseFloat(this.targetPosTarget.value)
    this.targetHashRate = hashrate * rateCalculation(ticketPercentage / 100)
    const powPercentage = 100 * this.targetHashRate / hashrate
    if (!this.preserveTargetPow) this.targetPowTarget.value = digitformat(powPercentage, 2, true)
    else this.preserveTargetPow = false

    this.setAllValues(this.internalHashTargets, digitformat(this.targetHashRate, 4) + ' Ph/s ')
    switch (this.settings.attack_type) {
      case externalAttackType:
        this.setAllValues(this.newHashRateTargets, digitformat(this.targetHashRate + hashrate, 4))
        this.setAllValues(this.additionalHashRateTargets, digitformat(this.targetHashRate, 4))
        this.projectedPriceDivTarget.hidden = false
        this.internalAttackTextTarget.hidden = true
        this.internalAttackPosTextTarget.hidden = true
        this.showAll(this.externalAttackTextTargets)
        this.externalAttackPosTextTarget.hidden = false
        break
      case internalAttackType:
      default:
        this.projectedPriceDivTarget.hidden = true
        this.hideAll(this.externalAttackTextTargets)
        this.externalAttackPosTextTarget.hidden = true
        this.internalAttackTextTarget.hidden = false
        this.internalAttackPosTextTarget.hidden = false
        break
    }
  }

  updateSliderData () {
    const val = Math.min(parseFloat(this.attackPercentTarget.value) || 0, 0.99)
    if (!this.preserveTargetPoS) this.setAllInputs(this.targetPosTargets, val * 100)
    else this.preserveTargetPoS = false

    this.updateTargetHashRate()
    this.setActivePoint()

    this.ticketsTarget.textContent = digitformat(val * tpSize) + ' tickets '
    switch (this.settings.attack_type) {
      case externalAttackType:
        this.hideAll(this.internalAttackPosTextTargets)
        this.showAll(this.externalAttackPosTextTargets)
        break
      case internalAttackType:
      default:
        this.hideAll(this.externalAttackPosTextTargets)
        this.showAll(this.internalAttackPosTextTargets)
    }
    this.calculate(true)
  }

  calculate (disableHashRateUpdate) {
    if (!disableHashRateUpdate) this.updateTargetHashRate()
    this.updateQueryString()

    const deviceInfo = deviceList[this.selectedDevice()]
    const deviceCount = Math.ceil((this.targetHashRate * 1000) / deviceInfo.hashrate)
    const totalDeviceCost = deviceCount * deviceInfo.cost
    const totalKwh = deviceCount * deviceInfo.power * parseFloat(this.attackPeriodTarget.value) / 1000
    const totalElectricity = totalKwh * parseFloat(this.kwhRateTarget.value)
    const extraCost = parseFloat(this.otherCostsTarget.value) / 100 * (totalDeviceCost + totalElectricity)
    const totalPow = extraCost + totalDeviceCost + totalElectricity

    let ticketAttackSize, DCRNeed
    if (this.settings.attack_type === externalAttackType) {
      DCRNeed = tpValue / (1 - parseFloat(this.targetPosTarget.value) / 100)
      this.setAllValues(this.newTicketPoolValueTargets, digitformat(DCRNeed, 2))
      this.setAllValues(this.additionalDcrTargets, digitformat(DCRNeed - tpValue, 2))
    } else {
      ticketAttackSize = (tpSize * parseFloat(this.targetPosTarget.value)) / 100
      DCRNeed = tpValue * (parseFloat(this.targetPosTarget.value) / 100)
      this.setAllValues(this.ticketPoolAttackTargets, digitformat(DCRNeed))
    }
    const projectedTicketPrice = DCRNeed / tpSize
    this.projectedTicketPriceIncreaseTarget.textContent = digitformat(100 * Math.abs(projectedTicketPrice - tpPrice) / tpPrice, 2)
    this.projectedTicketPriceSignTarget.textContent = projectedTicketPrice > tpPrice ? 'increase' : 'decrease'

    const totalDCRPos = this.settings.attack_type === externalAttackType ? DCRNeed - tpValue : ticketAttackSize * projectedTicketPrice
    const totalPos = totalDCRPos * dcrPrice
    const timeStr = this.attackPeriodTarget.value
    const hourStr = timeStr > 1 ? 'hours' : 'hour'
    const timeHourStr = timeStr + ' ' + hourStr
    const devicePronounStr = deviceCount > 1 ? 'them' : 'it'
    const deviceSuffixStr = deviceCount > 1 ? 's' : ''

    this.ticketPoolSizeLabelTarget.textContent = digitformat(tpSize, 2)
    this.setAllValues(this.actualHashRateTargets, digitformat(hashrate, 4))
    this.priceDCRTarget.value = digitformat(dcrPrice, 2)
    this.setAllInputs(this.targetPosTargets, digitformat(parseFloat(this.targetPosTarget.value), 2))
    this.ticketPriceTarget.textContent = digitformat(tpPrice, 4)
    this.setAllValues(this.targetHashRateTargets, digitformat(this.targetHashRate, 4))
    this.setAllValues(this.additionalHashRateTargets, digitformat(this.targetHashRate, 4))
    this.durationUnitTarget.textContent = hourStr
    this.setAllValues(this.durationLongDescTargets, timeHourStr)
    this.setAllValues(this.countDeviceTargets, digitformat(deviceCount))
    this.devicePronounTarget.textContent = devicePronounStr
    this.deviceSuffixTarget.textContent = deviceSuffixStr
    // deviceInfo.link/name are constants, so this fixed markup is safe.
    this.setAllValues(this.deviceNameTargets, `<a href="${deviceInfo.link}" target="_blank" rel="noopener">${deviceInfo.name}</a>${deviceSuffixStr}`)
    this.setAllValues(this.totalDeviceCostTargets, digitformat(totalDeviceCost))
    this.setAllValues(this.totalKwhTargets, digitformat(totalKwh, 2))
    this.setAllValues(this.totalElectricityTargets, digitformat(totalElectricity, 2))
    this.setAllValues(this.otherCostsValueTargets, digitformat(extraCost, 2))
    this.setAllValues(this.totalPowTargets, digitformat(totalPow, 2))
    if (this.hasTicketSizeAttackTarget) this.setAllValues(this.ticketSizeAttackTargets, digitformat(ticketAttackSize))
    this.setAllValues(this.totalPosTargets, digitformat(totalPos))
    this.setAllValues(this.ticketPoolValueTargets, digitformat(tpValue))
    this.blockHeightTarget.textContent = digitformat(height)
    this.totalTarget.textContent = digitformat(totalPow + totalPos, 2)
    this.projectedTicketPriceTarget.textContent = digitformat(projectedTicketPrice, 2)
    this.attackPosPercentAmountLabelTarget.textContent = digitformat(this.targetPosTarget.value, 2)
    this.setAllValues(this.totalDCRPosLabelTargets, digitformat(totalDCRPos, 2))
    this.setAllValues(this.dcrPriceLabelTargets, digitformat(dcrPrice, 2))
    this.showPosCostWarning(DCRNeed)
  }

  setAllValues (targets, data) { targets.forEach((n) => { n.innerHTML = data }) }
  setAllInputs (targets, data) { targets.forEach((n) => { n.value = data }) }
  hideAll (targets) { targets.forEach(el => { el.hidden = true }) }
  showAll (targets) { targets.forEach(el => { el.hidden = false }) }

  showPosCostWarning (DCRNeed) {
    const totalDCRInCirculation = coinSupply / 100000000
    if (DCRNeed > totalDCRInCirculation) {
      this.coinSupplyTarget.textContent = digitformat(totalDCRInCirculation, 2)
      this.totalAttackCostContainerTarget.classList.add('impossible')
      this.showAll(this.attackNotPossibleWrapperDivTargets)
    } else {
      this.totalAttackCostContainerTarget.classList.remove('impossible')
      this.hideAll(this.attackNotPossibleWrapperDivTargets)
    }
  }
}
