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

const app = Application.start()
app.register('theme', ThemeController)
app.register('clipboard', ClipboardController)
app.register('age', AgeController)
app.register('chart', ChartController)

window.dcrdata = { app }
