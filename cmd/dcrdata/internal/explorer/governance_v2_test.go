// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"strings"
	"testing"

	"github.com/decred/dcrd/chaincfg/v3"
	pitypes "github.com/decred/dcrdata/gov/v6/politeia/types"
	"github.com/decred/dcrdata/v8/db/dbtypes"
)

// TestV2GovernanceTemplateRenders renders the redesigned governance page through
// the real template machinery + funcMap with mock treasury + proposal data,
// verifying the treasury tiles and proposal cards with vote bars.
func TestV2GovernanceTemplateRenders(t *testing.T) {
	assets := NewAssetManager("../../public")
	fm := makeTemplateFuncMap(chaincfg.MainNetParams(), assets)
	tm := newTemplates("../../views_v2", false, []string{"chrome"}, fm)
	if err := tm.addTemplate("governance"); err != nil {
		t.Fatalf("addTemplate(governance): %v", err)
	}

	page := &governancePage{
		CommonPageData: tdCommon(),
		Treasury:       &dbtypes.TreasuryBalance{Balance: 82000000000000, Added: 120000000000000, TBase: 480000000000000, Spent: 38000000000000},
		TreasuryDCR:    820000, ReceivedDCR: 6000000, SpentDCR: 380000,
		ProposalsTotal: 342, // more than shown: the show-all link must render
		Proposals: []govProposal{
			{ProposalRecord: &pitypes.ProposalRecord{Name: "Fund the thing", Token: "abc123"},
				Meta: &pitypes.ProposalMetadata{Yes: 8200, No: 1800, Approval: 82, Rejection: 18, IsPassing: true, VoteCount: 10000, VoteStatusDesc: "Approved"}},
			{ProposalRecord: &pitypes.ProposalRecord{Name: "Other idea", Token: "def456"},
				Meta: &pitypes.ProposalMetadata{Yes: 3000, No: 7000, Approval: 30, Rejection: 70, IsPassing: false, VoteCount: 10000, VoteStatusDesc: "Rejected"}},
		},
	}

	out, err := tm.exec("governance", page)
	if err != nil {
		t.Fatalf("exec(governance): %v", err)
	}

	for _, want := range []string{
		"/css/v2/main.css?v=",       // pipeline
		"class=\"sidebar\"",         // shell
		"Treasury Balance",          // treasury hero
		"Received",                  // received tile (Added + TBase)
		"6.00M",                     // rendered ReceivedDCR
		"data-controller=\"count\"", // count-up
		"Treasury over time",        // treasury chart placeholder
		"prop-card",                 // proposal cards
		"votebar",                   // vote bar
		"Fund the thing",            // proposal name
		"/proposal/abc123",          // proposal link
		"Approved",                  // vote status
		"?proposals=all",            // show-all link target
		"Show all 342 proposals",    // show-all link label
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered v2 governance page missing %q", want)
		}
	}
}
