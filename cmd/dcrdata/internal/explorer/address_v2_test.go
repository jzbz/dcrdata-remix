// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"strings"
	"testing"
	"time"

	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/dcrutil/v4"
	"github.com/decred/dcrdata/v8/db/dbtypes"
)

// TestV2AddressTemplateRenders renders the redesigned address template through
// the real template machinery + funcMap with mock AddressInfo data, verifying
// the balance tiles, chart placeholder and signed-amount history without a DB.
func TestV2AddressTemplateRenders(t *testing.T) {
	assets := NewAssetManager("../../public")
	fm := makeTemplateFuncMap(chaincfg.MainNetParams(), assets)
	tm := newTemplates("../../views_v2", false, []string{"chrome"}, fm)
	if err := tm.addTemplate("address"); err != nil {
		t.Fatalf("addTemplate(address): %v", err)
	}

	h, _ := chainhash.NewHashFromStr("a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90")
	page := &addressPage{
		CommonPageData: tdCommon(),
		Data: &dbtypes.AddressInfo{
			Address:  "DsXyz1aBcDeFgHiJkLmNoPqRsTuVwXyZ12",
			TxnCount: 42,
			// Page-scoped sums over the displayed rows only. The stat tiles
			// must NOT use these; they must come from Balance below.
			AmountReceived: dcrutil.Amount(500 * 1e8),
			AmountSent:     dcrutil.Amount(120.5 * 1e8),
			AmountUnspent:  dcrutil.Amount(379.5 * 1e8),
			Balance: &dbtypes.AddressBalance{
				NumUnspent:   7,
				TotalUnspent: 600 * 1e8,
				TotalSpent:   900 * 1e8,
			},
			Transactions: []*dbtypes.AddressTx{
				{TxID: dbtypes.ChainHash(*h), TxType: "Regular", IsFunding: true, ReceivedTotal: 500.0,
					Confirmations: 37, Time: dbtypes.NewTimeDef(time.Now())},
				{TxID: dbtypes.ChainHash(*h), TxType: "Regular", IsFunding: false, SentTotal: 120.5,
					Confirmations: 12, Time: dbtypes.NewTimeDef(time.Now().Add(-time.Hour))},
			},
		},
	}

	out, err := tm.exec("address", page)
	if err != nil {
		t.Fatalf("exec(address): %v", err)
	}

	for _, want := range []string{
		"/css/v2/main.css?v=",                // pipeline
		"class=\"sidebar\"",                  // shell
		"DsXyz1aBcDeFgHiJkLmNoPqRsTuVwXyZ12", // address
		"Balance",                            // balance tile
		">600<",                              // balance = Balance.TotalUnspent (600 DCR)
		">1.50k<",                            // received = TotalSpent + TotalUnspent (1500 DCR)
		">900<",                              // sent = Balance.TotalSpent (900 DCR)
		"Transaction history",                // history section
		"data-controller=\"chart\"",          // balance-over-time chart
		"/amountflow/day",                    // chart data source
		"amt-in",                             // signed incoming amount
		"amt-out",                            // signed outgoing amount
		"/tx/",                               // tx links to v2
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered v2 address page missing %q", want)
		}
	}

	// The stat tiles must reflect the full-history balance, not the sums of
	// only the transactions on the current page (which can even go negative
	// on a spend-heavy page).
	for _, stale := range []string{">500<", ">120.5<", ">380<", ">379.5<"} {
		if strings.Contains(out, stale) {
			t.Errorf("rendered v2 address page shows page-scoped amount %q in a stat tile", stale)
		}
	}
}
