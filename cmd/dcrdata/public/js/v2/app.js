// dcrdata v2 front-end entry — native ES module, no bundler.
//
// The bare specifier "@hotwired/stimulus" is resolved by the <script
// type="importmap"> in the page <head> to the vendored ESM build. Controllers
// are registered explicitly here (replacing webpack's require.context magic).

import { Application } from '@hotwired/stimulus'
import ThemeController from './controllers/theme_controller.js'
import ClipboardController from './controllers/clipboard_controller.js'
import AgeController from './controllers/age_controller.js'
import ChartController from './controllers/chart_controller.js'
import ChartDetailController from './controllers/chart_detail_controller.js'
import CountController from './controllers/count_controller.js'
import RawtxController from './controllers/rawtx_controller.js'
import AttackcostController from './controllers/attackcost_controller.js'
import MarketController from './controllers/market_controller.js'

const app = Application.start()
app.register('theme', ThemeController)
app.register('clipboard', ClipboardController)
app.register('age', AgeController)
app.register('chart', ChartController)
app.register('chart-detail', ChartDetailController)
app.register('count', CountController)
app.register('rawtx', RawtxController)
app.register('attackcost', AttackcostController)
app.register('market', MarketController)

window.dcrdata = { app }
