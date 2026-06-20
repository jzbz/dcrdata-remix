// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import "testing"

func TestV2RawTxRenders(t *testing.T) {
	out := renderV2(t, "rawtx", struct {
		*CommonPageData
	}{tdCommon()})
	mustContain(t, out, "rawtx",
		"class=\"sidebar\"",
		"data-controller=\"theme rawtx\"",
		"Decode / Broadcast",
		"data-rawtx-target=\"hex\"",       // textarea wired to controller
		"data-action=\"rawtx#decode\"",    // decode button
		"data-action=\"rawtx#broadcast\"", // broadcast button
		"data-rawtx-target=\"output\"",    // decoded output region
		"/js/v2/app.js?v=",                // native-ESM entry present
	)
}

func TestV2AttackCostRenders(t *testing.T) {
	out := renderV2(t, "attackcost", struct {
		*CommonPageData
		HashRate        float64
		Height          int64
		DCRPrice        float64
		TicketPrice     float64
		TicketPoolSize  int64
		TicketPoolValue float64
		CoinSupply      int64
	}{
		CommonPageData:  tdCommon(),
		HashRate:        58.2,
		Height:          905142,
		DCRPrice:        18.42,
		TicketPrice:     214.6,
		TicketPoolSize:  41000,
		TicketPoolValue: 8800000,
		CoinSupply:      1600000000000000,
	})
	mustContain(t, out, "attackcost",
		"class=\"sidebar\"",
		"data-controller=\"theme attackcost\"",
		"Majority Attack Cost",
		"data-attackcost-hashrate=\"58.2\"", // server params wired to controller
		"data-attackcost-coin-supply=\"1600000000000000\"",
		"data-attackcost-target=\"attackPercent\"", // the slider
		"data-attackcost-target=\"graph\"",         // chart container
		"data-attackcost-target=\"total\"",         // summary total
		"data-action=\"input->attackcost#updateSliderData\"",
	)
}
