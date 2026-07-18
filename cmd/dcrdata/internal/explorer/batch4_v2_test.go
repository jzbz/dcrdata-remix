// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"testing"
	"time"

	pitypes "github.com/decred/dcrdata/gov/v6/politeia/types"
	"github.com/decred/dcrdata/v8/db/dbtypes"
	ticketvotev1 "github.com/decred/politeia/politeiawww/api/ticketvote/v1"
)

func TestV2ProposalsTemplateRenders(t *testing.T) {
	out := renderV2(t, "proposals", struct {
		*CommonPageData
		Proposals   []*pitypes.ProposalRecord
		VotesStatus map[ticketvotev1.VoteStatusT]string
		Offset      int64
		Limit       int64
		TotalCount  int64
	}{tdCommon(), []*pitypes.ProposalRecord{
		{Name: "Fund the thing", Token: "abc1234def", Username: "alice", TotalVotes: 10000, CommentsCount: 42},
	}, map[ticketvotev1.VoteStatusT]string{}, 0, 20, 1})
	mustContain(t, out, "proposals", "class=\"sidebar\"", "Proposals", "/proposal/abc1234def", "Fund the thing", "alice")
}

func TestV2ProposalTemplateRenders(t *testing.T) {
	out := renderV2(t, "proposal", struct {
		*CommonPageData
		Data        *pitypes.ProposalRecord
		PoliteiaURL string
		ShortToken  string
		Metadata    *pitypes.ProposalMetadata
	}{tdCommon(), &pitypes.ProposalRecord{Name: "Fund the thing", Token: "abc1234567", Username: "alice", CommentsCount: 42},
		"https://proposals.decred.org", "abc1234",
		&pitypes.ProposalMetadata{Yes: 8200, No: 1800, Approval: 0.82, Rejection: 0.18, IsPassing: true,
			VoteCount: 10000, QuorumCount: 5000, QuorumAchieved: true, VoteStatusDesc: "Approved", ProposalStatusDesc: "Public"}})
	mustContain(t, out, "proposal", "Fund the thing", "Vote result", "votebar", "82.0%", "Approved", "View on Politeia")
}

func TestV2TreasuryTemplateRenders(t *testing.T) {
	out := renderV2(t, "treasury", struct {
		*CommonPageData
		Data  *TreasuryInfo
		Pages []pageNumber
	}{tdCommon(), &TreasuryInfo{
		Balance: &dbtypes.TreasuryBalance{Balance: 82000000000000, Added: 120000000000000,
			TBase: 480000000000000, Spent: 38000000000000},
		Transactions: []*dbtypes.TreasuryTx{
			{TxID: dbtypes.ChainHash{1, 2, 3}, Amount: 50000000000, BlockHeight: 905000, BlockTime: dbtypes.NewTimeDef(time.Now())},
			{TxID: dbtypes.ChainHash{4, 5, 6}, Amount: -25000000000, BlockHeight: 904000, BlockTime: dbtypes.NewTimeDef(time.Now())},
		},
	}, nil})
	mustContain(t, out, "treasury", "Treasury transactions", "/tx/", "amt-in", "amt-out", "credit", "debit",
		// Received must be Added + TBase (1.2M + 4.8M = 6M DCR), not TAdds
		// alone, comma-grouped with the two-decimal frac span.
		"Received", "6,000,000<span class=\"frac\">.00</span>")
}

// TestTreasuryTypeString verifies the pager's txntype query values round-trip
// through the parser, so paginated treasury views keep their filter.
func TestTreasuryTypeString(t *testing.T) {
	for _, s := range []string{"tspend", "tadd", "treasurybase", "all"} {
		if got := treasuryTypeString(parseTreasuryTransactionType(s)); got != s {
			t.Errorf("treasuryTypeString(parseTreasuryTransactionType(%q)) = %q", s, got)
		}
	}
}
