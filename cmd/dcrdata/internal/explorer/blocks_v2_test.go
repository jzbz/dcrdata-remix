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

// TestV2BlocksTemplateRenders loads the redesigned (npm-free) blocks template
// through the real template machinery + funcMap and renders it with mock data,
// verifying the whole pipeline (asset cache-busting, import map, controllers,
// real BlockBasic fields and helpers) without needing a database.
func TestV2BlocksTemplateRenders(t *testing.T) {
	assets := NewAssetManager("../../public")
	funcMap := makeTemplateFuncMap(chaincfg.MainNetParams(), assets)

	tmpls := newTemplates("../../views_v2", false, []string{"chrome"}, funcMap)
	if err := tmpls.addTemplate("blocks"); err != nil {
		t.Fatalf("addTemplate(blocks): %v", err)
	}

	now := time.Now()
	page := &blocksPage{
		Data: []*types.BlockBasic{
			{Height: 905142, Valid: true, Transactions: 6, Voters: 5, FreshStake: 3,
				Total: 12481.06, FormattedBytes: "23.4 KiB", Version: 11,
				BlockTime: types.NewTimeDef(now)},
			{Height: 905141, Valid: false, Transactions: 3, Voters: 5, FreshStake: 4,
				Revocations: 1, Total: 7512.93, FormattedBytes: "15.6 KiB", Version: 11,
				BlockTime: types.NewTimeDef(now.Add(-5 * time.Minute))},
		},
		BestBlock:    905142,
		OldestHeight: 2,
		Rows:         20,
		RowsCount:    2,
		WindowSize:   144,
		TimeGrouping: "Blocks",
		Pages: pageNumbers{
			{Active: true, Link: "/blocks?height=905142&rows=20", Str: "1"},
			{Active: false, Link: "/blocks?height=905122&rows=20", Str: "2"},
		},
	}

	out, err := tmpls.exec("blocks", page)
	if err != nil {
		t.Fatalf("exec(blocks): %v", err)
	}

	for _, want := range []string{
		"/css/v2/main.css?v=", // asset cache-buster ran (file hashed)
		"/js/v2/app.js?v=",    // native-ESM entry, versioned
		"@hotwired/stimulus",  // import map present
		"data-controller",     // Stimulus controllers wired
		"data-age-target",     // age controller hook on rows
		"905142",              // a real block height rendered
		"/block/905142",       // height links to the block page
		"⚠",                   // invalid-block marker (905141)
		"of 905,143 rows",     // rowcount via intComma/add helpers
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered v2 blocks page missing %q", want)
		}
	}
}
