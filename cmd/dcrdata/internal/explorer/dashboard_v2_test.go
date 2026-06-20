// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"strings"
	"testing"
	"time"

	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrdata/v8/explorer/types"
)

// TestV2DashboardTemplateRenders renders the redesigned dashboard through the
// real template machinery + funcMap with mock home/chain data, verifying the
// chain strip, stat tiles, charts/donuts and latest-blocks table without a DB.
func TestV2DashboardTemplateRenders(t *testing.T) {
	assets := NewAssetManager("../../public")
	fm := makeTemplateFuncMap(chaincfg.MainNetParams(), assets)
	tm := newTemplates("../../views_v2", false, []string{"chrome"}, fm)
	if err := tm.addTemplate("dashboard"); err != nil {
		t.Fatalf("addTemplate(dashboard): %v", err)
	}

	page := &dashboardPage{
		CommonPageData: tdCommon(),
		Info: &types.HomeInfo{
			Difficulty: 4.21e11, HashRate: 421.6, StakeDiff: 214.6, CoinSupply: 1689000000000000,
		},
		Blocks: []*types.BlockBasic{
			{Height: 905142, Transactions: 6, Voters: 5, Total: 12481.06,
				FormattedBytes: "23.4 KiB", BlockTime: types.NewTimeDef(time.Now())},
		},
		BestHeight: 905142, SupplyDCR: 16890000, SupplyPct: 80.4, StakePct: 47.3, TreasuryDCR: 820000,
	}

	out, err := tm.exec("dashboard", page)
	if err != nil {
		t.Fatalf("exec(dashboard): %v", err)
	}

	for _, want := range []string{
		"/css/v2/main.css?v=",       // pipeline
		"class=\"sidebar\"",         // shell
		"Network Overview",          // page title
		"chain-strip",               // chain header strip
		"905,142",                   // best height (count-up target rendered)
		"data-controller=\"count\"", // count-up motion
		"data-controller=\"chart\"", // charts/sparklines/donuts
		"/api/chart/coin-supply",    // hero chart fetches real API
		"donut-wrap",                // supply/stake donuts
		"47.3%",                     // staked percent
		"Latest blocks",             // latest blocks section
		"/block/905142",             // block link
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered v2 dashboard missing %q", want)
		}
	}
}
