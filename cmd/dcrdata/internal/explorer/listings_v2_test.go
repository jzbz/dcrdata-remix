// Copyright (c) 2024, The Decred developers
// See LICENSE for details.

package explorer

import (
	"strings"
	"testing"

	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrdata/v8/db/dbtypes"
)

func renderV2(t *testing.T, name string, data interface{}) string {
	t.Helper()
	assets := NewAssetManager("../../public")
	fm := makeTemplateFuncMap(chaincfg.MainNetParams(), assets)
	tm := newTemplates("../../views_v2", false, []string{"chrome"}, fm)
	if err := tm.addTemplate(name); err != nil {
		t.Fatalf("addTemplate(%s): %v", name, err)
	}
	out, err := tm.exec(name, data)
	if err != nil {
		t.Fatalf("exec(%s): %v", name, err)
	}
	return out
}

func mustContain(t *testing.T, out, name string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("rendered v2 %s missing %q", name, w)
		}
	}
}

func TestV2SideChainsTemplateRenders(t *testing.T) {
	out := renderV2(t, "sidechains", struct {
		*CommonPageData
		Data []*dbtypes.BlockStatus
	}{tdCommon(), []*dbtypes.BlockStatus{{Height: 905100, IsValid: true}}})
	mustContain(t, out, "sidechains", "class=\"sidebar\"", "Side chain blocks", "/v2/block/", "905100")
}

func TestV2DisapprovedTemplateRenders(t *testing.T) {
	out := renderV2(t, "disapproved", struct {
		*CommonPageData
		Data []*dbtypes.BlockStatus
	}{tdCommon(), []*dbtypes.BlockStatus{{Height: 905101, IsMainchain: true}}})
	mustContain(t, out, "disapproved", "PoS-invalidated", "⚠", "905101")
}

func TestV2WindowsTemplateRenders(t *testing.T) {
	out := renderV2(t, "windows", struct {
		*CommonPageData
		Data  []*dbtypes.BlocksGroupedInfo
		Pages pageNumbers
	}{tdCommon(), []*dbtypes.BlocksGroupedInfo{{IndexVal: 6284, EndBlock: 905100,
		TicketPrice: 21462000000, Difficulty: 4.2e11, Voters: 720, Transactions: 1820,
		FreshStake: 144, FormattedSize: "1.2 MiB", FormattedEndTime: "2026-06-20 12:00:00"}}, nil})
	mustContain(t, out, "windows", "Ticket price windows", "214.62", "6284")
}

func TestV2TimelistingTemplateRenders(t *testing.T) {
	out := renderV2(t, "timelisting", struct {
		*CommonPageData
		Data         []*dbtypes.BlocksGroupedInfo
		TimeGrouping string
		Pages        pageNumbers
	}{tdCommon(), []*dbtypes.BlocksGroupedInfo{{EndBlock: 905100, BlocksCount: 288,
		Transactions: 5400, Voters: 1440, FreshStake: 280, FormattedSize: "6.4 MiB",
		FormattedStartTime: "2026-06-20", FormattedEndTime: "2026-06-20 23:59"}}, "day", nil})
	mustContain(t, out, "timelisting", "Blocks by day", "905,100", "288")
}
