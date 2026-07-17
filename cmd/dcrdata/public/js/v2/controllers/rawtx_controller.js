// rawtx_controller — decode or broadcast a raw Decred transaction over the
// explorer's /ws JSON websocket. The protocol is a single message shape:
// send {event:"decodetx"|"sendtx", message:<hex>} and render the matching
// {event:"...Resp", message:<json|result|error>} reply. No bundler, no socket
// library — just the native WebSocket API.
import { Controller } from '@hotwired/stimulus'

export default class extends Controller {
  static targets = ['hex', 'output', 'header', 'status']

  connect () {
    this.open()
  }

  disconnect () {
    if (this.ws) {
      this.ws.removeEventListener('message', this.onMessage)
      this.ws.close()
      this.ws = null
    }
  }

  open () {
    // Tear down any previous socket first: the reconnect path in send() can
    // otherwise stack live sockets whose stale close/error handlers stomp the
    // status line after a new connection is already up.
    if (this.ws) {
      this.ws.removeEventListener('message', this.onMessage)
      this.ws.onclose = null
      this.ws.close()
      this.ws = null
    }
    const scheme = location.protocol === 'https:' ? 'wss' : 'ws'
    let ws
    try {
      ws = new WebSocket(`${scheme}://${location.host}/ws`)
    } catch (e) {
      this.setStatus('Unable to open a websocket connection.')
      return
    }
    this.ws = ws
    ws.addEventListener('message', this.onMessage)
    // Guard each status update: only the current socket may speak.
    ws.addEventListener('open', () => { if (this.ws === ws) this.setStatus('') })
    ws.addEventListener('close', () => { if (this.ws === ws) this.setStatus('Connection closed.') })
    ws.addEventListener('error', () => { if (this.ws === ws) this.setStatus('Connection error.') })
  }

  onMessage = (e) => {
    let data
    try { data = JSON.parse(e.data) } catch (_) { return }
    if (!data || !data.event) return
    if (data.event === 'decodetxResp') {
      this.headerTarget.textContent = 'Decoded transaction'
      this.show(data.message)
    } else if (data.event === 'sendtxResp') {
      this.headerTarget.textContent = 'Broadcast result'
      this.show(data.message)
    }
  }

  decode () { this.send('decodetx') }
  broadcast () { this.send('sendtx') }

  // Ctrl/Cmd+Enter in the textarea decodes, without inserting a newline.
  keydown (e) {
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault()
      this.decode()
    }
  }

  send (event) {
    const hex = this.hexTarget.value.trim()
    if (hex === '') { this.setStatus('Enter a transaction hex first.'); return }
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      this.setStatus('Websocket not connected — reconnecting, try again in a moment.')
      this.open()
      return
    }
    this.setStatus('')
    this.ws.send(JSON.stringify({ event, message: hex }))
  }

  show (text) {
    this.outputTarget.hidden = false
    this.outputTarget.textContent = text
  }

  setStatus (text) {
    if (!this.hasStatusTarget) return
    this.statusTarget.textContent = text
    this.statusTarget.hidden = text === ''
  }
}
