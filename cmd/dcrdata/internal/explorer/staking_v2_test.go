// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"strings"
	"testing"

	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrdata/v8/explorer/types"
)

// TestV2StakingTemplateRenders renders the redesigned staking page through the
// real template machinery + funcMap with mock home data, verifying the ticket
// price hero, stake donut, reward tiles, pool gauge and price chart.
func TestV2StakingTemplateRenders(t *testing.T) {
	assets := NewAssetManager("../../public")
	fm := makeTemplateFuncMap(chaincfg.MainNetParams(), assets)
	tm := newTemplates("../../views_v2", false, []string{"chrome"}, fm)
	if err := tm.addTemplate("staking"); err != nil {
		t.Fatalf("addTemplate(staking): %v", err)
	}

	page := &stakingPage{
		CommonPageData: tdCommon(),
		Info: &types.HomeInfo{
			StakeDiff: 214.6, NextExpectedStakeDiff: 218.1, IdxBlockInWindow: 42,
			TicketReward: 1.05, ASR: 8.7, RewardPeriod: "29 days",
			PoolInfo: types.TicketPoolInfo{Size: 41234, Value: 8850000, Target: 40960, PercentTarget: 100.7},
		},
		StakePct: 47.3, WindowSize: 144,
	}

	out, err := tm.exec("staking", page)
	if err != nil {
		t.Fatalf("exec(staking): %v", err)
	}

	for _, want := range []string{
		"/css/v2/main.css?v=",       // pipeline
		"class=\"sidebar\"",         // shell
		"Ticket Price",              // hero
		"data-controller=\"count\"", // count-up
		"Staked Supply",             // stake donut
		"donut-wrap",                // donut
		"47.3%",                     // staked percent
		"Ticket Reward",             // reward tile
		"gauge-fill",                // pool gauge
		"/api/chart/ticket-price",   // price history chart
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered v2 staking page missing %q", want)
		}
	}
}
