// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"strings"
	"testing"

	"github.com/decred/dcrd/chaincfg/v3"
)

// TestV2ChartsTemplateRenders renders the redesigned charts page (a grid of
// chart-controller cards) through the real template machinery, verifying the
// shell and the chart-card wiring. The charts themselves render client-side.
func TestV2ChartsTemplateRenders(t *testing.T) {
	assets := NewAssetManager("../../public")
	fm := makeTemplateFuncMap(chaincfg.MainNetParams(), assets)
	tm := newTemplates("../../views_v2", false, []string{"chrome"}, fm)
	if err := tm.addTemplate("charts"); err != nil {
		t.Fatalf("addTemplate(charts): %v", err)
	}

	out, err := tm.exec("charts", struct{ *CommonPageData }{})
	if err != nil {
		t.Fatalf("exec(charts): %v", err)
	}

	for _, want := range []string{
		"/css/v2/main.css?v=",       // pipeline
		"class=\"sidebar\"",         // shell
		"chart-grid",                // chart grid
		"data-controller=\"chart\"", // chart controller wired
		"/api/chart/coin-supply",    // fetches real chart API
		"Hashrate",                  // a chart card
		"Privacy Participation",     // a chart card
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered v2 charts page missing %q", want)
		}
	}
}
