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

	out, err := tm.exec("charts", struct{ *CommonPageData }{tdCommon()})
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

// TestChartDetailMetaUnits pins the detail-view scale/unit pairs that must
// match what the chart API serves: the "rate" series is Th/s while the rest of
// the UI shows Ph/s, and "work" is served pre-divided to exahash.
func TestChartDetailMetaUnits(t *testing.T) {
	if m := chartDetailMetas["hashrate"]; m.Scale != 1e-3 || m.Unit != "Ph/s" {
		t.Errorf("hashrate meta = scale %v unit %q, want scale 1e-3 unit Ph/s", m.Scale, m.Unit)
	}
	if m := chartDetailMetas["chainwork"]; m.Scale != 1 || m.Unit != "EH" {
		t.Errorf("chainwork meta = scale %v unit %q, want scale 1 unit EH", m.Scale, m.Unit)
	}
}
