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

// TestV2BlockTemplateRenders renders the redesigned block-detail template
// through the real template machinery + funcMap with mock BlockInfo data,
// verifying the shell, stat grid, details panel and tx tables without a DB.
func TestV2BlockTemplateRenders(t *testing.T) {
	assets := NewAssetManager("../../public")
	fm := makeTemplateFuncMap(chaincfg.MainNetParams(), assets)
	tm := newTemplates("../../views_v2", false, []string{"chrome"}, fm)
	if err := tm.addTemplate("block"); err != nil {
		t.Fatalf("addTemplate(block): %v", err)
	}

	mkTx := func(id, typ string) *types.TrimmedTxInfo {
		return &types.TrimmedTxInfo{
			TxBasic:   &types.TxBasic{TxID: id, Type: typ, FormattedSize: "412 B", Total: 105.5},
			Fees:      0.0003,
			VinCount:  1,
			VoutCount: 2,
		}
	}
	page := &blockPage{
		CommonPageData: tdCommon(),
		Data: &types.BlockInfo{
			BlockBasic: &types.BlockBasic{
				Height: 905142, Hash: "0000000000000000abcdef", Version: 11,
				Valid: true, MainChain: true, TxCount: 9, FormattedBytes: "23.4 KiB",
				Total: 12481.06, BlockTime: types.NewTimeDef(time.Now()),
			},
			Confirmations: 37, TotalSent: 12481.06, Difficulty: 4.21e11, SBits: 214.6,
			MiningFee: 0.12, PoolSize: 41234, Nonce: 123456, Bits: "1a01b2c3",
			StakeVersion: 9, FinalState: "abc123", MerkleRoot: "merkleroot00", StakeRoot: "stakeroot00",
			PreviousHash: "prevhash000", NextHash: "nexthash000",
			Tx:    []*types.TrimmedTxInfo{mkTx("tx0coinbase", "Coinbase"), mkTx("tx1regular", "Regular")},
			Votes: []*types.TrimmedTxInfo{mkTx("vote0aaa", "Vote")},
		},
	}

	out, err := tm.exec("block", page)
	if err != nil {
		t.Fatalf("exec(block): %v", err)
	}

	for _, want := range []string{
		"/css/v2/main.css?v=",  // pipeline/asset-buster
		"Block 905142",         // <title> via printf helper
		"class=\"sidebar\"",    // shell
		"/block/prevhash000",   // prev nav
		"/block/nexthash000",   // next nav
		"Block details",        // details panel
		"Merkle Root",          // kv summary
		"Confirmations",        // stat tile
		"Ticket Price",         // stat tile
		"Regular transactions", // tx section
		"/tx/tx1regular",       // tx links to v2 tx page
		"Votes",                // stake section
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered v2 block page missing %q", want)
		}
	}
}
