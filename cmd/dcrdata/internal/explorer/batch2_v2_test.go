// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"testing"
	"time"

	"github.com/decred/dcrdata/v8/explorer/types"
)

func TestV2MempoolTemplateRenders(t *testing.T) {
	out := renderV2(t, "mempool", struct {
		*CommonPageData
		Mempool *types.MempoolInfo
	}{tdCommon(), &types.MempoolInfo{
		MempoolShort: types.MempoolShort{NumAll: 142, NumRegular: 80, NumVotes: 45, NumTickets: 12,
			NumRevokes: 5, TotalOut: 12345.6, FormattedTotalSize: "210 KiB"},
		Transactions: []types.MempoolTx{{TxID: "abc123def", Type: "Regular", VinCount: 1, VoutCount: 2,
			Size: 412, FeeRate: 0.0001, TotalOut: 105.5, Time: time.Now().Unix()}},
		Votes: []types.MempoolTx{{TxID: "vote111", Size: 300, TotalOut: 214.6, Time: time.Now().Unix()}},
	}})
	mustContain(t, out, "mempool", "class=\"sidebar\"", "Mempool", "Regular transactions",
		"/v2/tx/abc123def", "Votes", "data-age-target")
}

func TestV2TicketpoolTemplateRenders(t *testing.T) {
	out := renderV2(t, "ticketpool", tdCommon())
	mustContain(t, out, "ticketpool", "class=\"sidebar\"", "Ticket pool", "Pool Size",
		"/api/chart/ticket-pool-size", "/api/chart/ticket-pool-value")
}
