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
