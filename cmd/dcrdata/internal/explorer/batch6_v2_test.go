// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"testing"
	"time"

	chainjson "github.com/decred/dcrd/rpc/jsonrpc/types/v4"
	exchanges "github.com/decred/dcrdata/exchanges/v3"
	"github.com/decred/dcrdata/v8/explorer/types"
)

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

func tvtx(id string, total float64, valid bool) *types.TrimmedTxInfo {
	return &types.TrimmedTxInfo{TxBasic: &types.TxBasic{TxID: id, Total: total}, VoteValid: valid, VinCount: 1, VoutCount: 2}
}

func TestV2VisualBlocksRenders(t *testing.T) {
	now := time.Now()
	out := renderV2(t, "visualblocks", struct {
		*CommonPageData
		Info    *types.HomeInfo
		Mempool *types.TrimmedMempoolInfo
		Blocks  []*types.TrimmedBlockInfo
	}{
		CommonPageData: tdCommon(),
		Info:           &types.HomeInfo{},
		Mempool: &types.TrimmedMempoolInfo{
			Total: 421.5, Fees: 0.12, Time: now.Unix(),
			Subsidy:      types.BlockSubsidy{PoW: 800000000, PoS: 200000000, Dev: 100000000},
			Votes:        []*types.TrimmedTxInfo{tvtx("v1", 1.2, true), tvtx("v2", 1.2, false)},
			Tickets:      []*types.TrimmedTxInfo{tvtx("t1", 214.6, false)},
			Transactions: []*types.TrimmedTxInfo{tvtx("x1", 50.4, false)},
		},
		Blocks: []*types.TrimmedBlockInfo{
			{
				Height: 905142, Total: 12481.06, Fees: 0.34, Time: types.NewTimeDef(now),
				Subsidy:      &chainjson.GetBlockSubsidyResult{PoW: 800000000, PoS: 200000000, Developer: 100000000},
				Votes:        []*types.TrimmedTxInfo{tvtx("bv1", 1.2, true), tvtx("bv2", 1.2, true), tvtx("bv3", 1.2, true), tvtx("bv4", 1.2, true), tvtx("bv5", 1.2, true)},
				Tickets:      []*types.TrimmedTxInfo{tvtx("bt1", 214.6, false), tvtx("bt2", 214.6, false)},
				Revocations:  []*types.TrimmedTxInfo{tvtx("br1", 214.6, false)},
				Transactions: []*types.TrimmedTxInfo{tvtx("bx1", 50.4, false), tvtx("bx2", 9.1, false)},
			},
		},
	})
	mustContain(t, out, "visualblocks",
		"class=\"sidebar\"",
		"Visual Blocks",
		"vblock is-mempool",       // mempool card present
		"/mempool",                // mempool link
		"/block/905142",           // block links to v2 block page
		"seg pow",                 // reward track segments
		"cell vote-yes",           // valid vote cell
		"cell vote-no",            // invalid vote cell
		"cell ticket",             // ticket cell
		"cell rev",                // revocation cell
		"data-age-target=\"age\"", // age controller wired
	)
}

func tdXcState() *exchanges.ExchangeBotState {
	xs := func(price, vol, change float64) *exchanges.ExchangeState {
		return &exchanges.ExchangeState{BaseState: exchanges.BaseState{Price: price, Volume: vol, Change: change}}
	}
	return &exchanges.ExchangeBotState{
		Index: "USD", Price: 18.42, Volume: 42000, BtcPrice: 64000,
		DCRExchanges: map[string]map[exchanges.CurrencyPair]*exchanges.ExchangeState{
			exchanges.Binance:  {exchanges.CurrencyPairDCRUSDT: xs(18.40, 30000, 0.6)},
			exchanges.Coinbase: {exchanges.CurrencyPairDCRBTC: xs(0.00029, 12000, -0.3)},
		},
		FiatIndices: map[string]map[exchanges.CurrencyPair]*exchanges.ExchangeState{
			"bittrex": {
				exchanges.BTCIndex:  xs(64000, 0, 0),
				exchanges.USDTIndex: xs(1.0, 0, 0),
			},
		},
	}
}

func TestV2MarketRenders(t *testing.T) {
	out := renderV2(t, "market", struct {
		*CommonPageData
		XcState *exchanges.ExchangeBotState
	}{tdCommon(), tdXcState()})
	mustContain(t, out, "market",
		"class=\"sidebar\"",
		"data-controller=\"theme market\"",
		"Market Data",
		"1 DCR =",
		"$18.42",                          // aggregate price
		"data-market-target=\"exchange\"", // chart exchange picker
		"data-market-target=\"chart\"",    // chart container
		"Decred markets",                  // comparison table
		"Aggregate",                       // aggregate row
		"Bitcoin indices",                 // indices section
	)
}

func TestV2MarketDisabled(t *testing.T) {
	out := renderV2(t, "market", struct {
		*CommonPageData
		XcState *exchanges.ExchangeBotState
	}{tdCommon(), nil})
	mustContain(t, out, "market", "Exchange monitoring is disabled")
}
