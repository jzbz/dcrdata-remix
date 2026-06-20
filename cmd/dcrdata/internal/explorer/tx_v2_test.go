// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"strings"
	"testing"
	"time"

	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/dcrutil/v4"
	chainjson "github.com/decred/dcrd/rpc/jsonrpc/types/v4"
	"github.com/decred/dcrdata/v8/explorer/types"
)

// TestV2TxTemplateRenders renders the redesigned transaction template through
// the real template machinery + funcMap with mock TxInfo data, verifying the
// status header, summary tiles and the inputs/outputs flow without a DB.
func TestV2TxTemplateRenders(t *testing.T) {
	assets := NewAssetManager("../../public")
	fm := makeTemplateFuncMap(chaincfg.MainNetParams(), assets)
	tm := newTemplates("../../views_v2", false, []string{"chrome"}, fm)
	if err := tm.addTemplate("tx"); err != nil {
		t.Fatalf("addTemplate(tx): %v", err)
	}

	page := &txPage{
		Data: &types.TxInfo{
			TxBasic: &types.TxBasic{
				TxID: "9c0d1e2f3a4b5c6d7e8f90a1b2c3d4e5", Type: "Regular",
				FormattedSize: "412 B", Total: 642.18,
				Fee: dcrutil.Amount(12300), FeeRate: dcrutil.Amount(29800),
			},
			BlockHeight: 905142, BlockHash: "0000abcd", Confirmations: 37,
			Time: types.NewTimeDef(time.Now()),
			Vin: []types.Vin{
				{Vin: &chainjson.Vin{Txid: "prevtx0aaa"}, FormattedAmount: "642.20", DisplayText: "prevtx0aaa:1", TextIsHash: true},
			},
			Vout: []types.Vout{
				{Addresses: []string{"DsAddrOne111"}, FormattedAmount: "640.00", Type: "pubkeyhash", Spent: false},
				{Addresses: []string{"DsAddrTwo222"}, FormattedAmount: "2.18", Type: "pubkeyhash", Spent: true},
			},
		},
	}

	out, err := tm.exec("tx", page)
	if err != nil {
		t.Fatalf("exec(tx): %v", err)
	}

	for _, want := range []string{
		"/css/v2/main.css?v=",              // pipeline
		"class=\"sidebar\"",                // shell
		"9c0d1e2f3a4b5c6d7e8f90a1b2c3d4e5", // txid
		"Confirmed",                        // status badge
		"io-flow",                          // inputs/outputs flow
		"/v2/tx/prevtx0aaa",                // input links to v2 tx
		"/v2/address/DsAddrOne111",         // output links to v2 address
		"/v2/block/0000abcd",               // block link
		"2 outputs",                        // output count
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered v2 tx page missing %q", want)
		}
	}
}
