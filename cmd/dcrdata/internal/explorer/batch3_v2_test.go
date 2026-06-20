// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"testing"

	chainjson "github.com/decred/dcrd/rpc/jsonrpc/types/v4"
	"github.com/decred/dcrdata/gov/v6/agendas"
	"github.com/decred/dcrdata/v8/explorer/types"
)

func TestV2ParametersTemplateRenders(t *testing.T) {
	out := renderV2(t, "parameters", struct {
		*CommonPageData
		MaximumBlockSize     int64
		ActualTicketPoolSize int64
		AddressPrefix        []types.AddrPrefix
	}{tdCommon(), 393216, 40960, []types.AddrPrefix{{Name: "P2PKH", Prefix: "Ds", Description: "pay-to-pubkey-hash"}}})
	mustContain(t, out, "parameters", "class=\"sidebar\"", "Chain parameters", "Max Block Size",
		"Address prefixes", "P2PKH", "Ds")
}

func TestV2AgendasTemplateRenders(t *testing.T) {
	out := renderV2(t, "agendas", struct {
		*CommonPageData
		Agendas       []*agendas.AgendaTagged
		VotingSummary *agendas.VoteSummary
	}{tdCommon(), []*agendas.AgendaTagged{
		{ID: "treasury", Description: "Enable the decentralized treasury", QuorumProgress: 0.62, VoteVersion: 9},
	}, nil})
	mustContain(t, out, "agendas", "Consensus agendas", "/agenda/treasury", "treasury", "Quorum progress")
}

func TestV2AgendaTemplateRenders(t *testing.T) {
	out := renderV2(t, "agenda", struct {
		*CommonPageData
		Ai            *agendas.AgendaTagged
		QuorumVotes   uint32
		RuleChangeQ   uint32
		VotingStarted int64
		LockedIn      int64
		BlocksLeft    int64
		TimeRemaining string
		TotalVotes    uint32
	}{tdCommon(), &agendas.AgendaTagged{ID: "treasury", Description: "Enable the decentralized treasury",
		Choices: []chainjson.Choice{
			{ID: "yes", Description: "Yes", Count: 8200, Progress: 0.82},
			{ID: "no", Description: "No", Count: 1800, Progress: 0.18, IsNo: true},
		}}, 9120, 14716, 0, 0, 1872, "5d 4h", 11000})
	mustContain(t, out, "agenda", "Vote choices", "treasury", "Total Votes", "gauge-fill")
}
